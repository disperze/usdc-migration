package keeper

import (
	"context"

	"cosmossdk.io/collections"
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
	for _, pr := range gs.PendingReturns {
		if err := k.PendingReturns.Set(ctx, collections.Join(pr.ChannelId, pr.Sequence), pr.Amount); err != nil {
			return err
		}
	}
	return nil
}

// ExportGenesis exports the module state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	gs := types.DefaultGenesis()
	gs.Params = params

	err = k.ReturnedTotals.Walk(ctx, nil, func(channelID string, amount math.Int) (bool, error) {
		gs.ReturnedTotals = append(gs.ReturnedTotals, types.ReturnedTotal{ChannelId: channelID, Amount: amount})
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	err = k.PendingReturns.Walk(ctx, nil, func(key collections.Pair[string, uint64], amount math.Int) (bool, error) {
		gs.PendingReturns = append(gs.PendingReturns, types.PendingReturn{ChannelId: key.K1(), Sequence: key.K2(), Amount: amount})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return gs, nil
}
