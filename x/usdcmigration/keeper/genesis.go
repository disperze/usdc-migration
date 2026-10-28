package keeper

import (
	"context"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

// InitGenesis initializes the module state from genesis.
func (k *Keeper) InitGenesis(ctx context.Context, gs types.GenesisState) error {
	return k.SetParams(ctx, gs.Params)
}

// ExportGenesis exports the module state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	return &types.GenesisState{Params: k.GetParams(ctx)}, nil
}
