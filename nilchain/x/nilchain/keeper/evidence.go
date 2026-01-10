package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

var evidenceSeedTag = []byte("nilstore/evidence/v1")

func deriveEvidenceID(kind string, dealID uint64, epochID uint64, extra []byte) [32]byte {
	buf := make([]byte, 0, len(evidenceSeedTag)+len(kind)+8+8+len(extra))
	buf = append(buf, evidenceSeedTag...)
	buf = append(buf, []byte(kind)...)
	buf = binary.BigEndian.AppendUint64(buf, dealID)
	buf = binary.BigEndian.AppendUint64(buf, epochID)
	buf = append(buf, extra...)
	return sha256.Sum256(buf)
}

func (k Keeper) evidenceRecorded(ctx sdk.Context, evidenceID []byte) (bool, error) {
	if len(evidenceID) == 0 {
		return false, nil
	}
	_, err := k.EvidenceRecords.Get(ctx, evidenceID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, collections.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func (k Keeper) recordEvidence(ctx sdk.Context, evidenceID []byte) error {
	if len(evidenceID) == 0 {
		return nil
	}
	return k.EvidenceRecords.Set(ctx, evidenceID, uint64(ctx.BlockHeight()))
}

func (k Keeper) slashProviderBond(ctx sdk.Context, provider string, bps uint64) (math.Int, error) {
	if bps == 0 {
		return math.ZeroInt(), nil
	}
	addr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return math.ZeroInt(), err
	}

	spendable := k.BankKeeper.SpendableCoins(ctx, addr).AmountOf(sdk.DefaultBondDenom)
	if !spendable.IsPositive() {
		return math.ZeroInt(), nil
	}

	bpsInt := math.NewIntFromUint64(bps)
	divisor := math.NewInt(10000)
	slash := spendable.Mul(bpsInt).Quo(divisor)
	if slash.IsZero() {
		return math.ZeroInt(), nil
	}
	if slash.GT(spendable) {
		slash = spendable
	}

	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, slash))
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
		return math.ZeroInt(), err
	}
	if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return math.ZeroInt(), err
	}

	state, err := k.ensureProviderBondState(ctx, provider)
	if err == nil {
		_ = k.ProviderBonds.Set(ctx, strings.TrimSpace(provider), state)
	}

	return slash, nil
}

func (k Keeper) jailProvider(ctx sdk.Context, provider string, jailEpochs uint64) (int64, error) {
	if jailEpochs == 0 {
		return 0, nil
	}
	params := k.GetParams(ctx)
	if params.EpochLenBlocks == 0 {
		return 0, nil
	}
	jailBlocks, overflow := mulUint64(jailEpochs, params.EpochLenBlocks)
	if overflow {
		return 0, fmt.Errorf("jail duration overflow")
	}
	end := ctx.BlockHeight() + int64(jailBlocks)
	if err := k.ProviderJails.Set(ctx, strings.TrimSpace(provider), end); err != nil {
		return 0, err
	}
	if record, err := k.Providers.Get(ctx, provider); err == nil {
		record.Status = "Jailed"
		_ = k.Providers.Set(ctx, provider, record)
	}
	return end, nil
}

func (k Keeper) providerIsJailed(ctx sdk.Context, provider string) (bool, int64, error) {
	end, err := k.ProviderJails.Get(ctx, strings.TrimSpace(provider))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, 0, nil
		}
		return false, 0, err
	}
	if ctx.BlockHeight() >= end {
		_ = k.ProviderJails.Remove(ctx, strings.TrimSpace(provider))
		if record, err := k.Providers.Get(ctx, provider); err == nil && strings.TrimSpace(record.Status) == "Jailed" {
			record.Status = "Active"
			_ = k.Providers.Set(ctx, provider, record)
		}
		return false, 0, nil
	}
	return true, end, nil
}

func (k Keeper) applyHardFaultEvidence(
	ctx sdk.Context,
	dealID uint64,
	provider string,
	kind string,
	epochID uint64,
	extra []byte,
	slashBps uint64,
	jailEpochs uint64,
) (bool, error) {
	eid := deriveEvidenceID(kind, dealID, epochID, extra)
	seen, err := k.evidenceRecorded(ctx, eid[:])
	if err != nil {
		return false, err
	}
	if seen {
		return false, nil
	}
	if err := k.recordEvidence(ctx, eid[:]); err != nil {
		return false, err
	}
	if _, err := k.slashProviderBond(ctx, provider, slashBps); err != nil {
		return true, err
	}
	if _, err := k.jailProvider(ctx, provider, jailEpochs); err != nil {
		return true, err
	}
	return true, nil
}
