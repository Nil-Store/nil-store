#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/ci/check_yes_merge.sh --fixture <path>
  scripts/ci/check_yes_merge.sh --repo <owner/repo> --pr-number <num> --token <github_token>

Behavior:
- Passes only if at least one human-authored comment body contains the exact phrase: YES MERGE
- Human-authored means GitHub user type is "User" (bots are ignored)
USAGE
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: required command not found: $cmd" >&2
    exit 2
  fi
}

fixture=""
repo="${GITHUB_REPOSITORY:-}"
pr_number=""
token="${GITHUB_TOKEN:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --fixture)
      fixture="${2:-}"
      shift 2
      ;;
    --repo)
      repo="${2:-}"
      shift 2
      ;;
    --pr-number)
      pr_number="${2:-}"
      shift 2
      ;;
    --token)
      token="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

require_cmd jq

data_file=""
tmpdir=""
cleanup() {
  if [[ -n "$tmpdir" && -d "$tmpdir" ]]; then
    rm -rf "$tmpdir"
  fi
}
trap cleanup EXIT

if [[ -n "$fixture" ]]; then
  if [[ ! -f "$fixture" ]]; then
    echo "ERROR: fixture file not found: $fixture" >&2
    exit 2
  fi
  data_file="$fixture"
else
  if [[ -z "$repo" || -z "$pr_number" || -z "$token" ]]; then
    echo "ERROR: --repo, --pr-number, and --token are required when --fixture is not used" >&2
    usage >&2
    exit 2
  fi
  require_cmd curl

  tmpdir="$(mktemp -d)"
  issue_comments="$tmpdir/issue_comments.json"
  review_comments="$tmpdir/review_comments.json"
  reviews="$tmpdir/reviews.json"
  combined="$tmpdir/combined.json"

  api_base="https://api.github.com/repos/$repo"
  auth_header="Authorization: Bearer $token"

  fetch_paginated() {
    local endpoint="$1"
    local out_file="$2"
    local page=1
    local per_page=100
    local page_file="$tmpdir/page.json"
    local merged_file="$tmpdir/merged.json"
    local page_count=0

    printf '[]\n' >"$out_file"
    while true; do
      curl -fsSL \
        -H "Accept: application/vnd.github+json" \
        -H "$auth_header" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "$api_base/$endpoint?per_page=$per_page&page=$page" \
        -o "$page_file"

      if ! jq -e 'type == "array"' "$page_file" >/dev/null; then
        echo "ERROR: expected array response from GitHub API for $endpoint page $page" >&2
        exit 1
      fi

      jq -s '.[0] + .[1]' "$out_file" "$page_file" >"$merged_file"
      mv "$merged_file" "$out_file"

      page_count="$(jq 'length' "$page_file")"
      if [[ "$page_count" -lt "$per_page" ]]; then
        break
      fi
      page=$((page + 1))
    done
  }

  fetch_paginated "issues/$pr_number/comments" "$issue_comments"
  fetch_paginated "pulls/$pr_number/comments" "$review_comments"
  fetch_paginated "pulls/$pr_number/reviews" "$reviews"

  jq -s 'add' "$issue_comments" "$review_comments" "$reviews" > "$combined"
  data_file="$combined"
fi

phrase_re='(^|[^[:alnum:]_])YES MERGE([^[:alnum:]_]|$)'

count="$(jq -r --arg re "$phrase_re" '
  [ .[]
    | select((.user.type // "") == "User")
    | (.body // "")
    | tostring
    | select(test($re))
  ] | length
' "$data_file")"

if [[ "$count" -gt 0 ]]; then
  echo "PASS: found human-authored YES MERGE approval"
  exit 0
fi

echo "FAIL: missing required human-authored approval phrase: YES MERGE" >&2
exit 1
