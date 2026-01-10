package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/types"
)

func TestMatchesMode1SyntheticChallenge(t *testing.T) {
	seed := [32]byte{}
	for i := range seed {
		seed[i] = byte(i)
	}
	in := quotaInputs{metaMdus: 2, userMdus: 5}
	provider := make([]byte, 20)
	for i := range provider {
		provider[i] = byte(0xA0 + i)
	}

	mduIndex, blobIndex := deriveMode1Challenge(seed, 42, 7, provider, 0, in)
	require.GreaterOrEqual(t, mduIndex, in.metaMdus)
	require.Less(t, blobIndex, uint32(types.BlobsPerMdu))
	require.True(t, matchesMode1SyntheticChallenge(seed, 42, 7, provider, in, 3, mduIndex, blobIndex))

	require.False(t, matchesMode1SyntheticChallenge(seed, 42, 7, provider, in, 3, in.metaMdus-1, 0))
}

func TestMatchesMode2SyntheticChallenge(t *testing.T) {
	seed := [32]byte{}
	for i := range seed {
		seed[i] = byte(0xFF - i)
	}
	in := quotaInputs{metaMdus: 3, userMdus: 9}
	stripe := stripeParams{mode: 2, rows: 4, leafCount: 12, slotCount: 3}

	mduIndex, blobIndex := deriveMode2Challenge(seed, 99, 3, 1, 0, in, stripe)
	require.GreaterOrEqual(t, mduIndex, in.metaMdus)
	require.Less(t, blobIndex, uint32(stripe.leafCount))
	require.True(t, matchesMode2SyntheticChallenge(seed, 99, 3, 1, in, stripe, 4, mduIndex, blobIndex))

	require.False(t, matchesMode2SyntheticChallenge(seed, 99, 3, 1, in, stripe, 4, in.metaMdus-1, 0))
}

func TestMode2SlotActiveForSynthetic(t *testing.T) {
	deal := types.Deal{
		RedundancyMode: 2,
		Mode2Slots: []*types.DealSlot{
			{Slot: 0, Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
			{Slot: 1, Status: types.SlotStatus_SLOT_STATUS_REPAIRING},
		},
	}

	require.True(t, mode2SlotActiveForSynthetic(deal, 0))
	require.False(t, mode2SlotActiveForSynthetic(deal, 1))
	require.False(t, mode2SlotActiveForSynthetic(deal, 2))

	legacy := types.Deal{RedundancyMode: 2}
	require.True(t, mode2SlotActiveForSynthetic(legacy, 0))
}

func TestMode2SlotRewardEligible(t *testing.T) {
	stripe := stripeParams{mode: 2, rows: 4, leafCount: 12, slotCount: 3}
	deal := types.Deal{
		RedundancyMode: 2,
		Mode2Profile:   &types.StripeReplicaProfile{K: 2, M: 1},
		Mode2Slots: []*types.DealSlot{
			{Slot: 0, Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
			{Slot: 1, Status: types.SlotStatus_SLOT_STATUS_REPAIRING},
			{Slot: 2, Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
		},
	}

	eligible, err := mode2SlotRewardEligible(deal, stripe, 0)
	require.NoError(t, err)
	require.True(t, eligible)

	eligible, err = mode2SlotRewardEligible(deal, stripe, 4)
	require.NoError(t, err)
	require.False(t, eligible)
}
