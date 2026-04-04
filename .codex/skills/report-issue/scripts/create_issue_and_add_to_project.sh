#!/usr/bin/env bash
set -euo pipefail

owner="BetterAndBetterII"
project_number="2"
project_id="PVT_kwHOCG1pys4BSO3u"
status_field_id="PVTSSF_lAHOCG1pys4BSO3uzg_09GU"
todo_option_id="1d2e5cb6"

title=""
body_file=""
status_name="Todo"
declare -a blocked_by_issue_numbers=()

status_option_id_for() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    backlog)
      printf '%s\n' "800c1928"
      ;;
    todo)
      printf '%s\n' "1d2e5cb6"
      ;;
    "in progress"|"in-progress"|in_progress)
      printf '%s\n' "e210014d"
      ;;
    rework)
      printf '%s\n' "af10a761"
      ;;
    "in review"|"in-review"|in_review)
      printf '%s\n' "26375592"
      ;;
    merging)
      printf '%s\n' "b00ec75e"
      ;;
    done)
      printf '%s\n' "6ff964cb"
      ;;
    canceled|cancelled)
      printf '%s\n' "9427733c"
      ;;
    duplicated)
      printf '%s\n' "1be5d09e"
      ;;
    *)
      return 1
      ;;
  esac
}

usage() {
  cat <<'EOF'
Usage:
  create_issue_and_add_to_project.sh --title "<title>" --body-file <path> [--status "<status>"] [--blocked-by <issue-number> ...]

Creates a GitHub issue in the current repo, optionally wires blocked-by issue
dependencies, then adds it to the OpenASE Automation project and sets project
status.

Supported statuses:
  Backlog
  Todo
  In Progress
  Rework
  In Review
  Merging
  Done
  Canceled
  Duplicated

Defaults to `Todo` when --status is omitted.

Examples:
  create_issue_and_add_to_project.sh --title "Example" --body-file /tmp/body.md
  create_issue_and_add_to_project.sh --title "Example" --body-file /tmp/body.md --blocked-by 322 --blocked-by 323
EOF
}

repo_slug() {
  gh repo view --json nameWithOwner -q .nameWithOwner
}

issue_node_id_for() {
  local issue_number="$1"
  local repo_full_name="$2"

  gh issue view "$issue_number" --repo "$repo_full_name" --json id -q .id
}

issue_number_from_url() {
  local issue_url="$1"
  printf '%s\n' "${issue_url##*/}"
}

add_blocked_by_dependency() {
  local issue_id="$1"
  local blocking_issue_id="$2"

  gh api graphql -f query='
mutation($issueId: ID!, $blockingIssueId: ID!) {
  addBlockedBy(input: {issueId: $issueId, blockingIssueId: $blockingIssueId}) {
    issue { id }
  }
}
' -F issueId="$issue_id" -F blockingIssueId="$blocking_issue_id" >/dev/null
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --title)
      title="${2:-}"
      shift 2
      ;;
    --body-file)
      body_file="${2:-}"
      shift 2
      ;;
    --status)
      status_name="${2:-}"
      shift 2
      ;;
    --blocked-by)
      blocked_by_issue_numbers+=("${2:-}")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$title" ]]; then
  echo "--title is required" >&2
  exit 1
fi

if [[ -z "$body_file" ]]; then
  echo "--body-file is required" >&2
  exit 1
fi

if [[ ! -f "$body_file" ]]; then
  echo "body file not found: $body_file" >&2
  exit 1
fi

if ! status_option_id="$(status_option_id_for "$status_name")"; then
  echo "unsupported --status: $status_name" >&2
  usage >&2
  exit 1
fi

for blocked_by_issue_number in "${blocked_by_issue_numbers[@]}"; do
  if [[ ! "$blocked_by_issue_number" =~ ^[0-9]+$ ]]; then
    echo "invalid --blocked-by issue number: $blocked_by_issue_number" >&2
    exit 1
  fi
done

repo_full_name="$(repo_slug)"
issue_url="$(gh issue create --title "$title" --body-file "$body_file")"
issue_number="$(issue_number_from_url "$issue_url")"
issue_id="$(issue_node_id_for "$issue_number" "$repo_full_name")"

for blocked_by_issue_number in "${blocked_by_issue_numbers[@]}"; do
  blocking_issue_id="$(issue_node_id_for "$blocked_by_issue_number" "$repo_full_name")"
  add_blocked_by_dependency "$issue_id" "$blocking_issue_id"
done

item_id="$(
  gh project item-add "$project_number" --owner "$owner" --url "$issue_url" --format json |
    python3 -c 'import json, sys; print(json.load(sys.stdin)["id"])'
)"
gh project item-edit \
  --project-id "$project_id" \
  --id "$item_id" \
  --field-id "$status_field_id" \
  --single-select-option-id "$status_option_id" \
  >/dev/null

printf 'ISSUE_URL=%s\nISSUE_NUMBER=%s\nITEM_ID=%s\nSTATUS=%s\n' \
  "$issue_url" "$issue_number" "$item_id" "$status_name"
