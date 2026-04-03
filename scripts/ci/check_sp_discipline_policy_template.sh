#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

TEMPLATE="ci/workflow_templates/sp_discipline_stack_policy.yml"
INSTALLER="scripts/ci/install_sp_discipline_policy_workflow.sh"

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
