package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
)

// InitGenesis initializes the module state from genesis.
func (k *Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := gs.Validate(); err != nil {
		return err
	}
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}
	for _, rt := range gs.ReturnedTotals {
		if err := k.ReturnedTotals.Set(ctx, rt.ChannelId, rt.Amount); err != nil {
			return err
		}
	}
	return nil
}

// ExportGenesis exports the module state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	gs := types.DefaultGenesis()
	gs.Params = k.GetParams(ctx)

	err := k.ReturnedTotals.Walk(ctx, nil, func(channelID string, amount math.Int) (bool, error) {
		gs.ReturnedTotals = append(gs.ReturnedTotals, types.ReturnedTotal{ChannelId: channelID, Amount: amount})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return gs, nil
}
