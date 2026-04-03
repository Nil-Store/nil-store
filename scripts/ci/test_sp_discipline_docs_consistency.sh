#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CHECK_SCRIPT="scripts/ci/check_sp_discipline_docs_consistency.sh"
if [ ! -x "$CHECK_SCRIPT" ]; then
  echo "ERROR: missing executable check script: $CHECK_SCRIPT" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

PLAN_DOC="$TMPDIR/plan.md"
EVIDENCE_DOC="$TMPDIR/evidence.md"

write_docs() {
  local plan_list="$1"
  local evidence_list="$2"
  local include_yes_merge="${3:-1}"
  cat >"$PLAN_DOC" <<EOF
# Plan
$plan_list
YES MERGE
EOF
  if [ "$include_yes_merge" = "1" ]; then
    cat >"$EVIDENCE_DOC" <<EOF
# Evidence
$evidence_list
YES MERGE
EOF
  else
    cat >"$EVIDENCE_DOC" <<EOF
# Evidence
$evidence_list
EOF
  fi
}

run_case() {
  SP_DISCIPLINE_PLAN_DOC_PATH="$PLAN_DOC" \
  SP_DISCIPLINE_EVIDENCE_DOC_PATH="$EVIDENCE_DOC" \
  bash "$CHECK_SCRIPT" "$@"
}

expect_fail() {
  local label="$1"
  shift
  if run_case "$@"; then
    echo "ERROR: expected failure for case: $label" >&2
    exit 1
  fi
  echo "OK: expected failure for case: $label"
}

BASE_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-01-taxonomy-and-signals`'
MISMATCH_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-02-conviction-state`'
DUP_PLAN_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-01-taxonomy-and-signals`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
DUP_EVIDENCE_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-01-taxonomy-and-signals`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
GAP_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-02-conviction-state`'
OUT_OF_ORDER_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-02-conviction-state`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
ORDINAL_GAP_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
ORDINAL_OFFSET_LIST=$'2. `stack/sp-discipline-00-merge-gate`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'

echo "==> Case 1: matching branch lists and YES MERGE text pass"
write_docs "$BASE_LIST" "$BASE_LIST" "1"
run_case

echo "==> Case 2: mismatched branch lists fail"
write_docs "$BASE_LIST" "$MISMATCH_LIST" "1"
expect_fail "branch-mismatch"

echo "==> Case 3: duplicate plan branch entries fail"
write_docs "$DUP_PLAN_LIST" "$BASE_LIST" "1"
expect_fail "plan-duplicate"

echo "==> Case 4: duplicate evidence branch entries fail"
write_docs "$BASE_LIST" "$DUP_EVIDENCE_LIST" "1"
expect_fail "evidence-duplicate"

echo "==> Case 5: missing YES MERGE reminder text fails"
write_docs "$BASE_LIST" "$BASE_LIST" "0"
expect_fail "missing-yes-merge"

echo "==> Case 6: non-contiguous branch numbering fails"
write_docs "$GAP_LIST" "$GAP_LIST" "1"
expect_fail "branch-gap"

echo "==> Case 7: out-of-order branch numbering fails"
write_docs "$OUT_OF_ORDER_LIST" "$OUT_OF_ORDER_LIST" "1"
expect_fail "branch-out-of-order"

echo "==> Case 8: markdown list ordinal gaps fail"
write_docs "$ORDINAL_GAP_LIST" "$ORDINAL_GAP_LIST" "1"
expect_fail "ordinal-gap"

echo "==> Case 9: markdown list ordinal offset fails"
write_docs "$ORDINAL_OFFSET_LIST" "$ORDINAL_OFFSET_LIST" "1"
expect_fail "ordinal-offset"

echo "SP discipline docs consistency scenario tests passed."
