#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
Usage:
  upsert_workpad.sh [ticket-id] (--body <markdown> | --body-file <path|->)

Upsert the persistent `## Workpad` comment for the current ticket by
combining primitive `openase ticket comment list/create/update` commands.
EOF
}

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

ticket_id=""
body=""
body_file=""

while [[ $# -gt 0 ]]; do
	case "$1" in
		--body)
			[[ $# -ge 2 ]] || fail "--body requires a value"
			body="$2"
			shift 2
			;;
		--body-file)
			[[ $# -ge 2 ]] || fail "--body-file requires a value"
			body_file="$2"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		--)
			shift
			break
			;;
		-*)
			fail "unknown argument: $1"
			;;
		*)
			[[ -z "$ticket_id" ]] || fail "unexpected argument: $1"
			ticket_id="$1"
			shift
			;;
	esac
done

[[ $# -eq 0 ]] || fail "unexpected argument: $1"
[[ -n "$body" || -n "$body_file" ]] || fail "one of --body or --body-file is required"
[[ -z "$body" || -z "$body_file" ]] || fail "--body and --body-file are mutually exclusive"

ticket_id="${ticket_id:-${OPENASE_TICKET_ID:-}}"
[[ -n "$ticket_id" ]] || fail "ticket id is required via [ticket-id] or OPENASE_TICKET_ID"

openase_bin="./.openase/bin/openase"
[[ -x "$openase_bin" ]] || fail "expected executable OpenASE wrapper at $openase_bin"
require_command python3

tmpdir="$(mktemp -d)"
cleanup() {
	rm -rf "$tmpdir"
}
trap cleanup EXIT

body_path="$tmpdir/workpad.md"
comments_path="$tmpdir/comments.json"

if [[ -n "$body_file" ]]; then
	if [[ "$body_file" == "-" ]]; then
		cat >"$body_path"
	else
		cat "$body_file" >"$body_path"
	fi
else
	printf '%s' "$body" >"$body_path"
fi

python3 - "$body_path" <<'PY'
from pathlib import Path
import sys

body_path = Path(sys.argv[1])
heading = "## Workpad"
raw = body_path.read_text(encoding="utf-8")
trimmed = raw.strip()
if trimmed.startswith(heading):
    normalized = trimmed
elif trimmed:
    normalized = f"{heading}\n\n{trimmed}"
else:
    normalized = heading
body_path.write_text(normalized, encoding="utf-8")
PY

"$openase_bin" ticket comment list "$ticket_id" >"$comments_path"

comment_id="$(
	python3 - "$comments_path" <<'PY'
from pathlib import Path
import json
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
comments = payload.get("comments", payload)
if not isinstance(comments, list):
    raise SystemExit("ticket comment list response does not contain a comments array")

for comment in comments:
    if not isinstance(comment, dict):
        continue
    if comment.get("is_deleted"):
        continue
    body = comment.get("body_markdown")
    if body is None:
        body = comment.get("body", "")
    if isinstance(body, str) and body.lstrip().startswith("## Workpad"):
        comment_id = str(comment.get("id", "")).strip()
        if comment_id:
            print(comment_id)
            break
PY
)"

if [[ -n "$comment_id" ]]; then
	exec "$openase_bin" ticket comment update "$ticket_id" "$comment_id" --body-file "$body_path"
fi

exec "$openase_bin" ticket comment create "$ticket_id" --body-file "$body_path"
