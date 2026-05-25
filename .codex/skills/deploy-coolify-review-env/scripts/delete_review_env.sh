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
  die "--branch, --env-name, or --ticket-identifier is required"
}

ensure_common_runtime_env

if [[ -z "$env_name" ]]; then
  if [[ -n "$ticket_identifier" && -n "${COOLIFY_ENVIRONMENT_NAME:-}" ]]; then
    env_name="$COOLIFY_ENVIRONMENT_NAME"
  fi
fi
if [[ -z "$env_name" ]]; then
  env_name="$(derive_env_name "$branch")"
fi
if [[ -z "$ticket_identifier" && -z "$app_name" ]]; then
  app_name="$(derive_app_name "$branch")"
fi

app_uuid=""
result="absent"
matched_applications=""
environment_result="skipped"

if [[ -n "$ticket_identifier" ]]; then
  matches_file="$(mktemp)"
  trap 'rm -f "$matches_file"' EXIT
  find_applications_by_ticket_identifier "$env_name" "$ticket_identifier" >"$matches_file"
  match_count="$(wc -l <"$matches_file" | tr -d ' ')"
  if [[ "$match_count" == "0" ]]; then
    info "application for ticket $ticket_identifier is already absent from $env_name"
  elif [[ "$match_count" != "1" ]]; then
    result="ambiguous"
    matched_applications="$(python3 - "$matches_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    names = [json.loads(line).get("name", "") for line in fh if line.strip()]
print(",".join(name for name in names if name))
PY
)"
    info "ticket $ticket_identifier matched multiple applications in $env_name: ${matched_applications:-unknown}"
  else
    app_name="$(python3 - "$matches_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    item = json.loads(next(line for line in fh if line.strip()))
print(item.get("name", ""))
PY
)"
    app_uuid="$(python3 - "$matches_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    item = json.loads(next(line for line in fh if line.strip()))
print(item.get("uuid", ""))
PY
)"
    info "deleting application $app_name ($app_uuid) for ticket $ticket_identifier"
    api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
    [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
    result="deleted"
  fi
else
  app_uuid="$(find_application_uuid_by_name "$app_name")"
  if [[ -n "$app_uuid" ]]; then
    info "deleting application $app_name ($app_uuid)"
    api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
    [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
    result="deleted"
  else
    info "application $app_name is already absent"
  fi
fi

if [[ -z "$ticket_identifier" && "${COOLIFY_ENVIRONMENT_NAME:-}" != "$env_name" ]]; then
  api_request DELETE "/api/v1/projects/$COOLIFY_PROJECT_UUID/environments/$env_name"
  case "$API_STATUS" in
    200)
      info "deleted environment $env_name"
      environment_result="deleted"
      ;;
    404|422)
      info "environment $env_name is already absent or not empty"
      environment_result="absent-or-not-empty"
      ;;
    *)
      die "failed to delete environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac
else
  info "keeping shared environment $env_name"
fi

cat <<EOF
environment_name=$env_name
ticket_identifier=$ticket_identifier
application_name=$app_name
application_uuid=$app_uuid
result=$result
environment_result=$environment_result
matched_applications=$matched_applications
EOF
