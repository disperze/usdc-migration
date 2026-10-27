package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// Keeper holds the module state and wraps the ICS4 channel used by the
// transfer keeper. It must be handled as a pointer because the ICS4 wrapper
// may be wired after construction.
type Keeper struct {
	cdc       codec.BinaryCodec
	authority string

	bankKeeper  types.BankKeeper
	ics4Wrapper porttypes.ICS4Wrapper

	Schema         collections.Schema
	Params         collections.Item[types.Params]
	ReturnedTotals collections.Map[string, math.Int]
}

// NewKeeper creates a new usdcmigration Keeper. ics4Wrapper is the next ICS4
// wrapper in the send path (the one the transfer keeper used before this
// module was added, e.g. the PFM keeper or the channel keeper). authority is
// the gov module account address.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	ics4Wrapper porttypes.ICS4Wrapper,
	authority string,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid usdcmigration authority address: %w", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:            cdc,
		authority:      authority,
		bankKeeper:     bankKeeper,
		ics4Wrapper:    ics4Wrapper,
		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ReturnedTotals: collections.NewMap(sb, types.ReturnedTotalPrefix, "returned_totals", collections.StringKey, sdk.IntValue),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// WithICS4Wrapper sets the next ICS4 wrapper in the send path, for apps that
// build their ICS4 chain after the keepers. MsgReturnToNoble also sends
// through it.
func (k *Keeper) WithICS4Wrapper(w porttypes.ICS4Wrapper) {
	k.ics4Wrapper = w
}

// GetAuthority returns the module authority (gov) address.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// GetParams returns the module params, or the defaults if they were never set.
func (k *Keeper) GetParams(ctx context.Context) types.Params {
	p, err := k.Params.Get(ctx)
	if err != nil {
		return types.DefaultParams()
	}
	return p
}

// SetParams validates and stores the module params. Upgrade handlers should
// use it instead of writing Params directly.
func (k *Keeper) SetParams(ctx context.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return errorsmod.Wrap(types.ErrInvalidParams, err.Error())
	}
	return k.Params.Set(ctx, p)
}

// GetReturnedTotal returns the amount returned to Noble over channelID and not refunded.
func (k *Keeper) GetReturnedTotal(ctx context.Context, channelID string) math.Int {
	amt, err := k.ReturnedTotals.Get(ctx, channelID)
	if err != nil {
		return math.ZeroInt()
	}
	return amt
}

func (k *Keeper) addReturnedTotal(ctx context.Context, channelID string, amt math.Int) error {
	return k.ReturnedTotals.Set(ctx, channelID, k.GetReturnedTotal(ctx, channelID).Add(amt))
}

// subReturnedTotal decreases the returned total after a refund, flooring at zero.
func (k *Keeper) subReturnedTotal(ctx context.Context, channelID string, amt math.Int) error {
	total := k.GetReturnedTotal(ctx, channelID).Sub(amt)
	if total.IsNegative() {
		total = math.ZeroInt()
	}
	return k.ReturnedTotals.Set(ctx, channelID, total)
}

// ReturnCapInfo describes how much voucher can still be returned over a channel.
type ReturnCapInfo struct {
	Denom    string
	Supply   math.Int
	Returned math.Int
	Cap      math.Int
}

// ReturnCap computes how much voucher can still be returned over channelID:
// supply - returned total, floored at zero. Returns do not burn the voucher,
// so while inbound USDC and outbound vouchers are blocked, this equals Noble's
// escrow for the channel minus what has already been released.
func (k *Keeper) ReturnCap(ctx context.Context, channelID string) ReturnCapInfo {
	denom := types.VoucherDenom(channelID, k.GetParams(ctx).BaseDenom)
	supply := k.bankKeeper.GetSupply(ctx, denom).Amount
	returned := k.GetReturnedTotal(ctx, channelID)
	return ReturnCapInfo{
		Denom:    denom,
		Supply:   supply,
		Returned: returned,
		Cap:      math.MaxInt(supply.Sub(returned), math.ZeroInt()),
	}
}
