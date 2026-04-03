#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CAPTURE_SCRIPT="scripts/ci/capture_sp_discipline_gate_evidence.sh"
if [ ! -x "$CAPTURE_SCRIPT" ]; then
  echo "ERROR: missing executable capture script: $CAPTURE_SCRIPT" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

OUT_DIR="$TMPDIR/out"
mkdir -p "$OUT_DIR"

RUNNER_OK="$TMPDIR/runner-ok.sh"
RUNNER_FAIL="$TMPDIR/runner-fail.sh"
RUNNER_MISSING="$TMPDIR/runner-missing.sh"

cat >"$RUNNER_OK" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "runner ok output"
exit 0
EOF

cat >"$RUNNER_FAIL" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "runner fail output"
exit 7
EOF

chmod +x "$RUNNER_OK" "$RUNNER_FAIL"

expect_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Fq -- "$pattern" "$file"; then
    echo "ERROR: expected pattern not found in $file: $pattern" >&2
    exit 1
  fi
}

echo "==> Case 1: passing runner writes pass artifacts"
TS_PASS="20260403T120001Z"
SP_DISCIPLINE_EVIDENCE_DIR="$OUT_DIR" \
SP_DISCIPLINE_GATE_RUNNER="$RUNNER_OK" \
SP_DISCIPLINE_EVIDENCE_TS_UTC="$TS_PASS" \
bash "$CAPTURE_SCRIPT"

PASS_LOG="$OUT_DIR/sp_discipline_stack_gates_${TS_PASS}.log"
PASS_SUMMARY="$OUT_DIR/sp_discipline_stack_gates_${TS_PASS}.md"

if [ ! -f "$PASS_LOG" ] || [ ! -f "$PASS_SUMMARY" ]; then
  echo "ERROR: expected pass artifacts were not created" >&2
  exit 1
fi

expect_contains "$PASS_LOG" "runner ok output"
expect_contains "$PASS_SUMMARY" "- Command: \`bash $RUNNER_OK\`"
expect_contains "$PASS_SUMMARY" "- Result: pass"
expect_contains "$PASS_SUMMARY" "- Exit code: 0"
expect_contains "$PASS_SUMMARY" "YES MERGE"

echo "==> Case 2: failing runner writes fail artifacts and returns runner status"
TS_FAIL="20260403T120002Z"
set +e
SP_DISCIPLINE_EVIDENCE_DIR="$OUT_DIR" \
SP_DISCIPLINE_GATE_RUNNER="$RUNNER_FAIL" \
SP_DISCIPLINE_EVIDENCE_TS_UTC="$TS_FAIL" \
bash "$CAPTURE_SCRIPT"
FAIL_STATUS=$?
set -e

if [ "$FAIL_STATUS" -ne 7 ]; then
  echo "ERROR: expected exit status 7 for failing runner, got $FAIL_STATUS" >&2
  exit 1
fi

FAIL_LOG="$OUT_DIR/sp_discipline_stack_gates_${TS_FAIL}.log"
FAIL_SUMMARY="$OUT_DIR/sp_discipline_stack_gates_${TS_FAIL}.md"

if [ ! -f "$FAIL_LOG" ] || [ ! -f "$FAIL_SUMMARY" ]; then
  echo "ERROR: expected fail artifacts were not created" >&2
  exit 1
fi

expect_contains "$FAIL_LOG" "runner fail output"
expect_contains "$FAIL_SUMMARY" "- Command: \`bash $RUNNER_FAIL\`"
expect_contains "$FAIL_SUMMARY" "- Result: fail"
expect_contains "$FAIL_SUMMARY" "- Exit code: 7"
expect_contains "$FAIL_SUMMARY" "YES MERGE"

echo "==> Case 3: missing runner script fails with exit 2"
set +e
SP_DISCIPLINE_EVIDENCE_DIR="$OUT_DIR" \
SP_DISCIPLINE_GATE_RUNNER="$RUNNER_MISSING" \
SP_DISCIPLINE_EVIDENCE_TS_UTC="20260403T120003Z" \
bash "$CAPTURE_SCRIPT"
MISSING_STATUS=$?
set -e

if [ "$MISSING_STATUS" -ne 2 ]; then
  echo "ERROR: expected exit status 2 for missing runner, got $MISSING_STATUS" >&2
  exit 1
fi

echo "==> Case 4: existing timestamp artifacts fail without overwrite flag"
TS_COLLIDE="20260403T120004Z"
COLLIDE_LOG="$OUT_DIR/sp_discipline_stack_gates_${TS_COLLIDE}.log"
COLLIDE_SUMMARY="$OUT_DIR/sp_discipline_stack_gates_${TS_COLLIDE}.md"
echo "preexisting log" >"$COLLIDE_LOG"
echo "preexisting summary" >"$COLLIDE_SUMMARY"
set +e
SP_DISCIPLINE_EVIDENCE_DIR="$OUT_DIR" \
SP_DISCIPLINE_GATE_RUNNER="$RUNNER_OK" \
SP_DISCIPLINE_EVIDENCE_TS_UTC="$TS_COLLIDE" \
bash "$CAPTURE_SCRIPT"
COLLIDE_STATUS=$?
set -e

if [ "$COLLIDE_STATUS" -ne 2 ]; then
  echo "ERROR: expected collision exit status 2, got $COLLIDE_STATUS" >&2
  exit 1
fi
expect_contains "$COLLIDE_LOG" "preexisting log"
expect_contains "$COLLIDE_SUMMARY" "preexisting summary"

echo "==> Case 5: overwrite flag allows replacing existing timestamp artifacts"
SP_DISCIPLINE_EVIDENCE_DIR="$OUT_DIR" \
SP_DISCIPLINE_GATE_RUNNER="$RUNNER_OK" \
SP_DISCIPLINE_EVIDENCE_TS_UTC="$TS_COLLIDE" \
SP_DISCIPLINE_EVIDENCE_OVERWRITE=1 \
bash "$CAPTURE_SCRIPT"

expect_contains "$COLLIDE_LOG" "runner ok output"
expect_contains "$COLLIDE_SUMMARY" "- Result: pass"

echo "SP discipline evidence-capture scenario tests passed."
