#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

TEMPLATE="ci/workflow_templates/sp_discipline_stack_policy.yml"
INSTALLED=".github/workflows/sp_discipline_stack_policy.yml"
REQUIRE_INSTALLED="0"

if [ "${1:-}" = "--require-installed" ]; then
  REQUIRE_INSTALLED="1"
elif [ -n "${1:-}" ]; then
  echo "ERROR: unknown option: $1 (use --require-installed or no args)" >&2
  exit 2
fi

if [ ! -f "$TEMPLATE" ]; then
  echo "ERROR: missing workflow template: $TEMPLATE" >&2
  exit 2
fi

ruby -e "require 'yaml'; YAML.load_file('$TEMPLATE')"

if [ ! -f "$INSTALLED" ]; then
  if [ "$REQUIRE_INSTALLED" = "1" ]; then
    echo "ERROR: policy workflow is not installed at $INSTALLED" >&2
    echo "Install it with: bash scripts/ci/install_sp_discipline_policy_workflow.sh --apply" >&2
    exit 1
  fi
  echo "INFO: policy workflow is not installed at $INSTALLED (template-only mode)." >&2
  echo "      To install: bash scripts/ci/install_sp_discipline_policy_workflow.sh --apply" >&2
  echo "SP discipline policy install-state check passed (template mode)."
  exit 0
fi

ruby -e "require 'yaml'; YAML.load_file('$INSTALLED')"

if ! cmp -s "$TEMPLATE" "$INSTALLED"; then
  echo "ERROR: installed workflow differs from template." >&2
  echo "Reinstall with: bash scripts/ci/install_sp_discipline_policy_workflow.sh --apply" >&2
  exit 1
fi

echo "SP discipline policy install-state check passed (installed and in sync)."
