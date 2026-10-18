package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name.
	ModuleName = "usdcmigration"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// TransferPortID is the ICS20 port the module watches.
	TransferPortID = "transfer"

	// DefaultTimeoutSeconds is used by MsgReturnToNoble when timeout_seconds is zero.
	DefaultTimeoutSeconds uint64 = 3600

	// MaxTimeoutSeconds caps MsgReturnToNoble.timeout_seconds (7 days).
	MaxTimeoutSeconds uint64 = 7 * 24 * 3600
)

// NobleBech32Prefix is the account prefix receivers of MsgReturnToNoble must
// use. It is a variable only so in-process IBC tests can use their own prefix.
var NobleBech32Prefix = "noble"

var (
	ParamsKey           = collections.NewPrefix(0)
	ReturnedTotalPrefix = collections.NewPrefix(1)
)
