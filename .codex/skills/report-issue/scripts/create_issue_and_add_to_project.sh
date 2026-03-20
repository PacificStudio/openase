#!/usr/bin/env bash
set -euo pipefail

owner="BetterAndBetterII"
project_number="2"
project_id="PVT_kwHOCG1pys4BSO3u"
status_field_id="PVTSSF_lAHOCG1pys4BSO3uzg_09GU"
todo_option_id="1d2e5cb6"

title=""
body_file=""

usage() {
  cat <<'EOF'
Usage:
  create_issue_and_add_to_project.sh --title "<title>" --body-file <path>

Creates a GitHub issue in the current repo, adds it to the OpenASE Automation
project, and sets project status to Todo.
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

issue_url="$(gh issue create --title "$title" --body-file "$body_file")"
item_id="$(
  gh project item-add "$project_number" --owner "$owner" --url "$issue_url" --format json |
    python3 -c 'import json, sys; print(json.load(sys.stdin)["id"])'
)"
gh project item-edit \
  --project-id "$project_id" \
  --id "$item_id" \
  --field-id "$status_field_id" \
  --single-select-option-id "$todo_option_id" \
  >/dev/null

printf 'ISSUE_URL=%s\nITEM_ID=%s\n' "$issue_url" "$item_id"
