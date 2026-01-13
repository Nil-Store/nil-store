#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="$ROOT_DIR/market_pricing_rfc_assets.md"

files=(
  "AGENTS_MAINNET_PARITY.md"
  "MAINNET_GAP_TRACKER.md"
  "MAINNET_ECON_PARITY_CHECKLIST.md"
  "notes/mainnet_policy_resolution_jan2026.md"
  "ECONOMY.md"
  "spec.md"
  "rfcs/rfc-pricing-and-escrow-accounting.md"
  "rfcs/rfc-challenge-derivation-and-quotas.md"
  "rfcs/rfc-retrieval-validation.md"
  "rfcs/rfc-mode2-onchain-state.md"
  "nilchain/proto/nilchain/nilchain/v1/params.proto"
)

: > "$OUT_FILE"

for path in "${files[@]}"; do
  abs="$ROOT_DIR/$path"
  if [[ ! -f "$abs" ]]; then
    echo "WARN: missing $path" >> "$OUT_FILE"
    continue
  fi
  ext="${path##*.}"
  echo "\`\`\`$path" >> "$OUT_FILE"
  cat "$abs" >> "$OUT_FILE"
  echo "\`\`\`" >> "$OUT_FILE"
  echo >> "$OUT_FILE"
done

echo "Wrote $OUT_FILE"
