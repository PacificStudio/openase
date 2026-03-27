#!/usr/bin/env python3

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from typing import Any


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Verify ephemeral chat works against a locally deployed OpenASE instance."
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("OPENASE_BASE_URL", "http://127.0.0.1:19836"),
        help="OpenASE base URL, for example http://127.0.0.1:19836",
    )
    parser.add_argument(
        "--timeout-seconds",
        type=int,
        default=60,
        help="Maximum seconds to wait for the chat stream to produce a terminal event.",
    )
    return parser


def request_json(base_url: str, method: str, path: str, payload=None):
    body = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"

    request = urllib.request.Request(
        base_url.rstrip("/") + path,
        data=body,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            raw = response.read().decode("utf-8")
            return json.loads(raw)
    except urllib.error.HTTPError as exc:
        payload_text = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} returned {exc.code}: {payload_text}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"{method} {path} failed: {exc}") from exc


def require_by_name(items: list[dict[str, Any]], key: str, want: str) -> dict[str, Any]:
    for item in items:
        if item.get(key) == want:
            return item
    raise RuntimeError(f"could not find item with {key}={want!r} in {items!r}")


def resolve_chat_provider(items: list[dict[str, Any]]) -> dict[str, Any]:
    preferred_adapters = ["claude-code-cli", "codex-app-server", "gemini-cli"]
    for adapter_type in preferred_adapters:
        for item in items:
            if item.get("adapter_type") == adapter_type and item.get("available"):
                return item

    raise RuntimeError(
        f"could not find an available Ephemeral Chat provider in {json.dumps(items, ensure_ascii=False)}"
    )


def wait_for_chat_provider(base_url: str, org_id: str, timeout_seconds: int) -> dict[str, Any]:
    deadline = time.time() + timeout_seconds
    last_providers: list[dict[str, Any]] = []

    while time.time() < deadline:
        providers = request_json(base_url, "GET", f"/api/v1/orgs/{org_id}/providers")["providers"]
        last_providers = providers
        try:
            return resolve_chat_provider(providers)
        except RuntimeError:
            time.sleep(1)

    raise RuntimeError(
        "timed out waiting for an available Ephemeral Chat provider in "
        + json.dumps(last_providers, ensure_ascii=False)
    )


def read_sse_stream(response, timeout_seconds: int):
    deadline = time.time() + timeout_seconds
    current_event = None
    data_lines: list[str] = []

    while time.time() < deadline:
        raw_line = response.readline()
        if not raw_line:
            return

        line = raw_line.decode("utf-8", errors="replace").rstrip("\n")
        if line.startswith(":"):
            continue
        if line.startswith("event:"):
            current_event = line[len("event:") :].strip()
            continue
        if line.startswith("data:"):
            data_lines.append(line[len("data:") :].strip())
            continue
        if line == "":
            if current_event and data_lines:
                yield current_event, json.loads("\n".join(data_lines))
            current_event = None
            data_lines = []

    raise RuntimeError(f"timed out after {timeout_seconds}s waiting for chat SSE terminal event")


def start_chat_turn(base_url: str, timeout_seconds: int, payload: dict[str, Any]) -> tuple[str, dict[str, Any]]:
    request = urllib.request.Request(
        base_url + "/api/v1/chat",
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Accept": "text/event-stream",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    text_parts: list[str] = []
    done_payload = None
    with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
        for event_name, event_payload in read_sse_stream(response, timeout_seconds):
            if event_name == "error":
                raise RuntimeError(
                    f"chat stream returned error payload: {json.dumps(event_payload, ensure_ascii=False)}"
                )
            if event_name == "message" and event_payload.get("type") == "text":
                text_parts.append(str(event_payload.get("content", "")))
            if event_name == "done":
                done_payload = event_payload
                break

    assistant_text = "".join(text_parts)
    if not assistant_text.strip():
        raise RuntimeError("expected chat stream to emit a non-empty assistant text message")
    if not isinstance(done_payload, dict):
        raise RuntimeError("expected chat stream to emit a done event")
    session_id = str(done_payload.get("session_id", "")).strip()
    if session_id == "":
        raise RuntimeError(
            f"expected done event to include session_id, got {json.dumps(done_payload, ensure_ascii=False)}"
        )

    return assistant_text, done_payload


def main() -> int:
    args = build_parser().parse_args()
    base_url = args.base_url.rstrip("/")
    stamp = time.strftime("%Y%m%d%H%M%S")

    print(f"[1/7] health check against {base_url}")
    request_json(base_url, "GET", "/healthz")
    request_json(base_url, "GET", "/api/v1/healthz")

    print("[2/7] create isolated organization and project")
    org = request_json(
        base_url,
        "POST",
        "/api/v1/orgs",
        {
            "name": f"Ephemeral Chat Blackbox {stamp}",
            "slug": f"ephemeral-chat-blackbox-{stamp}",
        },
    )["organization"]
    project = request_json(
        base_url,
        "POST",
        f"/api/v1/orgs/{org['id']}/projects",
        {
            "name": f"Ephemeral Chat Project {stamp}",
            "slug": f"ephemeral-chat-project-{stamp}",
            "description": "Temporary project for local ephemeral chat verification.",
            "status": "active",
            "max_concurrent_agents": 1,
        },
    )["project"]

    print("[3/9] wait for an available Ephemeral Chat provider")
    chat_provider = wait_for_chat_provider(base_url, org["id"], args.timeout_seconds)

    print("[4/9] set the selected provider as the default project provider")
    project = request_json(
        base_url,
        "PATCH",
        f"/api/v1/projects/{project['id']}",
        {
            "default_agent_provider_id": chat_provider["id"],
        },
    )["project"]

    print("[5/9] start project-sidebar chat with explicit provider selection")
    explicit_payload = {
        "message": "Reply with one short sentence confirming this project sidebar chat is working.",
        "source": "project_sidebar",
        "provider_id": chat_provider["id"],
        "context": {
            "project_id": project["id"],
        },
        "session_id": None,
    }
    explicit_first_text, explicit_done_payload = start_chat_turn(base_url, args.timeout_seconds, explicit_payload)
    explicit_session_id = str(explicit_done_payload.get("session_id", "")).strip()

    print("[6/9] close the explicit-provider chat session")
    close_request = urllib.request.Request(
        base_url + f"/api/v1/chat/{explicit_session_id}",
        headers={"Accept": "application/json"},
        method="DELETE",
    )
    with urllib.request.urlopen(close_request, timeout=20) as response:
        if response.status != 204:
            raise RuntimeError(
                f"expected DELETE /api/v1/chat/{explicit_session_id} to return 204, got {response.status}"
            )

    print("[7/9] start project-sidebar chat via default-provider fallback")
    fallback_payload = {
        "message": "Reply with one short sentence confirming default provider fallback works.",
        "source": "project_sidebar",
        "context": {
            "project_id": project["id"],
        },
        "session_id": None,
    }
    fallback_first_text, fallback_done_payload = start_chat_turn(
        base_url, args.timeout_seconds, fallback_payload
    )
    fallback_session_id = str(fallback_done_payload.get("session_id", "")).strip()

    print("[8/9] close the fallback chat session")
    close_request = urllib.request.Request(
        base_url + f"/api/v1/chat/{fallback_session_id}",
        headers={"Accept": "application/json"},
        method="DELETE",
    )
    with urllib.request.urlopen(close_request, timeout=20) as response:
        if response.status != 204:
            raise RuntimeError(
                f"expected DELETE /api/v1/chat/{fallback_session_id} to return 204, got {response.status}"
            )

    print("[9/9] summarize results")
    print(
        json.dumps(
            {
                "base_url": base_url,
                "organization": org,
                "project": project,
                "provider": chat_provider,
                "explicit": {
                "assistant_text": explicit_first_text,
                "done": explicit_done_payload,
            },
            "fallback": {
                "assistant_text": fallback_first_text,
                "done": fallback_done_payload,
            },
            },
            indent=2,
            ensure_ascii=False,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
