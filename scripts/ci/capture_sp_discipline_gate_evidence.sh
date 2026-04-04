#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${SP_DISCIPLINE_EVIDENCE_DIR:-$ROOT_DIR/_artifacts/ci}"
mkdir -p "$OUT_DIR"

GATE_RUNNER_SCRIPT="${SP_DISCIPLINE_GATE_RUNNER:-scripts/ci/run_sp_discipline_stack_gates.sh}"
if [ ! -f "$GATE_RUNNER_SCRIPT" ]; then
  echo "ERROR: missing gate runner script: $GATE_RUNNER_SCRIPT" >&2
  exit 2
fi

TS_UTC="$(date -u +"%Y%m%dT%H%M%SZ")"
LOG_FILE="$OUT_DIR/sp_discipline_stack_gates_${TS_UTC}.log"
SUMMARY_FILE="$OUT_DIR/sp_discipline_stack_gates_${TS_UTC}.md"
printf -v RUN_COMMAND 'bash %q' "$GATE_RUNNER_SCRIPT"

echo "==> Running full SP discipline stack gates..."
echo "    log: $LOG_FILE"
set +e
bash "$GATE_RUNNER_SCRIPT" 2>&1 | tee "$LOG_FILE"
PIPELINE_STATUS=("${PIPESTATUS[@]}")
set -e
RUN_STATUS=${PIPELINE_STATUS[0]:-1}
TEE_STATUS=${PIPELINE_STATUS[1]:-1}

if [ "$TEE_STATUS" -ne 0 ]; then
  echo "ERROR: failed to write SP discipline gate log artifact: $LOG_FILE (tee exit $TEE_STATUS)" >&2
  exit "$TEE_STATUS"
fi

RESULT_WORD="pass"
if [ "$RUN_STATUS" -ne 0 ]; then
  RESULT_WORD="fail"
fi

cat > "$SUMMARY_FILE" <<EOF_SUMMARY
# SP Discipline Gate Evidence

- Timestamp (UTC): $TS_UTC
- Command: \`$RUN_COMMAND\`
- Result: $RESULT_WORD
- Exit code: $RUN_STATUS
- Log file: \`$LOG_FILE\`

## Merge Safety Reminder
No PR in this stack may be merged to \`main\` without explicit human approval containing the exact phrase:
\`YES MERGE\`
EOF_SUMMARY

echo
echo "==> Evidence summary written:"
echo "    $SUMMARY_FILE"

if [ "$RUN_STATUS" -ne 0 ]; then
  echo "ERROR: SP discipline stack gate run failed (exit $RUN_STATUS)" >&2
  exit "$RUN_STATUS"
fi

echo "SP discipline gate evidence capture completed successfully."
