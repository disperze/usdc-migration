package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
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

// ReturnCap implements types.QueryServer.
func (q queryServer) ReturnCap(ctx context.Context, req *types.QueryReturnCapRequest) (*types.QueryReturnCapResponse, error) {
	if req == nil || req.ChannelId == "" {
		return nil, status.Error(codes.InvalidArgument, "channel_id is required")
	}
	if !q.GetParams(ctx).IsNobleChannel(req.ChannelId) {
		return nil, status.Errorf(codes.NotFound, "%s is not a configured Noble channel", req.ChannelId)
	}
	info := q.Keeper.ReturnCap(ctx, req.ChannelId)
	return &types.QueryReturnCapResponse{
		Denom:         info.Denom,
		Supply:        info.Supply,
		ReturnedTotal: info.Returned,
		Cap:           info.Cap,
	}, nil
}
