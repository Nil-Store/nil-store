#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

step() {
  local msg="$1"
  echo
  echo "==> $msg"
}

expected_fail() {
  local msg="$1"
  shift
  step "$msg"
  if "$@"; then
    echo "ERROR: expected command to fail, but it passed" >&2
    return 1
  fi
  echo "OK: command failed as expected"
}

run_ok() {
  local msg="$1"
  shift
  step "$msg"
  "$@"
}

run_ok "SP discipline stack integrity" \
  bash scripts/ci/check_sp_discipline_stack_integrity.sh

run_ok "SP discipline stack-integrity scenarios" \
  bash scripts/ci/test_sp_discipline_stack_integrity.sh

run_ok "SP discipline docs consistency" \
  bash scripts/ci/check_sp_discipline_docs_consistency.sh

run_ok "SP discipline docs consistency scenarios" \
  bash scripts/ci/test_sp_discipline_docs_consistency.sh

run_ok "SP discipline policy template" \
  bash scripts/ci/check_sp_discipline_policy_template.sh

run_ok "SP discipline policy-template scenarios" \
  bash scripts/ci/test_sp_discipline_policy_template.sh

run_ok "SP discipline policy install state" \
  bash scripts/ci/check_sp_discipline_policy_install_state.sh

run_ok "SP discipline policy install-state scenarios" \
  bash scripts/ci/test_sp_discipline_policy_install_state.sh

expected_fail "YES MERGE negative fixture check (must fail)" \
  bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json

run_ok "YES MERGE positive fixture check (must pass)" \
  bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json

run_ok "YES MERGE scenario checks" \
  bash scripts/ci/test_yes_merge_check.sh

run_ok "SP discipline evidence-capture scenarios" \
  bash scripts/ci/test_sp_discipline_evidence_capture.sh

run_ok "keeper determinism/status subset (count=2)" \
  bash -lc "cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/keeper -run 'TestCheckMissedProofs_.*|Test.*Discipline.*|Test.*Status.*' -count=2"

run_ok "keeper full package tests" \
  bash -lc "cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/keeper"

run_ok "nilchain module tests" \
  bash -lc "cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/..."

run_ok "deputy ghost repair e2e" \
  bash ./scripts/e2e_deputy_ghost_repair_multi_sp.sh

run_ok "nil-website unit tests" \
  npm --prefix nil-website run test:unit

run_ok "nil-website production build" \
  npm --prefix nil-website run build:app

echo
echo "All SP discipline stack gates passed."
