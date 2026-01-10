package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"nilchain/x/nilchain/types"
)

func providerMatchesServiceHint(provider types.Provider, serviceHint string) bool {
	info, err := types.ParseServiceHint(serviceHint)
	base := strings.TrimSpace(serviceHint)
	if err == nil && strings.TrimSpace(info.Base) != "" {
		base = info.Base
	}
	base = strings.ToLower(strings.TrimSpace(base))

	switch base {
	case "hot":
		return provider.Capabilities == "General" || provider.Capabilities == "Edge"
	case "cold":
		return provider.Capabilities == "Archive" || provider.Capabilities == "General"
	default:
		return true
	}
}

func (k Keeper) selectMode2ReplacementProvider(ctx sdk.Context, deal types.Deal, slot uint32, epochID uint64, attempt uint64) (string, error) {
	if len(deal.Mode2Slots) == 0 {
		return "", fmt.Errorf("mode2 slot map is empty")
	}

	params := k.GetParams(ctx)
	requiredBond, err := k.requiredBondPerSlot(ctx, deal)
	if err != nil {
		return "", err
	}

	outgoing := ""
	if int(slot) >= 0 && int(slot) < len(deal.Mode2Slots) {
		if s := deal.Mode2Slots[int(slot)]; s != nil {
			outgoing = strings.TrimSpace(s.Provider)
		}
	}

	exclude := make(map[string]struct{}, len(deal.Mode2Slots)*2)
	for _, s := range deal.Mode2Slots {
		if s == nil {
			continue
		}
		if addr := strings.TrimSpace(s.Provider); addr != "" {
			exclude[addr] = struct{}{}
		}
		if addr := strings.TrimSpace(s.PendingProvider); addr != "" {
			exclude[addr] = struct{}{}
		}
	}

	candidates := make([]string, 0, 8)
	if err := k.Providers.Walk(ctx, nil, func(addr string, provider types.Provider) (stop bool, err error) {
		if strings.TrimSpace(provider.Status) != "Active" {
			return false, nil
		}
		if !providerMatchesServiceHint(provider, deal.ServiceHint) {
			return false, nil
		}
		if !k.providerMeetsMinBond(ctx, provider, params) {
			return false, nil
		}
		if !k.providerHasAvailableBond(ctx, provider, requiredBond) {
			return false, nil
		}
		if _, blocked := exclude[strings.TrimSpace(provider.Address)]; blocked {
			return false, nil
		}
		candidates = append(candidates, provider.Address)
		return false, nil
	}); err != nil {
		return "", err
	}

	// Devnet/PoC fallback: when the network has exactly N=K+M providers and the deal
	// uses all of them, there may be no "unused" candidates to select from. In this
	// case, deterministically reuse another active provider (excluding the outgoing
	// one) so repairs remain possible without requiring extra providers.
	if len(candidates) == 0 {
		if err := k.Providers.Walk(ctx, nil, func(addr string, provider types.Provider) (stop bool, err error) {
			if strings.TrimSpace(provider.Status) != "Active" {
				return false, nil
			}
			if !providerMatchesServiceHint(provider, deal.ServiceHint) {
				return false, nil
			}
			if !k.providerMeetsMinBond(ctx, provider, params) {
				return false, nil
			}
			if !k.providerHasAvailableBond(ctx, provider, requiredBond) {
				return false, nil
			}
			cand := strings.TrimSpace(provider.Address)
			if cand == "" || cand == outgoing {
				return false, nil
			}
			candidates = append(candidates, cand)
			return false, nil
		}); err != nil {
			return "", err
		}
		if len(candidates) == 0 {
			return "", fmt.Errorf("no replacement provider candidates available")
		}
	}

	sort.Strings(candidates)

	seed := k.getEpochSeed(ctx, epochID)
	buf := make([]byte, 0, 32+8+4+8+8)
	buf = append(buf, seed[:]...)
	buf = append(buf, sdk.Uint64ToBigEndian(deal.Id)...)
	var slotBytes [4]byte
	binary.BigEndian.PutUint32(slotBytes[:], slot)
	buf = append(buf, slotBytes[:]...)
	buf = append(buf, sdk.Uint64ToBigEndian(deal.CurrentGen)...)
	buf = append(buf, sdk.Uint64ToBigEndian(attempt)...)
	sum := sha256.Sum256(buf)

	idx := int(binary.BigEndian.Uint64(sum[:8]) % uint64(len(candidates)))
	return candidates[idx], nil
}

func (k Keeper) prepareMode2RepairStart(ctx sdk.Context, dealID uint64, slot uint32) (uint64, uint64, error) {
	params := k.GetParams(ctx)
	if err := k.checkMode2RepairCooldown(ctx, dealID, slot, params); err != nil {
		return 0, 0, err
	}
	return k.nextMode2RepairAttempt(ctx, dealID, slot, params)
}

func (k Keeper) checkMode2RepairCooldown(ctx sdk.Context, dealID uint64, slot uint32, params types.Params) error {
	if params.ReplacementCooldownBlocks == 0 {
		return nil
	}
	key := collections.Join(dealID, slot)
	lastStart, err := k.Mode2RepairLastStart.Get(ctx, key)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil {
		height := uint64(ctx.BlockHeight())
		readyAt := lastStart + params.ReplacementCooldownBlocks
		if height < readyAt {
			remaining := readyAt - height
			return sdkerrors.ErrInvalidRequest.Wrapf("replacement cooldown active (%d blocks remaining)", remaining)
		}
	}
	return nil
}

func (k Keeper) nextMode2RepairAttempt(ctx sdk.Context, dealID uint64, slot uint32, params types.Params) (uint64, uint64, error) {
	if params.RepairAttemptsCap == 0 || params.RepairAttemptWindowBlocks == 0 {
		return 0, 0, sdkerrors.ErrInvalidRequest.Wrap("repair attempt limits are not configured")
	}

	key := collections.Join(dealID, slot)
	windowStart, err := k.Mode2RepairWindowStart.Get(ctx, key)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return 0, 0, err
	}
	attempts, err := k.Mode2RepairAttempts.Get(ctx, key)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return 0, 0, err
	}

	height := uint64(ctx.BlockHeight())
	if windowStart == 0 || height >= windowStart+params.RepairAttemptWindowBlocks {
		windowStart = height
		attempts = 0
	}

	if attempts >= params.RepairAttemptsCap {
		return 0, 0, sdkerrors.ErrInvalidRequest.Wrap("repair attempts cap reached")
	}

	return attempts + 1, windowStart, nil
}

func (k Keeper) recordMode2RepairStart(ctx sdk.Context, dealID uint64, slot uint32, attempt uint64, windowStart uint64) error {
	key := collections.Join(dealID, slot)
	if err := k.Mode2RepairWindowStart.Set(ctx, key, windowStart); err != nil {
		return err
	}
	if err := k.Mode2RepairAttempts.Set(ctx, key, attempt); err != nil {
		return err
	}
	return k.Mode2RepairLastStart.Set(ctx, key, uint64(ctx.BlockHeight()))
}
