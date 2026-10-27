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
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryParamsResponse{Params: params}, nil
}

// ReturnCap implements types.QueryServer.
func (q queryServer) ReturnCap(ctx context.Context, req *types.QueryReturnCapRequest) (*types.QueryReturnCapResponse, error) {
	if req == nil || req.ChannelId == "" {
		return nil, status.Error(codes.InvalidArgument, "channel_id is required")
	}
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !params.IsNobleChannel(req.ChannelId) {
		return nil, status.Errorf(codes.NotFound, "%s is not a configured Noble channel", req.ChannelId)
	}
	info, err := q.Keeper.ReturnCap(ctx, req.ChannelId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryReturnCapResponse{
		Denom:         info.Denom,
		Supply:        info.Supply,
		ReturnedTotal: info.Returned,
		Cap:           info.Cap,
	}, nil
}
