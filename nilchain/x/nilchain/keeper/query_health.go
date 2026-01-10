package keeper

import (
	"context"
	"errors"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"nilchain/x/nilchain/types"
)

func (q queryServer) GetDealProviderHealth(goCtx context.Context, req *types.QueryGetDealProviderHealthRequest) (*types.QueryGetDealProviderHealthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state, err := q.k.DealProviderHealth.Get(ctx, collections.Join(req.DealId, provider))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryGetDealProviderHealthResponse{State: types.HealthState{}}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetDealProviderHealthResponse{State: state}, nil
}

func (q queryServer) GetDealSlotHealth(goCtx context.Context, req *types.QueryGetDealSlotHealthRequest) (*types.QueryGetDealSlotHealthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state, err := q.k.DealSlotHealth.Get(ctx, collections.Join(req.DealId, req.Slot))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryGetDealSlotHealthResponse{State: types.HealthState{}}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetDealSlotHealthResponse{State: state}, nil
}

func (q queryServer) ListDealProviderHealth(goCtx context.Context, req *types.QueryListDealProviderHealthRequest) (*types.QueryListDealProviderHealthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	entries, pageRes, err := query.CollectionPaginate(
		goCtx,
		q.k.DealProviderHealth,
		req.Pagination,
		func(key collections.Pair[uint64, string], value types.HealthState) (types.DealProviderHealthEntry, error) {
			return types.DealProviderHealthEntry{
				DealId:   key.K1(),
				Provider: key.K2(),
				State:    value,
			}, nil
		},
		query.WithCollectionPaginationPairPrefix[uint64, string](req.DealId),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListDealProviderHealthResponse{Entries: entries, Pagination: pageRes}, nil
}

func (q queryServer) ListDealSlotHealth(goCtx context.Context, req *types.QueryListDealSlotHealthRequest) (*types.QueryListDealSlotHealthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	entries, pageRes, err := query.CollectionPaginate(
		goCtx,
		q.k.DealSlotHealth,
		req.Pagination,
		func(key collections.Pair[uint64, uint32], value types.HealthState) (types.DealSlotHealthEntry, error) {
			return types.DealSlotHealthEntry{
				DealId: key.K1(),
				Slot:   key.K2(),
				State:  value,
			}, nil
		},
		query.WithCollectionPaginationPairPrefix[uint64, uint32](req.DealId),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListDealSlotHealthResponse{Entries: entries, Pagination: pageRes}, nil
}
