package types

import "fmt"

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		ReturnedTotals: []ReturnedTotal{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(gs.ReturnedTotals))
	for _, rt := range gs.ReturnedTotals {
		if err := validateChannelID(rt.ChannelId); err != nil {
			return err
		}
		if _, dup := seen[rt.ChannelId]; dup {
			return fmt.Errorf("duplicate returned total for %s", rt.ChannelId)
		}
		seen[rt.ChannelId] = struct{}{}
		if rt.Amount.IsNil() || rt.Amount.IsNegative() {
			return fmt.Errorf("invalid returned total for %s", rt.ChannelId)
		}
	}
	return nil
}
