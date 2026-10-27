package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		ReturnedTotals: []ReturnedTotal{},
		PendingReturns: []PendingReturn{},
	}
}

// Validate performs basic genesis state validation. Pending returns must be
// covered by their channel's returned total, since an error ack or timeout
// releases the pending amount from it.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	totals := make(map[string]math.Int, len(gs.ReturnedTotals))
	for _, rt := range gs.ReturnedTotals {
		if err := validateChannelID(rt.ChannelId); err != nil {
			return err
		}
		if _, dup := totals[rt.ChannelId]; dup {
			return fmt.Errorf("duplicate returned total for %s", rt.ChannelId)
		}
		if rt.Amount.IsNil() || rt.Amount.IsNegative() {
			return fmt.Errorf("invalid returned total for %s", rt.ChannelId)
		}
		totals[rt.ChannelId] = rt.Amount
	}

	type packetID struct {
		channel  string
		sequence uint64
	}
	seen := make(map[packetID]struct{}, len(gs.PendingReturns))
	pending := make(map[string]math.Int)
	for _, pr := range gs.PendingReturns {
		if err := validateChannelID(pr.ChannelId); err != nil {
			return err
		}
		id := packetID{pr.ChannelId, pr.Sequence}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("duplicate pending return %s/%d", pr.ChannelId, pr.Sequence)
		}
		seen[id] = struct{}{}
		if pr.Amount.IsNil() || !pr.Amount.IsPositive() {
			return fmt.Errorf("invalid pending return amount for %s/%d", pr.ChannelId, pr.Sequence)
		}
		sum, ok := pending[pr.ChannelId]
		if !ok {
			sum = math.ZeroInt()
		}
		pending[pr.ChannelId] = sum.Add(pr.Amount)
	}
	for _, pr := range gs.PendingReturns {
		total, ok := totals[pr.ChannelId]
		if !ok || pending[pr.ChannelId].GT(total) {
			return fmt.Errorf("pending returns on %s exceed its returned total", pr.ChannelId)
		}
	}
	return nil
}
