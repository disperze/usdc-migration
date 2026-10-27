package keeper_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/keeper"
	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

type fakeBank struct {
	supply sdkmath.Int
}

func (b fakeBank) GetSupply(context.Context, string) sdk.Coin { return sdk.NewCoin("uusdc", b.supply) }

// chanCap stands in for the transfer module's capability of channel-224.
var chanCap = &capabilitytypes.Capability{Index: 7}

// fakeScoped owns the transfer capability of channel-224 only.
type fakeScoped struct{}

func (fakeScoped) GetCapability(_ sdk.Context, name string) (*capabilitytypes.Capability, bool) {
	if name != host.ChannelCapabilityPath(types.TransferPortID, "channel-224") {
		return nil, false
	}
	return chanCap, true
}

// fakeICS4 records the packets handed to the next ICS4 wrapper.
type fakeICS4 struct {
	chanCap       *capabilitytypes.Capability
	port, channel string
	timeout       uint64
	data          []byte
	calls         int
}

func (w *fakeICS4) SendPacket(_ sdk.Context, cap *capabilitytypes.Capability, port, channel string, _ clienttypes.Height, timeout uint64, data []byte) (uint64, error) {
	w.chanCap, w.port, w.channel, w.timeout, w.data = cap, port, channel, timeout, data
	w.calls++
	return uint64(w.calls), nil
}

func (w *fakeICS4) WriteAcknowledgement(sdk.Context, *capabilitytypes.Capability, exported.PacketI, exported.Acknowledgement) error {
	return nil
}

func (w *fakeICS4) GetAppVersion(sdk.Context, string, string) (string, bool) {
	return transfertypes.Version, true
}

func setup(t *testing.T, bank fakeBank) (sdk.Context, *keeper.Keeper, *fakeICS4) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("t")).Ctx
	cdc := moduletestutil.MakeTestEncodingConfig().Codec

	ics4 := &fakeICS4{}
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(key), bank, ics4, fakeScoped{}, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	p := types.DefaultParams()
	p.NobleChannels = []string{"channel-224"}
	require.NoError(t, k.Params.Set(ctx, p))
	return ctx, k, ics4
}

func returnMsg(k *keeper.Keeper) *types.MsgReturnToNoble {
	return &types.MsgReturnToNoble{
		Authority: k.GetAuthority(),
		ChannelId: "channel-224",
		Receiver:  sdk.MustBech32ifyAddressBytes(types.NobleBech32Prefix, []byte("noble-multisig______")),
		Amount:    sdkmath.ZeroInt(),
	}
}

func TestReturnToNobleBuildsPacket(t *testing.T) {
	ctx, k, ics4 := setup(t, fakeBank{supply: sdkmath.NewInt(100)})
	msg := returnMsg(k)
	msg.Memo = "migration"

	res, err := keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(1), res.Sequence)
	require.Equal(t, sdkmath.NewInt(100), res.Amount)
	require.Equal(t, types.VoucherDenom("channel-224", "uusdc"), res.Denom)

	require.Equal(t, chanCap, ics4.chanCap)
	require.Equal(t, types.TransferPortID, ics4.port)
	require.Equal(t, "channel-224", ics4.channel)
	require.Equal(t, uint64(ctx.BlockTime().UnixNano())+types.DefaultTimeoutSeconds*1_000_000_000, ics4.timeout)

	var data transfertypes.FungibleTokenPacketData
	require.NoError(t, json.Unmarshal(ics4.data, &data))
	require.Equal(t, transfertypes.NewFungibleTokenPacketData(
		"transfer/channel-224/uusdc", "100", k.GetAuthority(), msg.Receiver, "migration",
	), data)

	require.Equal(t, sdkmath.NewInt(100), k.GetReturnedTotal(ctx, "channel-224"))
	_, err = keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, returnMsg(k))
	require.ErrorIs(t, err, types.ErrNothingToReturn)
	require.Equal(t, 1, ics4.calls)
}

func TestReturnToNobleRequiresChannelCapability(t *testing.T) {
	ctx, k, ics4 := setup(t, fakeBank{supply: sdkmath.NewInt(100)})
	p := k.GetParams(ctx)
	p.NobleChannels = append(p.NobleChannels, "channel-225")
	require.NoError(t, k.Params.Set(ctx, p))

	msg := returnMsg(k)
	msg.ChannelId = "channel-225"
	_, err := keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, msg)
	require.ErrorIs(t, err, channeltypes.ErrChannelCapabilityNotFound)
	require.Zero(t, ics4.calls)
	require.True(t, k.GetReturnedTotal(ctx, "channel-225").IsZero())
}

func TestReturnToNobleNothingToReturn(t *testing.T) {
	ctx, k, ics4 := setup(t, fakeBank{supply: sdkmath.ZeroInt()})
	_, err := keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, returnMsg(k))
	require.ErrorIs(t, err, types.ErrNothingToReturn)
	require.Zero(t, ics4.calls)
}

func TestSendPacketBlocksVoucherForAuthority(t *testing.T) {
	ctx, k, ics4 := setup(t, fakeBank{supply: sdkmath.NewInt(100)})
	data := transfertypes.NewFungibleTokenPacketData("transfer/channel-224/uusdc", "1", k.GetAuthority(), "noble1x", "")
	_, err := k.SendPacket(ctx, chanCap, types.TransferPortID, "channel-224", clienttypes.ZeroHeight(), 1, data.GetBytes())
	require.ErrorIs(t, err, types.ErrSendBlocked)
	require.Zero(t, ics4.calls)
}

func TestUpdateParams(t *testing.T) {
	ctx, k, _ := setup(t, fakeBank{supply: sdkmath.ZeroInt()})
	srv := keeper.NewMsgServerImpl(k)

	p := types.DefaultParams()
	p.NobleChannels = []string{"channel-1", "channel-2"}
	p.RecvBlockEnabled = false

	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{Authority: sdk.AccAddress([]byte("not-gov-____________")).String(), Params: p})
	require.ErrorIs(t, err, types.ErrInvalidAuthority)

	bad := p
	bad.NobleChannels = []string{"07-tendermint-0"}
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: bad})
	require.ErrorIs(t, err, types.ErrInvalidParams)

	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: p})
	require.NoError(t, err)
	require.Equal(t, p, k.GetParams(ctx))
}

func TestSetParamsValidates(t *testing.T) {
	ctx, k, _ := setup(t, fakeBank{supply: sdkmath.ZeroInt()})
	p := types.DefaultParams()
	p.NobleChannels = []string{"channel-1", "channel-1"}
	require.ErrorIs(t, k.SetParams(ctx, p), types.ErrInvalidParams)
}
