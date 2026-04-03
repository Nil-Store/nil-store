#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

STACK_INTEGRITY_CHECK="${SP_DISCIPLINE_STACK_INTEGRITY_CHECK:-scripts/ci/check_sp_discipline_stack_integrity.sh}"
STACK_INTEGRITY_SCENARIOS="${SP_DISCIPLINE_STACK_INTEGRITY_SCENARIOS:-scripts/ci/test_sp_discipline_stack_integrity.sh}"
DOCS_CONSISTENCY_CHECK="${SP_DISCIPLINE_DOCS_CONSISTENCY_CHECK:-scripts/ci/check_sp_discipline_docs_consistency.sh}"
DOCS_CONSISTENCY_SCENARIOS="${SP_DISCIPLINE_DOCS_CONSISTENCY_SCENARIOS:-scripts/ci/test_sp_discipline_docs_consistency.sh}"
POLICY_TEMPLATE_CHECK="${SP_DISCIPLINE_POLICY_TEMPLATE_CHECK:-scripts/ci/check_sp_discipline_policy_template.sh}"
POLICY_TEMPLATE_SCENARIOS="${SP_DISCIPLINE_POLICY_TEMPLATE_SCENARIOS:-scripts/ci/test_sp_discipline_policy_template.sh}"
POLICY_INSTALL_STATE_CHECK="${SP_DISCIPLINE_POLICY_INSTALL_STATE_CHECK:-scripts/ci/check_sp_discipline_policy_install_state.sh}"
POLICY_INSTALL_STATE_SCENARIOS="${SP_DISCIPLINE_POLICY_INSTALL_STATE_SCENARIOS:-scripts/ci/test_sp_discipline_policy_install_state.sh}"
YES_MERGE_CHECK="${SP_DISCIPLINE_YES_MERGE_CHECK:-scripts/ci/check_yes_merge.sh}"
YES_MERGE_SCENARIOS="${SP_DISCIPLINE_YES_MERGE_SCENARIOS:-scripts/ci/test_yes_merge_check.sh}"
EVIDENCE_CAPTURE_SCENARIOS="${SP_DISCIPLINE_EVIDENCE_CAPTURE_SCENARIOS:-scripts/ci/test_sp_discipline_evidence_capture.sh}"
GATE_RUNNER_SCENARIOS="${SP_DISCIPLINE_GATE_RUNNER_SCENARIOS:-scripts/ci/test_sp_discipline_gate_runner.sh}"
FAST_ONLY="${SP_DISCIPLINE_STACK_FAST_ONLY:-0}"

KEEPER_SUBSET_CMD="${SP_DISCIPLINE_KEEPER_SUBSET_CMD:-cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/keeper -run 'TestCheckMissedProofs_.*|Test.*Discipline.*|Test.*Status.*' -count=2}"
KEEPER_FULL_CMD="${SP_DISCIPLINE_KEEPER_FULL_CMD:-cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/keeper}"
NILCHAIN_MODULES_CMD="${SP_DISCIPLINE_NILCHAIN_MODULES_CMD:-cd nilchain && GOFLAGS=-mod=mod go test ./x/nilchain/...}"
DEPUTY_E2E_CMD="${SP_DISCIPLINE_DEPUTY_E2E_CMD:-bash ./scripts/e2e_deputy_ghost_repair_multi_sp.sh}"
WEBSITE_UNIT_CMD="${SP_DISCIPLINE_WEBSITE_UNIT_CMD:-npm --prefix nil-website run test:unit}"
WEBSITE_BUILD_CMD="${SP_DISCIPLINE_WEBSITE_BUILD_CMD:-npm --prefix nil-website run build:app}"

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
  bash "$STACK_INTEGRITY_CHECK"

run_ok "SP discipline stack-integrity scenarios" \
  bash "$STACK_INTEGRITY_SCENARIOS"

run_ok "SP discipline docs consistency" \
  bash "$DOCS_CONSISTENCY_CHECK"

run_ok "SP discipline docs consistency scenarios" \
  bash "$DOCS_CONSISTENCY_SCENARIOS"

run_ok "SP discipline policy template" \
  bash "$POLICY_TEMPLATE_CHECK"

run_ok "SP discipline policy-template scenarios" \
  bash "$POLICY_TEMPLATE_SCENARIOS"

run_ok "SP discipline policy install state" \
  bash "$POLICY_INSTALL_STATE_CHECK"

run_ok "SP discipline policy install-state scenarios" \
  bash "$POLICY_INSTALL_STATE_SCENARIOS"

expected_fail "YES MERGE negative fixture check (must fail)" \
  bash "$YES_MERGE_CHECK" --fixture scripts/ci/fixtures/no_approval.json

run_ok "YES MERGE positive fixture check (must pass)" \
  bash "$YES_MERGE_CHECK" --fixture scripts/ci/fixtures/yes_merge.json

run_ok "YES MERGE scenario checks" \
  bash "$YES_MERGE_SCENARIOS"

run_ok "SP discipline evidence-capture scenarios" \
  bash "$EVIDENCE_CAPTURE_SCENARIOS"

run_ok "SP discipline gate-runner scenarios" \
  bash "$GATE_RUNNER_SCENARIOS"

if [ "$FAST_ONLY" = "1" ]; then
  step "Fast-only mode enabled; skipping heavy keeper/e2e/frontend gates"
  echo "SP_DISCIPLINE_STACK_FAST_ONLY=1"
  echo
  echo "All SP discipline stack gates passed."
  exit 0
fi

run_ok "keeper determinism/status subset (count=2)" \
  bash -lc "$KEEPER_SUBSET_CMD"

run_ok "keeper full package tests" \
  bash -lc "$KEEPER_FULL_CMD"

run_ok "nilchain module tests" \
  bash -lc "$NILCHAIN_MODULES_CMD"

run_ok "deputy ghost repair e2e" \
  bash -lc "$DEPUTY_E2E_CMD"

run_ok "nil-website unit tests" \
  bash -lc "$WEBSITE_UNIT_CMD"

run_ok "nil-website production build" \
  bash -lc "$WEBSITE_BUILD_CMD"

echo
echo "All SP discipline stack gates passed."
