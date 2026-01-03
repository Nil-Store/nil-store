package keeper_test

import (
	"bytes"
	"fmt"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestUnifiedLiveness_EndBlock_Mode2AutoRepairAfterMissedEpochs(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	params := f.keeper.GetParams(sdkCtx)
	params.EvictAfterMissedEpochs = 1
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Register enough providers so that each slot can be assigned a distinct repair candidate.
	for i := 0; i < 25; i++ {
		addrBz := make([]byte, 20)
		copy(addrBz, fmt.Sprintf("prov_mode2_%02d", i))
		addr, _ := f.addressCodec.BytesToString(addrBz)
		_, err := msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
			Creator:      addr,
			Capabilities: "General",
			TotalStorage: 100000000000,
			Endpoints:    testProviderEndpoints,
		})
		require.NoError(t, err)
	}

	ownerBz := make([]byte, 20)
	copy(ownerBz, "owner_mode2________")
	owner, _ := f.addressCodec.BytesToString(ownerBz)
	resDeal, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             owner,
		DurationBlocks:      1000,
		ServiceHint:         "General:rs=8+4",
		InitialEscrowAmount: math.NewInt(100000000),
		MaxMonthlySpend:     math.NewInt(10000000),
	})
	require.NoError(t, err)

	_, err = msgServer.UpdateDealContent(f.ctx, &types.MsgUpdateDealContent{
		Creator: owner, DealId: resDeal.DealId, Cid: dummyManifestCid, Size_: 8 * 1024 * 1024, TotalMdus: 2, WitnessMdus: 0,
	})
	require.NoError(t, err)

	dealBefore, err := f.keeper.Deals.Get(f.ctx, resDeal.DealId)
	require.NoError(t, err)
	require.Equal(t, uint64(1), dealBefore.CurrentGen)
	require.Len(t, dealBefore.Mode2Slots, int(types.DealBaseReplication))

	assigned := make(map[string]struct{}, len(dealBefore.Mode2Slots))
	for _, slot := range dealBefore.Mode2Slots {
		require.NotNil(t, slot)
		assigned[slot.Provider] = struct{}{}
		require.Equal(t, types.SlotStatus_SLOT_STATUS_ACTIVE, slot.Status)
		require.Empty(t, slot.PendingProvider)
	}

	// Jump to the end of epoch 0 and run the end blocker.
	endCtx := sdkCtx.WithBlockHeight(int64(params.EpochLenBlocks)).WithHeaderHash(bytes.Repeat([]byte{0xCD}, 32))
	require.NoError(t, f.keeper.EndBlock(endCtx))

	dealAfter, err := f.keeper.Deals.Get(endCtx, resDeal.DealId)
	require.NoError(t, err)
	require.Len(t, dealAfter.Mode2Slots, int(types.DealBaseReplication))

	for _, slot := range dealAfter.Mode2Slots {
		require.NotNil(t, slot)
		require.Equal(t, types.SlotStatus_SLOT_STATUS_REPAIRING, slot.Status)
		require.NotEmpty(t, slot.PendingProvider)
		require.Equal(t, endCtx.BlockHeight(), slot.StatusSinceHeight)
		require.Equal(t, dealAfter.CurrentGen, slot.RepairTargetGen)

		require.NotEqual(t, slot.Provider, slot.PendingProvider)
		_, ok := assigned[slot.PendingProvider]
		require.False(t, ok, "pending provider must not already be assigned to the deal")
	}

	missed, err := f.keeper.MissedEpochsMode2.Get(endCtx, collections.Join(resDeal.DealId, uint64(0)))
	require.NoError(t, err)
	require.Equal(t, uint64(1), missed)
}
