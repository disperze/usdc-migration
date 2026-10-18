package types

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VoucherPath returns the ICS20 denom path of a Noble native denom received
// over channelID, e.g. "transfer/channel-224/uusdc".
func VoucherPath(channelID, baseDenom string) string {
	return TransferPortID + "/" + channelID + "/" + baseDenom
}

// IBCDenom returns the ibc/HASH denom of an ICS20 denom path. It matches
// ParseDenomTrace(path).IBCDenom() (ibc-go v8) and
// ExtractDenomFromPath(path).IBCDenom() (ibc-go v10).
func IBCDenom(path string) string {
	hash := sha256.Sum256([]byte(path))
	return "ibc/" + strings.ToUpper(hex.EncodeToString(hash[:]))
}

// VoucherDenom returns the ibc/HASH voucher denom of baseDenom received over channelID.
func VoucherDenom(channelID, baseDenom string) string {
	return IBCDenom(VoucherPath(channelID, baseDenom))
}
