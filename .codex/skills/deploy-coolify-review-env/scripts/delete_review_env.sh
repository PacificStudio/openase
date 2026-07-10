#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

usage() {
  cat <<'EOF'
Usage:
  delete_review_env.sh --branch <branch> [--env-name <name>] [--app-name <name>]
  delete_review_env.sh --env-name <name> --ticket-identifier <ticket>

Examples:
  delete_review_env.sh --branch feature/review-env
  delete_review_env.sh --env-name review --ticket-identifier ASE-123
EOF
}

branch=""
env_name=""
app_name=""
ticket_identifier=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      branch="${2:-}"
      shift 2
      ;;
    --env-name)
      env_name="${2:-}"
      shift 2
      ;;
    --app-name)
      app_name="${2:-}"
      shift 2
      ;;
    --ticket-identifier)
      ticket_identifier="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

if [[ -n "$ticket_identifier" && ( -n "$branch" || -n "$app_name" ) ]]; then
  die "--ticket-identifier cannot be combined with --branch or --app-name"
fi

[[ -n "$branch" || -n "$env_name" || -n "$ticket_identifier" ]] || {
  usage >&2
  die "--branch, --env-name, or --ticket-identifier is required"
}

ensure_common_runtime_env

if [[ -z "$env_name" ]]; then
  env_name="$(derive_env_name "${branch:-$ticket_identifier}")"
fi
if [[ -z "$app_name" && -n "$branch" ]]; then
  app_name="$(derive_app_name "$branch")"
fi

delete_application() {
  local target_app_name="$1"
  local target_app_uuid="$2"

  info "deleting application $target_app_name ($target_app_uuid)"
  api_request DELETE "/api/v1/applications/$target_app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
  [[ "$API_STATUS" == "200" ]] || die "failed to delete application $target_app_uuid: HTTP $API_STATUS: $API_BODY"
}

if [[ -n "$ticket_identifier" ]]; then
  cache_key="$(printf '%s_%s\n' "$COOLIFY_PROJECT_UUID" "$env_name" | tr -c '[:alnum:]' '_')"
  env_cache_file="/tmp/deploy-coolify-review-env-${cache_key}.json"

  if [[ -f "$env_cache_file" ]]; then
    API_STATUS="200"
    API_BODY="$(<"$env_cache_file")"
  else
    api_request GET "/api/v1/projects/$COOLIFY_PROJECT_UUID/$env_name"
    if [[ "$API_STATUS" == "200" ]]; then
      printf '%s' "$API_BODY" >"$env_cache_file"
    fi
  fi

  case "$API_STATUS" in
    200)
      ;;
    404)
      info "environment $env_name is already absent"
      cat <<EOF
environment_name=$env_name
ticket_identifier=$ticket_identifier
result=already_absent
application_name=
application_uuid=
EOF
      exit 0
      ;;
    *)
      die "failed to inspect environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac

  mapped_lines="$(
    TARGET_TICKET_IDENTIFIER="$ticket_identifier" python3 -c '
import json
import os
import re
import sys

ticket_identifier = os.environ["TARGET_TICKET_IDENTIFIER"]
data = json.load(sys.stdin)
applications = data.get("applications", [])

def normalize(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    value = re.sub(r"-{2,}", "-", value).strip("-")
    return value

ticket_slug = normalize(ticket_identifier)
pattern = re.compile(r"(^|-){}(-|$)".format(re.escape(ticket_slug)))

for item in applications:
    haystacks = [
        item.get("name", ""),
        item.get("git_branch", ""),
        item.get("description", ""),
        item.get("fqdn", ""),
    ]
    normalized = [normalize(value) for value in haystacks if value]
    if any(pattern.search(value) for value in normalized):
        print("{}\t{}".format(item.get("name", ""), item.get("uuid", "")))
' <<<"$API_BODY"
  )"

  mapfile -t mapped_entries <<<"$mapped_lines"
  mapped_count=0
  if [[ -n "$mapped_lines" ]]; then
    mapped_count="${#mapped_entries[@]}"
  fi

  case "$mapped_count" in
    0)
      info "no application mapped to ticket $ticket_identifier in environment $env_name"
      cat <<EOF
environment_name=$env_name
ticket_identifier=$ticket_identifier
result=already_absent
application_name=
application_uuid=
EOF
      ;;
    1)
      IFS=$'\t' read -r mapped_name mapped_uuid <<<"${mapped_entries[0]}"
      delete_application "$mapped_name" "$mapped_uuid"
      rm -f "$env_cache_file"
      cat <<EOF
environment_name=$env_name
ticket_identifier=$ticket_identifier
result=deleted
application_name=$mapped_name
application_uuid=$mapped_uuid
EOF
      ;;
    *)
      printf 'environment_name=%s\n' "$env_name"
      printf 'ticket_identifier=%s\n' "$ticket_identifier"
      printf 'result=ambiguous\n'
      printf 'application_name='
      printf '%s\n' "$(printf '%s\n' "${mapped_entries[@]}" | cut -f1 | paste -sd, -)"
      printf 'application_uuid='
      printf '%s\n' "$(printf '%s\n' "${mapped_entries[@]}" | cut -f2 | paste -sd, -)"
      ;;
  esac
  exit 0
fi

app_uuid="$(find_application_uuid_by_name "$app_name")"
if [[ -n "$app_uuid" ]]; then
  delete_application "$app_name" "$app_uuid"
else
  info "application $app_name is already absent"
fi

api_request DELETE "/api/v1/projects/$COOLIFY_PROJECT_UUID/environments/$env_name"
case "$API_STATUS" in
  200)
    info "deleted environment $env_name"
    ;;
  404|422)
    info "environment $env_name is already absent or not empty"
    ;;
  *)
    die "failed to delete environment $env_name: HTTP $API_STATUS: $API_BODY"
    ;;
esac

cat <<EOF
environment_name=$env_name
application_name=$app_name
application_uuid=$app_uuid
EOF
