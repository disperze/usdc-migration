package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidParams    = errorsmod.Register(ModuleName, 2, "invalid params")
	ErrRecvBlocked      = errorsmod.Register(ModuleName, 3, "receiving the deprecated Noble denom is disabled")
	ErrSendBlocked      = errorsmod.Register(ModuleName, 4, "sending the deprecated Noble voucher is disabled")
	ErrUnknownChannel   = errorsmod.Register(ModuleName, 5, "channel is not a configured Noble channel")
	ErrInvalidReceiver  = errorsmod.Register(ModuleName, 6, "invalid Noble receiver address")
	ErrInvalidTimeout   = errorsmod.Register(ModuleName, 7, "invalid timeout")
	ErrInvalidAmount    = errorsmod.Register(ModuleName, 8, "invalid amount")
	ErrInvalidAuthority = errorsmod.Register(ModuleName, 9, "invalid authority")
	ErrInvalidState     = errorsmod.Register(ModuleName, 10, "invalid module state")
	ErrUnsupportedApp   = errorsmod.Register(ModuleName, 11, "wrapped app does not support the call")
)
