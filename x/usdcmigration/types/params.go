package types

import (
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultBaseDenom is Noble's USDC denom.
const DefaultBaseDenom = "uusdc"

// DefaultParams returns inert params: no Noble channel is configured, so
// nothing is blocked until a chain sets noble_channels (normally in its
// upgrade handler).
func DefaultParams() Params {
	return Params{
		NobleChannels:    []string{},
		BaseDenom:        DefaultBaseDenom,
		RecvBlockEnabled: true,
		SendBlockEnabled: true,
	}
}

// Validate performs stateless validation of the params.
func (p Params) Validate() error {
	if err := sdk.ValidateDenom(p.BaseDenom); err != nil {
		return fmt.Errorf("invalid base_denom: %w", err)
	}
	seen := make(map[string]struct{}, len(p.NobleChannels))
	for _, ch := range p.NobleChannels {
		if err := validateChannelID(ch); err != nil {
			return err
		}
		if _, dup := seen[ch]; dup {
			return fmt.Errorf("duplicate noble channel %s", ch)
		}
		seen[ch] = struct{}{}
	}
	return nil
}

// IsNobleChannel reports whether channelID is one of the configured Noble channels.
func (p Params) IsNobleChannel(channelID string) bool {
	return slices.Contains(p.NobleChannels, channelID)
}

// ShouldBlockRecv reports whether an ICS20 packet with packetDenom arriving on
// destChannel must be rejected. packetDenom is the denom as written by the
// sender chain, so Noble's native USDC arrives as the unprefixed base denom.
func (p Params) ShouldBlockRecv(destChannel, packetDenom string) bool {
	return p.RecvBlockEnabled && packetDenom == p.BaseDenom && p.IsNobleChannel(destChannel)
}

// ShouldBlockSend reports whether an outgoing ICS20 packet with denomPath (full
// ICS20 path, as written in the packet) must be rejected: the voucher of
// base_denom received over a Noble channel may not leave the chain over any
// channel, whoever the sender is (the authority included). Only
// MsgReturnToNoble returns it to Noble.
func (p Params) ShouldBlockSend(denomPath string) bool {
	return p.SendBlockEnabled && slices.ContainsFunc(p.NobleChannels, func(ch string) bool {
		return denomPath == VoucherPath(ch, p.BaseDenom)
	})
}

// validateChannelID mirrors ibc-go's host.ChannelIdentifierValidator
// ("channel-<n>") without importing a specific ibc-go major version.
func validateChannelID(id string) error {
	const prefix = "channel-"
	if len(id) <= len(prefix) || id[:len(prefix)] != prefix {
		return fmt.Errorf("invalid noble channel %q: must be channel-<sequence>", id)
	}
	for _, c := range id[len(prefix):] {
		if c < '0' || c > '9' {
			return fmt.Errorf("invalid noble channel %q: must be channel-<sequence>", id)
		}
	}
	return nil
}
