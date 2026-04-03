#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CHECK_SCRIPT="scripts/ci/check_sp_discipline_stack_integrity.sh"
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

STACK_DOC="$TMPDIR/stack.md"

write_stack_doc() {
  local branch_list="$1"
  cat >"$STACK_DOC" <<EOF
# Stack
$branch_list
EOF
}

run_case() {
  SP_STACK_DOC="$STACK_DOC" bash "$CHECK_SCRIPT" "$@"
}

run_case_with_base() {
  local base_ref="$1"
  SP_STACK_DOC="$STACK_DOC" SP_STACK_BASE_REF="$base_ref" bash "$CHECK_SCRIPT"
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

VALID_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-01-taxonomy-and-signals`'
DUP_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-01-taxonomy-and-signals`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
ORDER_BAD_LIST=$'1. `stack/sp-discipline-01-taxonomy-and-signals`\n2. `stack/sp-discipline-00-merge-gate`'
MISSING_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-99-not-a-real-branch`'
ORDINAL_GAP_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n3. `stack/sp-discipline-01-taxonomy-and-signals`'
INDEX_MISMATCH_LIST=$'1. `stack/sp-discipline-00-merge-gate`\n2. `stack/sp-discipline-02-conviction-state`'

echo "==> Case 1: valid stack order passes"
write_stack_doc "$VALID_LIST"
run_case

echo "==> Case 2: duplicate stack entries fail"
write_stack_doc "$DUP_LIST"
expect_fail "duplicate-branches"

echo "==> Case 3: missing branch reference fails"
write_stack_doc "$MISSING_LIST"
expect_fail "missing-branch"

echo "==> Case 4: ancestry order violation fails"
write_stack_doc "$ORDER_BAD_LIST"
expect_fail "order-violation"

MERGED_BASE_REF=""
for candidate in \
  "refs/remotes/origin/stack/sp-discipline-01-taxonomy-and-signals" \
  "refs/remotes/origin/stack/sp-discipline-02-conviction-state"; do
  if git show-ref --verify --quiet "$candidate"; then
    MERGED_BASE_REF="$candidate"
    break
  fi
done
if [ -z "$MERGED_BASE_REF" ]; then
  echo "ERROR: could not find a stack branch ref to use as merged-base fixture" >&2
  exit 2
fi

echo "==> Case 5: merged-into-base detection fails as expected"
write_stack_doc "$VALID_LIST"
if run_case_with_base "$MERGED_BASE_REF"; then
  echo "ERROR: expected failure for case: already-merged-into-base" >&2
  exit 1
fi
echo "OK: expected failure for case: already-merged-into-base"

echo "==> Case 6: markdown list ordinal gap fails"
write_stack_doc "$ORDINAL_GAP_LIST"
expect_fail "ordinal-gap"

echo "==> Case 7: list/branch index mismatch fails"
write_stack_doc "$INDEX_MISMATCH_LIST"
expect_fail "index-mismatch"

echo "SP discipline stack integrity scenario tests passed."
