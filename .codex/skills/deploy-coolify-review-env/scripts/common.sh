#!/usr/bin/env bash
set -euo pipefail

skill_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
env_file="$skill_dir/.env"

die() {
  echo "deploy-coolify-review-env: $*" >&2
  exit 1
}

info() {
  echo "[deploy-coolify-review-env] $*" >&2
}

load_env_file() {
  local path="$1"
  local line
  local key
  local value

  [[ -f "$path" ]] || return 0

  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    [[ "$line" == *=* ]] || continue
    key="${line%%=*}"
    value="${line#*=}"
    key="${key#"${key%%[![:space:]]*}"}"
    key="${key%"${key##*[![:space:]]}"}"
    [[ -n "$key" ]] || continue
    if [[ "$value" =~ ^\".*\"$ || "$value" =~ ^\'.*\'$ ]]; then
      value="${value:1:${#value}-2}"
    fi
    # Let explicit runtime env override the skill-local .env.
    if [[ -z "${!key:-}" ]]; then
      export "$key=$value"
    fi
  done < "$path"
}

require_env() {
  local name
  for name in "$@"; do
    [[ -n "${!name:-}" ]] || die "required environment variable $name is missing"
  done
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

json_get() {
  local path="$1"
  python3 -c '
import json
import sys

path = sys.argv[1]
data = json.load(sys.stdin)
cur = data
for part in path.split("."):
    if not part:
        continue
    if isinstance(cur, list):
        cur = cur[int(part)]
    else:
        cur = cur.get(part)
    if cur is None:
        break
if cur is None:
    sys.exit(1)
if isinstance(cur, (dict, list)):
    print(json.dumps(cur, ensure_ascii=True))
else:
    print(cur)
' "$path"
}

slugify_name() {
  local raw="$1"
  local max_len="${COOLIFY_NAME_MAX_LENGTH:-48}"
  python3 - "$raw" "$max_len" <<'PY'
import re
import sys

raw = sys.argv[1].strip().lower()
max_len = int(sys.argv[2])
slug = re.sub(r"[^a-z0-9]+", "-", raw).strip("-")
slug = re.sub(r"-{2,}", "-", slug)
if not slug:
    slug = "preview"
print(slug[:max_len].rstrip("-") or "preview")
PY
}

derive_env_name() {
  local branch="$1"
  if [[ -n "${COOLIFY_ENVIRONMENT_NAME:-}" ]]; then
    printf '%s\n' "$COOLIFY_ENVIRONMENT_NAME"
    return 0
  fi
  local prefix="${COOLIFY_ENV_PREFIX:-review}"
  slugify_name "${prefix}-${branch}"
}

derive_app_name() {
  local branch="$1"
  local prefix="${COOLIFY_ENV_PREFIX:-review}"
  slugify_name "${prefix}-${branch}"
}

normalize_ticket_identifier() {
  local ticket_identifier="$1"
  slugify_name "$ticket_identifier"
}

interpolate_name_template() {
  local template="$1"
  local name="$2"
  python3 - "$template" "$name" <<'PY'
import sys

template = sys.argv[1]
name = sys.argv[2]
print(template.replace("{{name}}", name))
PY
}

api_request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local url
  local response
  local status

  require_env COOLIFY_BASE_URL COOLIFY_API_TOKEN
  url="$(trim_trailing_slash "$COOLIFY_BASE_URL")${path}"

  if [[ -n "$body" ]]; then
    response="$(
      curl -sS \
        -X "$method" \
        -H "Authorization: Bearer $COOLIFY_API_TOKEN" \
        -H "Accept: application/json" \
        -H "Content-Type: application/json" \
        --data "$body" \
        -w $'\n__STATUS__:%{http_code}' \
        "$url"
    )"
  else
    response="$(
      curl -sS \
        -X "$method" \
        -H "Authorization: Bearer $COOLIFY_API_TOKEN" \
        -H "Accept: application/json" \
        -w $'\n__STATUS__:%{http_code}' \
        "$url"
    )"
  fi

  status="${response##*$'\n'__STATUS__:}"
  API_STATUS="$status"
  API_BODY="${response%$'\n'__STATUS__:*}"
}

ensure_common_runtime_env() {
  require_env \
    COOLIFY_BASE_URL \
    COOLIFY_API_TOKEN \
    COOLIFY_PROJECT_UUID \
    COOLIFY_SERVER_UUID \
    COOLIFY_DESTINATION_UUID \
    COOLIFY_GIT_REPOSITORY \
    COOLIFY_PORTS_EXPOSES

  case "${COOLIFY_REPOSITORY_MODE:-public}" in
    public) ;;
    private-github-app)
      require_env COOLIFY_GITHUB_APP_UUID
      ;;
    private-deploy-key)
      require_env COOLIFY_PRIVATE_KEY_UUID
      ;;
    *)
      die "unsupported COOLIFY_REPOSITORY_MODE=${COOLIFY_REPOSITORY_MODE:-}"
      ;;
  esac
}

ensure_environment_exists() {
  local env_name="$1"

  api_request GET "/api/v1/projects/$COOLIFY_PROJECT_UUID/$env_name"
  case "$API_STATUS" in
    200)
      info "Coolify environment $env_name already exists"
      ;;
    404)
      info "creating Coolify environment $env_name"
      api_request POST "/api/v1/projects/$COOLIFY_PROJECT_UUID/environments" "{\"name\":\"$env_name\"}"
      [[ "$API_STATUS" == "201" ]] || die "failed to create environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
    *)
      die "failed to inspect environment $env_name: HTTP $API_STATUS: $API_BODY"
      ;;
  esac
}

find_application_uuid_by_name() {
  local app_name="$1"

  api_request GET "/api/v1/servers/$COOLIFY_SERVER_UUID/resources"
  [[ "$API_STATUS" == "200" ]] || die "failed to list server resources: HTTP $API_STATUS: $API_BODY"
  python3 -c '
import json
import sys

target = sys.argv[1]
items = json.load(sys.stdin)
for item in items:
    if item.get("type") == "application" and item.get("name") == target:
        print(item.get("uuid", ""))
        break
' "$app_name" <<<"$API_BODY"
}

find_application_records_by_ticket_identifier() {
  local ticket_identifier="$1"
  local env_name="${2:-}"
  local ticket_slug
  local env_prefix=""

  ticket_slug="$(normalize_ticket_identifier "$ticket_identifier")"
  if [[ -n "$env_name" ]]; then
    env_prefix="$(slugify_name "$env_name")-"
  fi

  api_request GET "/api/v1/servers/$COOLIFY_SERVER_UUID/resources"
  [[ "$API_STATUS" == "200" ]] || die "failed to list server resources: HTTP $API_STATUS: $API_BODY"
  python3 -c '
import json
import re
import sys

ticket_slug = sys.argv[1]
env_prefix = sys.argv[2]
pattern = re.compile(r"(^|-)%s(-|$)" % re.escape(ticket_slug))
items = json.load(sys.stdin)
for item in items:
    if item.get("type") != "application":
        continue
    name = (item.get("name") or "").strip()
    if not name:
        continue
    lowered = name.lower()
    if env_prefix and not lowered.startswith(env_prefix):
        continue
    if not pattern.search(lowered):
        continue
    print(json.dumps({"uuid": item.get("uuid", ""), "name": name}, ensure_ascii=True))
' "$ticket_slug" "$env_prefix" <<<"$API_BODY"
}

build_payload() {
  local mode="$1"
  local env_name="$2"
  local app_name="$3"
  local branch="$4"

  APP_ENV_NAME="$env_name" \
  APP_NAME="$app_name" \
  APP_BRANCH="$branch" \
  APP_MODE="$mode" \
  python3 <<'PY'
import json
import os

mode = os.environ["APP_MODE"]
env_name = os.environ["APP_ENV_NAME"]
app_name = os.environ["APP_NAME"]
branch = os.environ["APP_BRANCH"]
is_update = mode.startswith("update:")
repo_mode = mode.split(":", 1)[1] if ":" in mode else ""

description_prefix = os.environ.get("COOLIFY_DESCRIPTION_PREFIX", "OpenASE review env")
git_repository = os.environ["COOLIFY_GIT_REPOSITORY"]
if is_update and repo_mode == "public":
    for prefix in ("https://github.com/", "http://github.com/"):
        if git_repository.startswith(prefix):
            git_repository = git_repository[len(prefix):]
            break
payload = {
    "git_repository": git_repository,
    "git_branch": branch,
    "build_pack": os.environ.get("COOLIFY_BUILD_PACK", "dockerfile"),
    "ports_exposes": os.environ["COOLIFY_PORTS_EXPOSES"],
    "name": app_name,
    "description": f"{description_prefix}: {branch}",
    "is_auto_deploy_enabled": False,
}

if not is_update:
    payload["project_uuid"] = os.environ["COOLIFY_PROJECT_UUID"]
    payload["server_uuid"] = os.environ["COOLIFY_SERVER_UUID"]
    payload["environment_name"] = env_name
    payload["destination_uuid"] = os.environ["COOLIFY_DESTINATION_UUID"]

optional_map = {
    "COOLIFY_BASE_DIRECTORY": "base_directory",
    "COOLIFY_PUBLISH_DIRECTORY": "publish_directory",
    "COOLIFY_DOCKERFILE_LOCATION": "dockerfile_location",
    "COOLIFY_INSTALL_COMMAND": "install_command",
    "COOLIFY_BUILD_COMMAND": "build_command",
    "COOLIFY_START_COMMAND": "start_command",
    "COOLIFY_HEALTH_CHECK_PATH": "health_check_path",
    "COOLIFY_HEALTH_CHECK_PORT": "health_check_port",
}

for env_key, payload_key in optional_map.items():
    value = os.environ.get(env_key, "").strip()
    if value:
        payload[payload_key] = value

health_enabled = os.environ.get("COOLIFY_HEALTH_CHECK_ENABLED", "").strip().lower()
if health_enabled in {"true", "false"}:
    payload["health_check_enabled"] = health_enabled == "true"

domain_template = os.environ.get("COOLIFY_DOMAIN_TEMPLATE", "").strip()
if domain_template:
    payload["domains"] = domain_template.replace("{{name}}", app_name)
    payload["force_domain_override"] = True

autogenerate_domain = os.environ.get("COOLIFY_AUTOGENERATE_DOMAIN", "").strip().lower()
if not is_update and autogenerate_domain in {"true", "false"}:
    payload["autogenerate_domain"] = autogenerate_domain == "true"

if mode == "create:private-github-app" or mode == "update:private-github-app":
    payload["github_app_uuid"] = os.environ["COOLIFY_GITHUB_APP_UUID"]
elif mode == "create:private-deploy-key" or mode == "update:private-deploy-key":
    payload["private_key_uuid"] = os.environ["COOLIFY_PRIVATE_KEY_UUID"]

print(json.dumps(payload, ensure_ascii=True))
PY
}

deployment_is_success() {
  local status="$1"
  [[ "$status" =~ ^(success|successful|succeeded|finished|completed|ready)$ ]]
}

deployment_is_failure() {
  local status="$1"
  [[ "$status" =~ ^(failed|error|errored|cancelled|canceled|stopped)$ ]]
}

sync_envs_from_template() {
  local target_app_uuid="$1"
  local source_app_uuid="${COOLIFY_TEMPLATE_APPLICATION_UUID:-}"
  local payload

  [[ -n "$source_app_uuid" ]] || return 0

  api_request GET "/api/v1/applications/$source_app_uuid/envs"
  [[ "$API_STATUS" == "200" ]] || die "failed to fetch template envs from $source_app_uuid: HTTP $API_STATUS: $API_BODY"

  payload="$(python3 -c '
import json
import sys

items = json.load(sys.stdin)
chosen = {}
for item in items:
    key = item.get("key")
    if not key:
        continue
    current = chosen.get(key)
    if current is None:
        chosen[key] = item
        continue
    # Prefer normal runtime values over preview duplicates.
    if current.get("is_preview") and not item.get("is_preview"):
        chosen[key] = item

payload = {
    "data": [
        {
            "key": item["key"],
            "value": item.get("real_value") or item.get("value") or "",
            "is_preview": False,
            "is_literal": bool(item.get("is_literal", False)),
            "is_multiline": bool(item.get("is_multiline", False)),
            "is_shown_once": bool(item.get("is_shown_once", False)),
        }
        for item in chosen.values()
        if not item.get("is_coolify", False)
    ]
}
print(json.dumps(payload, ensure_ascii=True))
' <<<"$API_BODY")"

  api_request PATCH "/api/v1/applications/$target_app_uuid/envs/bulk" "$payload"
  [[ "$API_STATUS" == "200" || "$API_STATUS" == "201" ]] || die "failed to sync envs to $target_app_uuid: HTTP $API_STATUS: $API_BODY"
}

sync_file_storages_from_template() {
  local target_app_uuid="$1"
  local source_app_uuid="${COOLIFY_TEMPLATE_APPLICATION_UUID:-}"
  local source_file
  local target_file
  local actions
  local action
  local payload
  local storage_uuid

  [[ -n "$source_app_uuid" ]] || return 0

  source_file="$(mktemp)"
  target_file="$(mktemp)"
  trap 'rm -f "$source_file" "$target_file"' RETURN

  api_request GET "/api/v1/applications/$source_app_uuid/storages"
  [[ "$API_STATUS" == "200" ]] || die "failed to fetch template storages from $source_app_uuid: HTTP $API_STATUS: $API_BODY"
  printf '%s' "$API_BODY" > "$source_file"

  api_request GET "/api/v1/applications/$target_app_uuid/storages"
  [[ "$API_STATUS" == "200" ]] || die "failed to fetch target storages from $target_app_uuid: HTTP $API_STATUS: $API_BODY"
  printf '%s' "$API_BODY" > "$target_file"

  actions="$(python3 - "$source_file" "$target_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    source = json.load(fh)
with open(sys.argv[2], "r", encoding="utf-8") as fh:
    target = json.load(fh)

source_items = source.get("file_storages", [])
target_items = target.get("file_storages", [])
target_by_mount = {
    item.get("mount_path"): item
    for item in target_items
    if item.get("mount_path")
}

for item in source_items:
    mount_path = item.get("mount_path")
    if not mount_path:
        continue
    target_item = target_by_mount.get(mount_path)
    action = {
        "type": "file",
        "mount_path": mount_path,
        "content": item.get("content") or "",
        "is_directory": bool(item.get("is_directory", False)),
        "is_preview_suffix_enabled": bool(item.get("is_preview_suffix_enabled", False)),
    }
    fs_path = item.get("fs_path")
    if action["is_directory"] and fs_path:
        action["fs_path"] = fs_path
    if target_item is None:
        action["op"] = "create"
    else:
        action["op"] = "update"
        action["uuid"] = target_item.get("uuid")
    print(json.dumps(action, ensure_ascii=True))
PY
)"

  while IFS= read -r action || [[ -n "$action" ]]; do
    [[ -n "$action" ]] || continue

    payload="$(python3 -c '
import json
import sys

action = json.load(sys.stdin)
if action["op"] == "create":
    body = {
        "type": action["type"],
        "mount_path": action["mount_path"],
        "is_directory": action["is_directory"],
    }
    if action["is_directory"]:
        body["fs_path"] = action.get("fs_path", "")
    else:
        body["content"] = action.get("content", "")
else:
    body = {
        "uuid": action["uuid"],
        "type": action["type"],
        "mount_path": action["mount_path"],
        "is_preview_suffix_enabled": action["is_preview_suffix_enabled"],
    }
    if not action["is_directory"]:
        body["content"] = action.get("content", "")
print(json.dumps(body, ensure_ascii=True))
' <<<"$action")"

    if [[ "$(python3 -c 'import json,sys; print(json.load(sys.stdin)["op"])' <<<"$action")" == "create" ]]; then
      api_request POST "/api/v1/applications/$target_app_uuid/storages" "$payload"
      [[ "$API_STATUS" == "201" ]] || die "failed to create file storage on $target_app_uuid: HTTP $API_STATUS: $API_BODY"
      storage_uuid="$(json_get "uuid" <<<"$API_BODY")"
      payload="$(python3 -c '
import json
import sys

action = json.load(sys.stdin)
body = {
    "uuid": sys.argv[1],
    "type": action["type"],
    "is_preview_suffix_enabled": action["is_preview_suffix_enabled"],
}
print(json.dumps(body, ensure_ascii=True))
' "$storage_uuid" <<<"$action")"
      api_request PATCH "/api/v1/applications/$target_app_uuid/storages" "$payload"
      [[ "$API_STATUS" == "200" ]] || die "failed to update file storage preview flag on $target_app_uuid: HTTP $API_STATUS: $API_BODY"
    else
      api_request PATCH "/api/v1/applications/$target_app_uuid/storages" "$payload"
      [[ "$API_STATUS" == "200" ]] || die "failed to update file storage on $target_app_uuid: HTTP $API_STATUS: $API_BODY"
    fi
  done <<<"$actions"
}

load_env_file "$env_file"
