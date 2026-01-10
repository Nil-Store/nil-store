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

func (q queryServer) GetAuditDebt(goCtx context.Context, req *types.QueryGetAuditDebtRequest) (*types.QueryGetAuditDebtResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state, err := q.k.AuditDebtStates.Get(ctx, provider)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryGetAuditDebtResponse{State: types.AuditDebtState{}, Outstanding: 0}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetAuditDebtResponse{State: state, Outstanding: auditDebtOutstanding(state)}, nil
}

func (q queryServer) ListAuditDebt(goCtx context.Context, req *types.QueryListAuditDebtRequest) (*types.QueryListAuditDebtResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	entries, pageRes, err := query.CollectionPaginate(
		goCtx,
		q.k.AuditDebtStates,
		req.Pagination,
		func(key string, value types.AuditDebtState) (types.AuditDebtEntry, error) {
			return types.AuditDebtEntry{
				Provider: key,
				State:    value,
			}, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListAuditDebtResponse{Entries: entries, Pagination: pageRes}, nil
}
