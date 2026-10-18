package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the bank methods used by the module.
type BankKeeper interface {
	GetSupply(ctx context.Context, denom string) sdk.Coin
}
