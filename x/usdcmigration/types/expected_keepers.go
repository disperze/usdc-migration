package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
)

// BankKeeper defines the bank methods used by the module.
type BankKeeper interface {
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// ScopedKeeper is the transfer module's scoped capability keeper. It gives
// MsgReturnToNoble the channel capability needed to send on the transfer port.
type ScopedKeeper interface {
	GetCapability(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool)
}
