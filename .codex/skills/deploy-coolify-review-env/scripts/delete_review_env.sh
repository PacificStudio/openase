#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

usage() {
  cat <<'EOF'
Usage:
  delete_review_env.sh --branch <branch> [--env-name <name>] [--app-name <name>]
  delete_review_env.sh --env-name <name> --ticket-identifier <ticket-id>

Examples:
  delete_review_env.sh --branch feature/review-env
  delete_review_env.sh --env-name review --ticket-identifier ASE-184
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

[[ -n "$branch" || -n "$ticket_identifier" || -n "$env_name" ]] || {
  usage >&2
  die "--branch, --ticket-identifier, or --env-name is required"
}
[[ -z "$branch" || -z "$ticket_identifier" ]] || die "--branch and --ticket-identifier are mutually exclusive"

ensure_common_runtime_env

if [[ -n "$ticket_identifier" ]]; then
  if [[ -z "$env_name" ]]; then
    env_name="${COOLIFY_ENVIRONMENT_NAME:-review}"
  fi

  matched_records="$(find_application_records_by_ticket_identifier "$ticket_identifier" "$env_name")"
  deleted_names=()
  deleted_uuids=()

  if [[ -n "$matched_records" ]]; then
    while IFS= read -r record || [[ -n "$record" ]]; do
      [[ -n "$record" ]] || continue
      app_name="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["name"])' <<<"$record")"
      app_uuid="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["uuid"])' <<<"$record")"
      info "deleting application $app_name ($app_uuid) for ticket $ticket_identifier"
      api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
      [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
      deleted_names+=("$app_name")
      deleted_uuids+=("$app_uuid")
    done <<<"$matched_records"
  else
    info "no applications matched ticket identifier $ticket_identifier in environment $env_name"
  fi

  {
    printf 'environment_name=%s\n' "$env_name"
    printf 'ticket_identifier=%s\n' "$ticket_identifier"
    printf 'deleted_application_count=%s\n' "${#deleted_names[@]}"
    printf 'deleted_application_names=%s\n' "$(IFS=,; echo "${deleted_names[*]:-}")"
    printf 'deleted_application_uuids=%s\n' "$(IFS=,; echo "${deleted_uuids[*]:-}")"
  }
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
