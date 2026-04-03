package keeper

import (
	"errors"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type providerDisciplineBucket string

const (
	disciplineBucketInvalidProof providerDisciplineBucket = "invalid_proof"
	disciplineBucketNonResponse  providerDisciplineBucket = "non_response"
	disciplineBucketQuotaMiss    providerDisciplineBucket = "quota_miss"
	disciplineBucketDeputyMiss   providerDisciplineBucket = "deputy_miss"
	disciplineBucketHealthFail   providerDisciplineBucket = "health_fail"
)

func disciplineBucketFromEvidence(kind string, ok bool) providerDisciplineBucket {
	if ok {
		return ""
	}

	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "system_proof_invalid",
		"system_proof_rejected",
		"system_proof_wrong_challenge",
		"system_proof_wrong_provider":
		return disciplineBucketInvalidProof
	case "retrieval_non_response":
		return disciplineBucketNonResponse
	case "quota_miss_repair_started":
		return disciplineBucketQuotaMiss
	case "deputy_miss_repair_started":
		return disciplineBucketDeputyMiss
	case "provider_degraded_repair_started":
		return disciplineBucketHealthFail
	default:
		return ""
	}
}

func (k Keeper) disciplineDecayMaps() []collections.Map[string, uint64] {
	return []collections.Map[string, uint64]{
		k.ProviderDisciplineInvalidProof,
		k.ProviderDisciplineNonResponse,
		k.ProviderDisciplineQuotaMiss,
		k.ProviderDisciplineDeputyMiss,
		k.ProviderDisciplineHealthFail,
		k.ProviderDisciplineTotal,
	}
}

func (k Keeper) disciplineMapForBucket(bucket providerDisciplineBucket) collections.Map[string, uint64] {
	switch bucket {
	case disciplineBucketInvalidProof:
		return k.ProviderDisciplineInvalidProof
	case disciplineBucketNonResponse:
		return k.ProviderDisciplineNonResponse
	case disciplineBucketQuotaMiss:
		return k.ProviderDisciplineQuotaMiss
	case disciplineBucketDeputyMiss:
		return k.ProviderDisciplineDeputyMiss
	case disciplineBucketHealthFail:
		return k.ProviderDisciplineHealthFail
	default:
		return k.ProviderDisciplineTotal
	}
}

func (k Keeper) currentDisciplineEpoch(ctx sdk.Context) uint64 {
	params := k.GetParams(ctx)
	if params.EpochLenBlocks == 0 {
		return 1
	}
	eid := epochIDAtHeight(ctx.BlockHeight(), params.EpochLenBlocks)
	if eid == 0 {
		return 1
	}
	return eid
}

// applyProviderDisciplineDecay applies deterministic linear decay per elapsed
// epoch to keep conviction state bounded in time.
func (k Keeper) applyProviderDisciplineDecay(ctx sdk.Context, provider string, currentEpoch uint64) error {
	lastEpoch, err := k.ProviderDisciplineWindowEpoch.Get(ctx, provider)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if errors.Is(err, collections.ErrNotFound) || lastEpoch == 0 {
		return k.ProviderDisciplineWindowEpoch.Set(ctx, provider, currentEpoch)
	}
	if currentEpoch <= lastEpoch {
		return nil
	}

	delta := currentEpoch - lastEpoch
	for _, m := range k.disciplineDecayMaps() {
		cur, getErr := m.Get(ctx, provider)
		if getErr != nil && !errors.Is(getErr, collections.ErrNotFound) {
			return getErr
		}
		if errors.Is(getErr, collections.ErrNotFound) {
			continue
		}

		next := uint64(0)
		if cur > delta {
			next = cur - delta
		}
		if err := m.Set(ctx, provider, next); err != nil {
			return err
		}
	}

	return k.ProviderDisciplineWindowEpoch.Set(ctx, provider, currentEpoch)
}

func (k Keeper) incrementProviderDiscipline(ctx sdk.Context, provider string, kind string, ok bool) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil
	}
	bucket := disciplineBucketFromEvidence(kind, ok)
	if bucket == "" {
		return nil
	}

	currentEpoch := k.currentDisciplineEpoch(ctx)
	if err := k.applyProviderDisciplineDecay(ctx, provider, currentEpoch); err != nil {
		return err
	}

	bucketMap := k.disciplineMapForBucket(bucket)
	curBucket, err := bucketMap.Get(ctx, provider)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := bucketMap.Set(ctx, provider, curBucket+1); err != nil {
		return err
	}

	curTotal, err := k.ProviderDisciplineTotal.Get(ctx, provider)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err := k.ProviderDisciplineTotal.Set(ctx, provider, curTotal+1); err != nil {
		return err
	}
	if err := k.ProviderDisciplineWindowEpoch.Set(ctx, provider, currentEpoch); err != nil {
		return err
	}
	return k.applyProviderStatusFromDiscipline(ctx, provider)
}

func (k Keeper) applyProviderStatusFromDiscipline(ctx sdk.Context, provider string) error {
	p, err := k.Providers.Get(ctx, provider)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	total, err := k.ProviderDisciplineTotal.Get(ctx, provider)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	params := k.GetParams(ctx)
	offlineThreshold := params.EvictAfterMissedEpochs
	if offlineThreshold == 0 {
		offlineThreshold = 1
	}
	jailedThreshold := offlineThreshold * 2

	nextStatus := "Active"
	if total >= jailedThreshold {
		nextStatus = "Jailed"
	} else if total >= offlineThreshold {
		nextStatus = "Offline"
	}

	if p.Status == nextStatus {
		return nil
	}
	p.Status = nextStatus
	return k.Providers.Set(ctx, provider, p)
}

// refreshProviderDisciplineAndStatus applies epoch decay and updates provider
// status before selection logic reads provider eligibility.
func (k Keeper) refreshProviderDisciplineAndStatus(ctx sdk.Context, provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil
	}
	currentEpoch := k.currentDisciplineEpoch(ctx)
	if err := k.applyProviderDisciplineDecay(ctx, provider, currentEpoch); err != nil {
		return err
	}
	return k.applyProviderStatusFromDiscipline(ctx, provider)
}
