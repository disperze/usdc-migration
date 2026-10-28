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

// VoucherDenom returns the ibc/HASH voucher denom of baseDenom received over
// channelID. It matches ParseDenomTrace(path).IBCDenom() (ibc-go v8) and
// ExtractDenomFromPath(path).IBCDenom() (ibc-go v10).
func VoucherDenom(channelID, baseDenom string) string {
	hash := sha256.Sum256([]byte(VoucherPath(channelID, baseDenom)))
	return "ibc/" + strings.ToUpper(hex.EncodeToString(hash[:]))
}
