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
  create_issue_and_add_to_project.sh --title "<title>" --body-file <path> [--status "<status>"]

Creates a GitHub issue in the current repo, adds it to the OpenASE Automation
project, and sets project status.

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
EOF
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

issue_url="$(gh issue create --title "$title" --body-file "$body_file")"
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

printf 'ISSUE_URL=%s\nITEM_ID=%s\nSTATUS=%s\n' "$issue_url" "$item_id" "$status_name"
