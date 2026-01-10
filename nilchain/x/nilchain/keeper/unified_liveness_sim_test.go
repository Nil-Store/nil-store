package keeper

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/types"
)

func TestChallengeDerivationSimDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(1337))
	params := types.DefaultParams()
	chainID := "sim-chain"

	headerHash := make([]byte, 32)
	_, err := rng.Read(headerHash)
	require.NoError(t, err)

	for epochID := uint64(1); epochID <= 3; epochID++ {
		seed := deriveEpochSeed(chainID, epochID, headerHash)

		for dealIdx := 0; dealIdx < 4; dealIdx++ {
			dealID := uint64(100 + dealIdx)
			currentGen := uint64(1 + dealIdx)
			witnessMdus := uint64(1)
			totalMdus := witnessMdus + uint64(10+rng.Intn(4))

			deal := types.Deal{
				Id:             dealID,
				CurrentGen:     currentGen,
				TotalMdus:      totalMdus,
				WitnessMdus:    witnessMdus,
				ServiceHint:    "Hot",
				RedundancyMode: 1,
			}

			if dealIdx%2 == 1 {
				deal.RedundancyMode = 2
				deal.Mode2Profile = &types.StripeReplicaProfile{K: 8, M: 4}
				slotCount := int(deal.Mode2Profile.K + deal.Mode2Profile.M)
				deal.Mode2Slots = make([]*types.DealSlot, slotCount)
				for i := 0; i < slotCount; i++ {
					status := types.SlotStatus_SLOT_STATUS_ACTIVE
					if i == 1 {
						status = types.SlotStatus_SLOT_STATUS_REPAIRING
					}
					deal.Mode2Slots[i] = &types.DealSlot{
						Slot:     uint32(i),
						Status:   status,
						Provider: "provider",
					}
				}
			}

			in, ok := slabInputs(deal)
			require.True(t, ok)
			require.Greater(t, in.userMdus, uint64(0))

			if deal.RedundancyMode == 2 {
				stripe, err := stripeParamsForDeal(deal)
				require.NoError(t, err)

				required := requiredBlobsMode2(params, deal, stripe, in)
				require.Greater(t, required, uint64(0))

				ordinalLimit := required
				if ordinalLimit > 3 {
					ordinalLimit = 3
				}

				for slot := uint32(0); slot < 3; slot++ {
					if !mode2SlotActiveForSynthetic(deal, slot) {
						require.False(t, mode2SlotActiveForSynthetic(deal, slot))
						continue
					}

					for ordinal := uint64(0); ordinal < ordinalLimit; ordinal++ {
						mduA, blobA := deriveMode2Challenge(seed, deal.Id, deal.CurrentGen, uint64(slot), ordinal, in, stripe)
						mduB, blobB := deriveMode2Challenge(seed, deal.Id, deal.CurrentGen, uint64(slot), ordinal, in, stripe)

						require.Equal(t, mduA, mduB)
						require.Equal(t, blobA, blobB)
						require.GreaterOrEqual(t, mduA, in.metaMdus)
						require.Less(t, mduA, in.metaMdus+in.userMdus)
						require.Less(t, blobA, uint32(stripe.leafCount))
						require.True(t, matchesMode2SyntheticChallenge(seed, deal.Id, deal.CurrentGen, slot, in, stripe, required, mduA, blobA))
					}
				}
				continue
			}

			required := requiredBlobsMode1(params, deal, in)
			require.Greater(t, required, uint64(0))

			providers := make([][]byte, 2)
			for i := range providers {
				provider := make([]byte, 20)
				_, err := rng.Read(provider)
				require.NoError(t, err)
				providers[i] = provider
			}

			ordinalLimit := required
			if ordinalLimit > 3 {
				ordinalLimit = 3
			}

			for _, provider := range providers {
				for ordinal := uint64(0); ordinal < ordinalLimit; ordinal++ {
					mduA, blobA := deriveMode1Challenge(seed, deal.Id, deal.CurrentGen, provider, ordinal, in)
					mduB, blobB := deriveMode1Challenge(seed, deal.Id, deal.CurrentGen, provider, ordinal, in)

					require.Equal(t, mduA, mduB)
					require.Equal(t, blobA, blobB)
					require.GreaterOrEqual(t, mduA, in.metaMdus)
					require.Less(t, mduA, in.metaMdus+in.userMdus)
					require.Less(t, blobA, uint32(types.BlobsPerMdu))
					require.True(t, matchesMode1SyntheticChallenge(seed, deal.Id, deal.CurrentGen, provider, in, required, mduA, blobA))
				}
			}
		}
	}
}
