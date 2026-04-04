#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: required command not found: $cmd" >&2
    exit 2
  fi
}

require_cmd bash
require_cmd ruby

TEMPLATE="${SP_POLICY_TEMPLATE_PATH:-ci/workflow_templates/sp_discipline_stack_policy.yml}"
INSTALLER="${SP_POLICY_INSTALLER_PATH:-scripts/ci/install_sp_discipline_policy_workflow.sh}"

if [ ! -f "$TEMPLATE" ]; then
  echo "ERROR: missing workflow template: $TEMPLATE" >&2
  exit 2
fi

if [ ! -x "$INSTALLER" ]; then
  echo "ERROR: installer script is missing or not executable: $INSTALLER" >&2
  exit 2
fi

ruby -e "require 'yaml'; YAML.load_file('$TEMPLATE')"
bash "$INSTALLER" --dry-run >/dev/null

echo "SP discipline policy template check passed."
