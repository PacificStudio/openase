#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

usage() {
  cat <<'EOF'
Usage:
  delete_review_env.sh --ticket-identifier <ticket-id> --env-name <name>
  delete_review_env.sh --branch <branch> [--env-name <name>] [--app-name <name>]

Examples:
  delete_review_env.sh --env-name review --ticket-identifier ASE-123
  delete_review_env.sh --branch feature/review-env
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

[[ -n "$branch" || -n "$env_name" || -n "$ticket_identifier" ]] || {
  usage >&2
  die "--ticket-identifier, --branch, or --env-name is required"
}

ensure_common_runtime_env

if [[ -n "$ticket_identifier" ]]; then
  [[ -z "$branch" ]] || die "--branch cannot be combined with --ticket-identifier"
  [[ -z "$app_name" ]] || die "--app-name cannot be combined with --ticket-identifier"

  if [[ -z "$env_name" ]]; then
    env_name="${COOLIFY_ENVIRONMENT_NAME:-}"
  fi
  [[ -n "$env_name" ]] || die "--env-name is required when using --ticket-identifier unless COOLIFY_ENVIRONMENT_NAME is set"

  ticket_slug="$(slugify_name "$ticket_identifier")"
  api_request GET "/api/v1/projects/$COOLIFY_PROJECT_UUID/$env_name"
  case "$API_STATUS" in
    200)
      ;;
    404)
      info "environment $env_name is already absent"
      cat <<EOF
ticket_identifier=$ticket_identifier
environment_name=$env_name
matched_application_count=0
deleted_application_names=
deleted_application_uuids=
EOF
      exit 0
      ;;
    *)
      die "failed to inspect environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac

  matches="$(
    python3 - "$ticket_slug" "$API_BODY" <<'PY'
import json
import re
import sys

ticket_slug = sys.argv[1]
payload = json.loads(sys.argv[2])
pattern = re.compile(rf"(^|[^a-z0-9]){re.escape(ticket_slug)}([^a-z0-9]|$)")

for app in payload.get("applications", []):
    haystacks = [
        app.get("name", ""),
        app.get("git_branch", ""),
        app.get("description", ""),
        app.get("fqdn", ""),
    ]
    normalized = [value.lower() for value in haystacks if value]
    if any(pattern.search(value) for value in normalized):
        print(f"{app.get('name', '')}\t{app.get('uuid', '')}")
PY
  )"

  deleted_names=()
  deleted_uuids=()

  if [[ -z "$matches" ]]; then
    info "application for ticket $ticket_identifier is already absent in environment $env_name"
  else
    while IFS=$'\t' read -r matched_name matched_uuid; do
      [[ -n "$matched_name" && -n "$matched_uuid" ]] || continue
      info "deleting application $matched_name ($matched_uuid) for ticket $ticket_identifier"
      api_request DELETE "/api/v1/applications/$matched_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
      [[ "$API_STATUS" == "200" ]] || die "failed to delete application $matched_uuid: HTTP $API_STATUS: $API_BODY"
      deleted_names+=("$matched_name")
      deleted_uuids+=("$matched_uuid")
    done <<<"$matches"
  fi

  deleted_names_csv="$(IFS=,; printf '%s' "${deleted_names[*]-}")"
  deleted_uuids_csv="$(IFS=,; printf '%s' "${deleted_uuids[*]-}")"
  cat <<EOF
ticket_identifier=$ticket_identifier
environment_name=$env_name
matched_application_count=${#deleted_uuids[@]}
deleted_application_names=$deleted_names_csv
deleted_application_uuids=$deleted_uuids_csv
EOF
  exit 0
fi

if [[ -z "$env_name" ]]; then
  env_name="$(derive_env_name "$branch")"
fi
if [[ -z "$app_name" ]]; then
  app_name="$(derive_app_name "$branch")"
fi

app_uuid="$(find_application_uuid_by_name "$app_name")"
if [[ -n "$app_uuid" ]]; then
  info "deleting application $app_name ($app_uuid)"
  api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
  [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
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
