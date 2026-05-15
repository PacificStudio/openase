#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

usage() {
  cat <<'EOF'
Usage:
  delete_review_env.sh --branch <branch> [--env-name <name>] [--app-name <name>]
  delete_review_env.sh --env-name <name> --ticket-identifier <ticket-identifier>

Examples:
  delete_review_env.sh --branch feature/review-env
  delete_review_env.sh --env-name review --ticket-identifier ASE-123
EOF
}

branch=""
env_name=""
app_name=""
ticket_identifier=""
env_name_explicit=false

list_ticket_scoped_apps() {
  local env_name="$1"
  local ticket_identifier="$2"

  api_request GET "/api/v1/projects/$COOLIFY_PROJECT_UUID/$env_name"
  case "$API_STATUS" in
    200) ;;
    404)
      return 0
      ;;
    *)
      die "failed to inspect environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac

  python3 -c '
import json
import re
import sys

ticket = sys.argv[1].strip().lower()
ticket = re.sub(r"[^a-z0-9]+", "-", ticket).strip("-")
if not ticket:
    sys.exit(0)

pattern = re.compile(rf"(^|[^a-z0-9]){re.escape(ticket)}([^a-z0-9]|$)")
environment = json.load(sys.stdin)

for app in environment.get("applications", []):
    haystacks = [
        str(app.get("name") or ""),
        str(app.get("git_branch") or ""),
        str(app.get("fqdn") or ""),
        str(app.get("description") or ""),
    ]
    if any(pattern.search(item.lower()) for item in haystacks):
        print("%s\t%s" % (app.get("uuid", ""), app.get("name", "")))
' "$ticket_identifier" <<<"$API_BODY"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      branch="${2:-}"
      shift 2
      ;;
    --env-name)
      env_name="${2:-}"
      env_name_explicit=true
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

[[ -n "$branch" || -n "$ticket_identifier" || -n "$env_name" ]] || {
  usage >&2
  die "--branch, --ticket-identifier, or --env-name is required"
}
[[ -z "$branch" || -z "$ticket_identifier" ]] || die "--branch and --ticket-identifier are mutually exclusive"

ensure_common_runtime_env

if [[ -z "$env_name" && -n "$branch" ]]; then
  env_name="$(derive_env_name "$branch")"
fi
if [[ -z "$env_name" && -n "$ticket_identifier" ]]; then
  if [[ -n "${COOLIFY_ENVIRONMENT_NAME:-}" ]]; then
    env_name="$COOLIFY_ENVIRONMENT_NAME"
  else
    die "--env-name is required with --ticket-identifier when COOLIFY_ENVIRONMENT_NAME is not configured"
  fi
fi
if [[ -z "$app_name" && -n "$branch" ]]; then
  app_name="$(derive_app_name "$branch")"
fi

if [[ -n "$ticket_identifier" ]]; then
  mapfile -t matches < <(list_ticket_scoped_apps "$env_name" "$ticket_identifier")
  deleted_names=()
  deleted_uuids=()

  for match in "${matches[@]}"; do
    [[ -n "$match" ]] || continue
    app_uuid="${match%%$'\t'*}"
    app_name="${match#*$'\t'}"
    [[ -n "$app_uuid" ]] || continue
    info "deleting application $app_name ($app_uuid) for ticket $ticket_identifier"
    api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
    [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
    deleted_names+=("$app_name")
    deleted_uuids+=("$app_uuid")
  done

  if [[ "${#deleted_names[@]}" -eq 0 ]]; then
    info "no applications matched ticket $ticket_identifier in environment $env_name"
  fi

  (
    IFS=,
    cat <<EOF
environment_name=$env_name
ticket_identifier=$ticket_identifier
deleted_application_count=${#deleted_names[@]}
deleted_application_names=${deleted_names[*]}
deleted_application_uuids=${deleted_uuids[*]}
EOF
  )
else
  app_uuid="$(find_application_uuid_by_name "$app_name")"
  if [[ -n "$app_uuid" ]]; then
    info "deleting application $app_name ($app_uuid)"
    api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
    [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
  else
    info "application $app_name is already absent"
  fi

  if [[ "$env_name_explicit" != "true" ]]; then
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
  else
    info "leaving shared environment $env_name intact"
  fi

  cat <<EOF
environment_name=$env_name
application_name=$app_name
application_uuid=$app_uuid
EOF
fi
