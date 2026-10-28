package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// Keeper holds the module state and wraps the ICS4 channel used by the
// transfer keeper. It must be handled as a pointer because the ICS4 wrapper
// may be wired after construction.
type Keeper struct {
	authority   string
	ics4Wrapper porttypes.ICS4Wrapper

	Schema collections.Schema
	Params collections.Item[types.Params]
}

// NewKeeper creates a new usdcmigration Keeper. ics4Wrapper is the next ICS4
// wrapper in the send path (the one the transfer keeper used before this
// module was added, e.g. the PFM keeper or the channel keeper). authority is
// the gov module account address.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ics4Wrapper porttypes.ICS4Wrapper,
	authority string,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid usdcmigration authority address: %w", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		authority:   authority,
		ics4Wrapper: ics4Wrapper,
		Params:      collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
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

func (k *Keeper) checkAuthority(addr string) error {
	if addr != k.authority {
		return errorsmod.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", k.authority, addr)
	}
	return nil
}

// GetParams returns the module params, or the defaults if they were never
// set. It panics on a corrupt store rather than fall back to the defaults,
// which block nothing.
func (k *Keeper) GetParams(ctx context.Context) types.Params {
	p, err := k.Params.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return types.DefaultParams()
	}
	if err != nil {
		panic(err)
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
