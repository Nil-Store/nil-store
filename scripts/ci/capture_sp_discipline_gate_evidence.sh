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

TS_UTC="${SP_DISCIPLINE_EVIDENCE_TS_UTC:-$(date -u +"%Y%m%dT%H%M%SZ")}"
LOG_FILE="$OUT_DIR/sp_discipline_stack_gates_${TS_UTC}.log"
SUMMARY_FILE="$OUT_DIR/sp_discipline_stack_gates_${TS_UTC}.md"
RUN_COMMAND="bash $GATE_RUNNER_SCRIPT"
ALLOW_OVERWRITE="${SP_DISCIPLINE_EVIDENCE_OVERWRITE:-0}"

if [ "$ALLOW_OVERWRITE" != "1" ]; then
  if [ -e "$LOG_FILE" ] || [ -e "$SUMMARY_FILE" ]; then
    echo "ERROR: evidence artifact already exists for timestamp $TS_UTC" >&2
    echo "       log: $LOG_FILE" >&2
    echo "       summary: $SUMMARY_FILE" >&2
    echo "       Set SP_DISCIPLINE_EVIDENCE_OVERWRITE=1 to replace existing artifacts." >&2
    exit 2
  fi
fi

echo "==> Running full SP discipline stack gates..."
echo "    log: $LOG_FILE"
set +e
bash "$GATE_RUNNER_SCRIPT" 2>&1 | tee "$LOG_FILE"
RUN_STATUS=${PIPESTATUS[0]}
set -e

RESULT_WORD="pass"
if [ "$RUN_STATUS" -ne 0 ]; then
  RESULT_WORD="fail"
fi

cat > "$SUMMARY_FILE" <<EOF
# SP Discipline Gate Evidence

- Timestamp (UTC): $TS_UTC
- Command: \`$RUN_COMMAND\`
- Result: $RESULT_WORD
- Exit code: $RUN_STATUS
- Log file: \`$LOG_FILE\`

## Merge Safety Reminder
No PR in this stack may be merged to \`main\` without explicit human approval containing the exact phrase:
\`YES MERGE\`
EOF

echo
echo "==> Evidence summary written:"
echo "    $SUMMARY_FILE"

if [ "$RUN_STATUS" -ne 0 ]; then
  echo "ERROR: SP discipline stack gate run failed (exit $RUN_STATUS)" >&2
  exit "$RUN_STATUS"
fi

echo "SP discipline gate evidence capture completed successfully."
