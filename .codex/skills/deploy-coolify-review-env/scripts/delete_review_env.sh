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
  delete_review_env.sh --env-name review --ticket-identifier ASE-623
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
[[ -z "$ticket_identifier" || (-z "$branch" && -z "$app_name") ]] || die "--ticket-identifier cannot be combined with --branch or --app-name"

ensure_common_runtime_env

if [[ -z "$env_name" ]]; then
  if [[ -n "$ticket_identifier" && -n "${COOLIFY_ENVIRONMENT_NAME:-}" ]]; then
    env_name="$COOLIFY_ENVIRONMENT_NAME"
  fi
fi
if [[ -z "$env_name" ]]; then
  env_name="$(derive_env_name "$branch")"
fi
if [[ -n "$ticket_identifier" ]]; then
  api_request GET "/api/v1/projects/$COOLIFY_PROJECT_UUID/$env_name"
  case "$API_STATUS" in
    200)
      match_payload="$(
        TICKET_IDENTIFIER="$ticket_identifier" API_BODY_JSON="$API_BODY" python3 - <<'PY'
import json
import os
import re

ticket_identifier = os.environ["TICKET_IDENTIFIER"].strip().lower()
token = re.escape(ticket_identifier)
pattern = re.compile(rf"(^|[^a-z0-9]){token}($|[^a-z0-9])")

payload = json.loads(os.environ["API_BODY_JSON"])
matches = []
for item in payload.get("applications", []):
    fields = [
        item.get("name", ""),
        item.get("git_branch", ""),
        item.get("fqdn", ""),
        item.get("description", ""),
    ]
    haystacks = [field.strip().lower() for field in fields if isinstance(field, str)]
    if any(pattern.search(haystack) for haystack in haystacks):
        matches.append(
            {
                "name": item.get("name", ""),
                "uuid": item.get("uuid", ""),
                "git_branch": item.get("git_branch", ""),
                "fqdn": item.get("fqdn", ""),
            }
        )
print(json.dumps({"matches": matches}, ensure_ascii=True))
PY
      )"
      ;;
    404)
      match_payload='{"matches":[]}'
      ;;
    *)
      die "failed to inspect environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac

  match_count="$(
    python3 -c 'import json,sys; print(len(json.load(sys.stdin)["matches"]))' <<<"$match_payload"
  )"
  if [[ "$match_count" == "0" ]]; then
    info "application for ticket $ticket_identifier is already absent"
    cat <<EOF
ticket_identifier=$ticket_identifier
match_state=already_absent
environment_name=$env_name
application_name=
application_uuid=
environment_action=skipped_shared_env
matched_candidates=
EOF
    exit 0
  fi
  if [[ "$match_count" != "1" ]]; then
    matched_candidates="$(
      python3 -c 'import json,sys; print(",".join(item["name"] for item in json.load(sys.stdin)["matches"]))' <<<"$match_payload"
    )"
    info "ambiguous application mapping for ticket $ticket_identifier: $matched_candidates"
    cat <<EOF
ticket_identifier=$ticket_identifier
match_state=ambiguous
environment_name=$env_name
application_name=
application_uuid=
environment_action=skipped_shared_env
matched_candidates=$matched_candidates
EOF
    exit 0
  fi
  app_name="$(
    python3 -c 'import json,sys; print(json.load(sys.stdin)["matches"][0]["name"])' <<<"$match_payload"
  )"
  app_uuid="$(
    python3 -c 'import json,sys; print(json.load(sys.stdin)["matches"][0]["uuid"])' <<<"$match_payload"
  )"
else
  app_uuid=""
fi
if [[ -z "$app_name" ]]; then
  app_name="$(derive_app_name "$branch")"
fi

if [[ -z "$app_uuid" ]]; then
  app_uuid="$(find_application_uuid_by_name "$app_name")"
fi
if [[ -n "$app_uuid" ]]; then
  info "deleting application $app_name ($app_uuid)"
  api_request DELETE "/api/v1/applications/$app_uuid?delete_configurations=true&delete_volumes=true&docker_cleanup=true&delete_connected_networks=true"
  [[ "$API_STATUS" == "200" ]] || die "failed to delete application $app_uuid: HTTP $API_STATUS: $API_BODY"
  match_state="deleted"
else
  info "application $app_name is already absent"
  match_state="already_absent"
fi

environment_action="deleted_if_empty"
if [[ -n "${COOLIFY_ENVIRONMENT_NAME:-}" || -n "$ticket_identifier" ]]; then
  info "skipping environment deletion for shared environment $env_name"
  environment_action="skipped_shared_env"
else
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
fi

cat <<EOF
ticket_identifier=$ticket_identifier
match_state=$match_state
environment_name=$env_name
application_name=$app_name
application_uuid=$app_uuid
environment_action=$environment_action
matched_candidates=
EOF
