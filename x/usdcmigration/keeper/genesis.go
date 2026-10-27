package keeper

import (
	"context"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// InitGenesis initializes the module state from genesis.
func (k *Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	if err := k.Params.Set(ctx, gs.Params); err != nil {
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
func (k *Keeper) ExportGenesis(ctx context.Context) (_ *types.GenesisState, err error) {
	gs := types.DefaultGenesis()
	gs.Params = k.GetParams(ctx)

	iter, err := k.ReturnedTotals.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := iter.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	for ; iter.Valid(); iter.Next() {
		kv, err := iter.KeyValue()
		if err != nil {
			return nil, err
		}
		gs.ReturnedTotals = append(gs.ReturnedTotals, types.ReturnedTotal{ChannelId: kv.Key, Amount: kv.Value})
	}
	return gs, nil
}
