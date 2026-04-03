#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

RUNNER_SCRIPT="scripts/ci/run_sp_discipline_stack_gates.sh"
if [ ! -x "$RUNNER_SCRIPT" ]; then
  echo "ERROR: missing executable gate runner: $RUNNER_SCRIPT" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

INVOKE_LOG="$TMPDIR/invocations.log"
touch "$INVOKE_LOG"

expect_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Fq -- "$pattern" "$file"; then
    echo "ERROR: expected pattern not found in $file: $pattern" >&2
    exit 1
  fi
}

write_pass_stub() {
  local path="$1"
  cat >"$path" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$(basename "$0") $*" >>"${SP_DISCIPLINE_TEST_INVOKE_LOG:?}"
exit 0
EOF
  chmod +x "$path"
}

STACK_CHECK="$TMPDIR/check_sp_discipline_stack_integrity.sh"
STACK_SCENARIOS="$TMPDIR/test_sp_discipline_stack_integrity.sh"
DOCS_CHECK="$TMPDIR/check_sp_discipline_docs_consistency.sh"
DOCS_SCENARIOS="$TMPDIR/test_sp_discipline_docs_consistency.sh"
POLICY_TEMPLATE_CHECK="$TMPDIR/check_sp_discipline_policy_template.sh"
POLICY_TEMPLATE_SCENARIOS="$TMPDIR/test_sp_discipline_policy_template.sh"
POLICY_INSTALL_CHECK="$TMPDIR/check_sp_discipline_policy_install_state.sh"
POLICY_INSTALL_SCENARIOS="$TMPDIR/test_sp_discipline_policy_install_state.sh"
YES_MERGE_CHECK_FIXTURE="$TMPDIR/check_yes_merge_fixture.sh"
YES_MERGE_CHECK_ALWAYS_PASS="$TMPDIR/check_yes_merge_always_pass.sh"
YES_MERGE_SCENARIOS="$TMPDIR/test_yes_merge_check.sh"
EVIDENCE_SCENARIOS="$TMPDIR/test_sp_discipline_evidence_capture.sh"
GATE_RUNNER_SCENARIOS_STUB="$TMPDIR/test_sp_discipline_gate_runner_stub.sh"
DOCS_CHECK_FAIL="$TMPDIR/check_sp_discipline_docs_consistency_fail.sh"

write_pass_stub "$STACK_CHECK"
write_pass_stub "$STACK_SCENARIOS"
write_pass_stub "$DOCS_CHECK"
write_pass_stub "$DOCS_SCENARIOS"
write_pass_stub "$POLICY_TEMPLATE_CHECK"
write_pass_stub "$POLICY_TEMPLATE_SCENARIOS"
write_pass_stub "$POLICY_INSTALL_CHECK"
write_pass_stub "$POLICY_INSTALL_SCENARIOS"
write_pass_stub "$YES_MERGE_SCENARIOS"
write_pass_stub "$EVIDENCE_SCENARIOS"
write_pass_stub "$GATE_RUNNER_SCENARIOS_STUB"

cat >"$YES_MERGE_CHECK_FIXTURE" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$(basename "$0") $*" >>"${SP_DISCIPLINE_TEST_INVOKE_LOG:?}"
if [ "${1:-}" != "--fixture" ]; then
  exit 2
fi
case "${2:-}" in
  scripts/ci/fixtures/no_approval.json)
    exit 1
    ;;
  scripts/ci/fixtures/yes_merge.json)
    exit 0
    ;;
  *)
    exit 2
    ;;
esac
EOF
chmod +x "$YES_MERGE_CHECK_FIXTURE"

cat >"$YES_MERGE_CHECK_ALWAYS_PASS" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$(basename "$0") $*" >>"${SP_DISCIPLINE_TEST_INVOKE_LOG:?}"
exit 0
EOF
chmod +x "$YES_MERGE_CHECK_ALWAYS_PASS"

cat >"$DOCS_CHECK_FAIL" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "$(basename "$0") $*" >>"${SP_DISCIPLINE_TEST_INVOKE_LOG:?}"
echo "simulated docs consistency failure" >&2
exit 9
EOF
chmod +x "$DOCS_CHECK_FAIL"

echo "==> Case 1: fast-mode happy path passes with fixture-aware YES MERGE checker"
: >"$INVOKE_LOG"
CASE1_OUT="$TMPDIR/case1.out"
SP_DISCIPLINE_TEST_INVOKE_LOG="$INVOKE_LOG" \
SP_DISCIPLINE_STACK_FAST_ONLY=1 \
SP_DISCIPLINE_STACK_INTEGRITY_CHECK="$STACK_CHECK" \
SP_DISCIPLINE_STACK_INTEGRITY_SCENARIOS="$STACK_SCENARIOS" \
SP_DISCIPLINE_DOCS_CONSISTENCY_CHECK="$DOCS_CHECK" \
SP_DISCIPLINE_DOCS_CONSISTENCY_SCENARIOS="$DOCS_SCENARIOS" \
SP_DISCIPLINE_POLICY_TEMPLATE_CHECK="$POLICY_TEMPLATE_CHECK" \
SP_DISCIPLINE_POLICY_TEMPLATE_SCENARIOS="$POLICY_TEMPLATE_SCENARIOS" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_CHECK="$POLICY_INSTALL_CHECK" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_SCENARIOS="$POLICY_INSTALL_SCENARIOS" \
SP_DISCIPLINE_YES_MERGE_CHECK="$YES_MERGE_CHECK_FIXTURE" \
SP_DISCIPLINE_YES_MERGE_SCENARIOS="$YES_MERGE_SCENARIOS" \
SP_DISCIPLINE_EVIDENCE_CAPTURE_SCENARIOS="$EVIDENCE_SCENARIOS" \
SP_DISCIPLINE_GATE_RUNNER_SCENARIOS="$GATE_RUNNER_SCENARIOS_STUB" \
bash "$RUNNER_SCRIPT" >"$CASE1_OUT" 2>&1

expect_contains "$CASE1_OUT" "Fast-only mode enabled; skipping heavy keeper/e2e/frontend gates"
expect_contains "$CASE1_OUT" "All SP discipline stack gates passed."
expect_contains "$INVOKE_LOG" "check_yes_merge_fixture.sh --fixture scripts/ci/fixtures/no_approval.json"
expect_contains "$INVOKE_LOG" "check_yes_merge_fixture.sh --fixture scripts/ci/fixtures/yes_merge.json"

echo "==> Case 2: docs consistency failure stops runner early"
: >"$INVOKE_LOG"
CASE2_OUT="$TMPDIR/case2.out"
set +e
SP_DISCIPLINE_TEST_INVOKE_LOG="$INVOKE_LOG" \
SP_DISCIPLINE_STACK_FAST_ONLY=1 \
SP_DISCIPLINE_STACK_INTEGRITY_CHECK="$STACK_CHECK" \
SP_DISCIPLINE_STACK_INTEGRITY_SCENARIOS="$STACK_SCENARIOS" \
SP_DISCIPLINE_DOCS_CONSISTENCY_CHECK="$DOCS_CHECK_FAIL" \
SP_DISCIPLINE_DOCS_CONSISTENCY_SCENARIOS="$DOCS_SCENARIOS" \
SP_DISCIPLINE_POLICY_TEMPLATE_CHECK="$POLICY_TEMPLATE_CHECK" \
SP_DISCIPLINE_POLICY_TEMPLATE_SCENARIOS="$POLICY_TEMPLATE_SCENARIOS" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_CHECK="$POLICY_INSTALL_CHECK" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_SCENARIOS="$POLICY_INSTALL_SCENARIOS" \
SP_DISCIPLINE_YES_MERGE_CHECK="$YES_MERGE_CHECK_FIXTURE" \
SP_DISCIPLINE_YES_MERGE_SCENARIOS="$YES_MERGE_SCENARIOS" \
SP_DISCIPLINE_EVIDENCE_CAPTURE_SCENARIOS="$EVIDENCE_SCENARIOS" \
SP_DISCIPLINE_GATE_RUNNER_SCENARIOS="$GATE_RUNNER_SCENARIOS_STUB" \
bash "$RUNNER_SCRIPT" >"$CASE2_OUT" 2>&1
CASE2_STATUS=$?
set -e
if [ "$CASE2_STATUS" -ne 9 ]; then
  echo "ERROR: expected docs-check failure status 9, got $CASE2_STATUS" >&2
  exit 1
fi
expect_contains "$CASE2_OUT" "simulated docs consistency failure"
if grep -Fq -- "test_sp_discipline_docs_consistency.sh" "$INVOKE_LOG"; then
  echo "ERROR: docs-consistency scenarios ran despite docs check failure" >&2
  exit 1
fi

echo "==> Case 3: expected-fail guard fails when negative fixture unexpectedly passes"
: >"$INVOKE_LOG"
CASE3_OUT="$TMPDIR/case3.out"
set +e
SP_DISCIPLINE_TEST_INVOKE_LOG="$INVOKE_LOG" \
SP_DISCIPLINE_STACK_FAST_ONLY=1 \
SP_DISCIPLINE_STACK_INTEGRITY_CHECK="$STACK_CHECK" \
SP_DISCIPLINE_STACK_INTEGRITY_SCENARIOS="$STACK_SCENARIOS" \
SP_DISCIPLINE_DOCS_CONSISTENCY_CHECK="$DOCS_CHECK" \
SP_DISCIPLINE_DOCS_CONSISTENCY_SCENARIOS="$DOCS_SCENARIOS" \
SP_DISCIPLINE_POLICY_TEMPLATE_CHECK="$POLICY_TEMPLATE_CHECK" \
SP_DISCIPLINE_POLICY_TEMPLATE_SCENARIOS="$POLICY_TEMPLATE_SCENARIOS" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_CHECK="$POLICY_INSTALL_CHECK" \
SP_DISCIPLINE_POLICY_INSTALL_STATE_SCENARIOS="$POLICY_INSTALL_SCENARIOS" \
SP_DISCIPLINE_YES_MERGE_CHECK="$YES_MERGE_CHECK_ALWAYS_PASS" \
SP_DISCIPLINE_YES_MERGE_SCENARIOS="$YES_MERGE_SCENARIOS" \
SP_DISCIPLINE_EVIDENCE_CAPTURE_SCENARIOS="$EVIDENCE_SCENARIOS" \
SP_DISCIPLINE_GATE_RUNNER_SCENARIOS="$GATE_RUNNER_SCENARIOS_STUB" \
bash "$RUNNER_SCRIPT" >"$CASE3_OUT" 2>&1
CASE3_STATUS=$?
set -e
if [ "$CASE3_STATUS" -ne 1 ]; then
  echo "ERROR: expected expected-fail guard status 1, got $CASE3_STATUS" >&2
  exit 1
fi
expect_contains "$CASE3_OUT" "ERROR: expected command to fail, but it passed"

echo "SP discipline gate-runner scenario tests passed."
