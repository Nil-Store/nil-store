package keeper

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

// selectMode1ReplacementProvider picks a deterministic active replacement for a
// non-active Mode1 provider. This is a best-effort parity path for Mode1 deals.
func (k Keeper) selectMode1ReplacementProvider(ctx sdk.Context, deal types.Deal, outgoingIndex int, epochID uint64) (string, error) {
	if outgoingIndex < 0 || outgoingIndex >= len(deal.Providers) {
		return "", fmt.Errorf("invalid outgoing index %d", outgoingIndex)
	}
	outgoing := strings.TrimSpace(deal.Providers[outgoingIndex])
	if outgoing == "" {
		return "", fmt.Errorf("outgoing provider is empty")
	}

	exclude := make(map[string]struct{}, len(deal.Providers))
	for i, p := range deal.Providers {
		addr := strings.TrimSpace(p)
		if addr == "" {
			continue
		}
		if i == outgoingIndex {
			continue
		}
		exclude[addr] = struct{}{}
	}

	candidates := make([]string, 0, 8)
	if err := k.Providers.Walk(ctx, nil, func(_ string, provider types.Provider) (stop bool, err error) {
		addr := strings.TrimSpace(provider.Address)
		if addr == "" || addr == outgoing {
			return false, nil
		}
		if strings.TrimSpace(provider.Status) != "Active" {
			return false, nil
		}
		if provider.Draining {
			return false, nil
		}
		if !providerMatchesServiceHint(provider, deal.ServiceHint) {
			return false, nil
		}
		if _, blocked := exclude[addr]; blocked {
			return false, nil
		}
		candidates = append(candidates, addr)
		return false, nil
	}); err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no mode1 replacement candidates available")
	}

	seed := k.getEpochSeed(ctx, epochID)
	idx := deterministicIndex(seed, deal.Id, uint32(outgoingIndex), deal.CurrentGen, len(candidates))
	return candidates[idx], nil
}
