package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) EndBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	if params.EpochLenBlocks == 0 {
		return nil
	}
	if !isEpochEnd(ctx.BlockHeight(), params.EpochLenBlocks) {
		return nil
	}
	if err := k.CheckMissedProofs(goCtx); err != nil {
		return err
	}
	return k.mintAuditBudget(ctx, params)
}
