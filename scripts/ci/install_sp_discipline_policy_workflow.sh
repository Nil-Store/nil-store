#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

SRC="${SP_POLICY_INSTALL_SRC_PATH:-ci/workflow_templates/sp_discipline_stack_policy.yml}"
DST="${SP_POLICY_INSTALL_DST_PATH:-.github/workflows/sp_discipline_stack_policy.yml}"
MODE="${1:-dry-run}"

if [ ! -f "$SRC" ]; then
  echo "ERROR: workflow template not found: $SRC" >&2
  exit 2
fi

if [ "$MODE" = "--apply" ]; then
  mkdir -p "$(dirname "$DST")"
  cp "$SRC" "$DST"
  echo "Installed workflow template: $SRC -> $DST"
  echo "Reminder: pushing .github/workflows changes requires GitHub token with workflow scope."
  exit 0
fi

if [ "$MODE" = "--dry-run" ] || [ "$MODE" = "dry-run" ]; then
  echo "Dry run: would install $SRC -> $DST"
  echo "Run with --apply to write the workflow file."
  exit 0
fi

echo "ERROR: unknown mode: $MODE (use --dry-run or --apply)" >&2
exit 2
