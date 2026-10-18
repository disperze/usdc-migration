package keeper_test

import (
	"context"
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

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/keeper"
	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
)

type fakeBank struct {
	blocked bool
	supply  sdkmath.Int
}

func (b fakeBank) GetSupply(context.Context, string) sdk.Coin { return sdk.NewCoin("uusdc", b.supply) }
func (b fakeBank) GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin {
	return sdk.NewCoin("uusdc", sdkmath.ZeroInt())
}
func (b fakeBank) BlockedAddr(sdk.AccAddress) bool                    { return b.blocked }
func (b fakeBank) MintCoins(context.Context, string, sdk.Coins) error { return nil }
func (b fakeBank) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	return nil
}

type fakeTransfer struct{ called bool }

func (t *fakeTransfer) Transfer(context.Context, *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error) {
	t.called = true
	return &transfertypes.MsgTransferResponse{Sequence: 1}, nil
}

func setup(t *testing.T, bank fakeBank) (sdk.Context, *keeper.Keeper, *fakeTransfer) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("t")).Ctx
	cdc := moduletestutil.MakeTestEncodingConfig().Codec

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(key), bank, nil, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	tk := &fakeTransfer{}
	k.SetTransferKeeper(tk)

	p := types.DefaultParams()
	p.NobleChannels = []string{"channel-224"}
	require.NoError(t, k.Params.Set(ctx, p))
	return ctx, k, tk
}

func returnMsg(k *keeper.Keeper) *types.MsgReturnToNoble {
	return &types.MsgReturnToNoble{
		Authority: k.GetAuthority(),
		ChannelId: "channel-224",
		Receiver:  sdk.MustBech32ifyAddressBytes(types.NobleBech32Prefix, []byte("noble-multisig______")),
		Amount:    sdkmath.ZeroInt(),
	}
}

func TestReturnToNobleRejectsBlockedAuthority(t *testing.T) {
	ctx, k, tk := setup(t, fakeBank{blocked: true, supply: sdkmath.NewInt(100)})
	_, err := keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, returnMsg(k))
	require.ErrorIs(t, err, types.ErrAuthorityBlocked)
	require.False(t, tk.called)
}

func TestReturnToNobleNothingToReturn(t *testing.T) {
	ctx, k, tk := setup(t, fakeBank{supply: sdkmath.ZeroInt()})
	_, err := keeper.NewMsgServerImpl(k).ReturnToNoble(ctx, returnMsg(k))
	require.ErrorIs(t, err, types.ErrNothingToReturn)
	require.False(t, tk.called)
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
