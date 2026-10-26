package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidParams  = errorsmod.Register(ModuleName, 2, "invalid params")
	ErrRecvBlocked    = errorsmod.Register(ModuleName, 3, "receiving the deprecated Noble denom is disabled")
	ErrSendBlocked    = errorsmod.Register(ModuleName, 4, "sending the deprecated Noble voucher is disabled")
	ErrUnknownChannel = errorsmod.Register(ModuleName, 5, "channel is not a configured Noble channel")
	// Code 6 (ErrAuthorityBlocked) is retired: returns no longer move funds
	// through the authority account.
	// Code 7 (ErrExceedsReturnCap) is retired: an explicit return amount is
	// limited by Noble's escrow, not by the voucher supply.
	ErrNothingToReturn  = errorsmod.Register(ModuleName, 8, "no voucher left to return")
	ErrInvalidReceiver  = errorsmod.Register(ModuleName, 9, "invalid Noble receiver address")
	ErrInvalidTimeout   = errorsmod.Register(ModuleName, 10, "invalid timeout")
	ErrInvalidAmount    = errorsmod.Register(ModuleName, 11, "invalid amount")
	ErrInvalidAuthority = errorsmod.Register(ModuleName, 12, "invalid authority")
)
