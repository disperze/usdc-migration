package keeper

import (
	"context"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

type queryServer struct {
	*Keeper
}

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns the module's QueryServer implementation.
func NewQueryServerImpl(k *Keeper) types.QueryServer {
	return queryServer{Keeper: k}
}

// Params implements types.QueryServer.
func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}
