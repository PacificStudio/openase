#!/usr/bin/env python3

import argparse
import http.cookiejar
import json
import os
import sys
import tempfile
import time
import urllib.error
import urllib.request
from typing import Any


class ChatTurnError(RuntimeError):
    def __init__(self, payload: dict[str, Any]):
        self.payload = payload
        super().__init__(json.dumps(payload, ensure_ascii=False))


def build_parser() -> argparse.ArgumentParser:
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
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
    parser.add_argument(
        "--provider-mode",
        choices=("auto", "fake-codex"),
        default=os.environ.get("OPENASE_CHAT_PROVIDER_MODE", "auto"),
        help="Choose whether to use existing available providers or seed a deterministic fake Codex provider.",
    )
    parser.add_argument(
        "--python-bin",
        default=os.environ.get("PYTHON", "python3"),
        help="Python executable to use when --provider-mode=fake-codex.",
    )
    parser.add_argument(
        "--fake-codex-script",
        default=os.environ.get(
            "OPENASE_FAKE_CODEX_SCRIPT",
            os.path.join(repo_root, "scripts", "dev", "fake_codex_app_server.py"),
        ),
        help="Path to the fake Codex app server script used by --provider-mode=fake-codex.",
    )
    return parser


def build_opener() -> urllib.request.OpenerDirector:
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def request_json(
    opener: urllib.request.OpenerDirector,
    base_url: str,
    method: str,
    path: str,
    payload=None,
    headers: dict[str, str] | None = None,
):
    body = None
    request_headers = {"Accept": "application/json"}
    if headers:
        request_headers.update(headers)
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        request_headers["Content-Type"] = "application/json"

    request = urllib.request.Request(
        base_url.rstrip("/") + path,
        data=body,
        headers=request_headers,
        method=method,
    )
    try:
        with opener.open(request, timeout=20) as response:
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


def require_single_local_machine(opener: urllib.request.OpenerDirector, base_url: str, org_id: str) -> dict[str, Any]:
    machines = request_json(opener, base_url, "GET", f"/api/v1/orgs/{org_id}/machines").get("machines", [])
    local_machine = next((item for item in machines if item.get("name") == "local"), None)
    if local_machine is None:
        raise RuntimeError(f"organization {org_id} does not expose a local machine")
    if local_machine.get("status") != "online":
        raise RuntimeError(
            f"organization {org_id} local machine is not healthy: status={local_machine.get('status')!r}"
        )
    return local_machine


def resolve_chat_provider(items: list[dict[str, Any]]) -> dict[str, Any]:
    return resolve_chat_providers(items)[0]


def resolve_chat_providers(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    preferred_adapters = ["claude-code-cli", "codex-app-server", "gemini-cli"]
    resolved: list[dict[str, Any]] = []
    for adapter_type in preferred_adapters:
        for item in items:
            if item.get("adapter_type") == adapter_type and item.get("available"):
                resolved.append(item)

    if resolved:
        return resolved

    raise RuntimeError(
        f"could not find an available Ephemeral Chat provider in {json.dumps(items, ensure_ascii=False)}"
    )


def wait_for_chat_providers(
    opener: urllib.request.OpenerDirector, base_url: str, org_id: str, timeout_seconds: int
) -> list[dict[str, Any]]:
    deadline = time.time() + timeout_seconds
    last_providers: list[dict[str, Any]] = []

    while time.time() < deadline:
        providers = request_json(opener, base_url, "GET", f"/api/v1/orgs/{org_id}/providers")["providers"]
        last_providers = providers
        try:
            return resolve_chat_providers(providers)
        except RuntimeError:
            time.sleep(1)

    raise RuntimeError(
        "timed out waiting for an available Ephemeral Chat provider in "
        + json.dumps(last_providers, ensure_ascii=False)
    )


def wait_for_provider(
    opener: urllib.request.OpenerDirector, base_url: str, org_id: str, provider_id: str, timeout_seconds: int
) -> dict[str, Any]:
    deadline = time.time() + timeout_seconds
    last_provider: dict[str, Any] | None = None

    while time.time() < deadline:
        providers = request_json(opener, base_url, "GET", f"/api/v1/orgs/{org_id}/providers")["providers"]
        current = next((item for item in providers if item.get("id") == provider_id), None)
        if current is not None:
            last_provider = current
            if current.get("available"):
                return current
        time.sleep(1)

    raise RuntimeError(
        "timed out waiting for provider to become available: "
        + json.dumps(last_provider, ensure_ascii=False)
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


def start_chat_turn(
    opener: urllib.request.OpenerDirector,
    base_url: str,
    timeout_seconds: int,
    payload: dict[str, Any],
    *,
    headers: dict[str, str] | None = None,
    require_text: bool = True,
) -> dict[str, Any]:
    request_headers = {
        "Accept": "text/event-stream",
        "Content-Type": "application/json",
    }
    if headers:
        request_headers.update(headers)
    request = urllib.request.Request(
        base_url + "/api/v1/chat",
        data=json.dumps(payload).encode("utf-8"),
        headers=request_headers,
        method="POST",
    )

    text_parts: list[str] = []
    done_payload = None
    with opener.open(request, timeout=timeout_seconds) as response:
        for event_name, event_payload in read_sse_stream(response, timeout_seconds):
            if event_name == "error":
                raise ChatTurnError(event_payload)
            if event_name == "message" and event_payload.get("type") == "text":
                text_parts.append(str(event_payload.get("content", "")))
            if event_name == "done":
                done_payload = event_payload
                break

    assistant_text = "".join(text_parts)
    if require_text and not assistant_text.strip():
        raise RuntimeError("expected chat stream to emit a non-empty assistant text message")
    if not isinstance(done_payload, dict):
        raise RuntimeError("expected chat stream to emit a done event")
    session_id = str(done_payload.get("session_id", "")).strip()
    if session_id == "":
        raise RuntimeError(
            f"expected done event to include session_id, got {json.dumps(done_payload, ensure_ascii=False)}"
        )

    return {
        "assistant_text": assistant_text,
        "done": done_payload,
    }


def is_skippable_provider_error(error: ChatTurnError) -> bool:
    message = str(error.payload.get("message", "")).lower()
    return (
        "hit your limit" in message
        or "rate limit" in message
        or "missing environment variable" in message
        or "missing api key" in message
        or "api key" in message
    )


def create_fake_codex_launcher(python_bin: str, fake_codex_path: str) -> str:
    launcher_fd, launcher_path = tempfile.mkstemp(prefix="openase-fake-codex-", suffix=".sh")
    launcher_body = "#!/bin/sh\nshift 3\nexec \"$1\" \"$2\"\n"
    os.write(launcher_fd, launcher_body.encode("utf-8"))
    os.close(launcher_fd)
    os.chmod(launcher_path, 0o755)
    return launcher_path


def close_chat_session(
    opener: urllib.request.OpenerDirector,
    base_url: str,
    timeout_seconds: int,
    session_id: str,
    headers: dict[str, str],
) -> None:
    close_request = urllib.request.Request(
        base_url + f"/api/v1/chat/{session_id}",
        headers={"Accept": "application/json", **headers},
        method="DELETE",
    )
    with opener.open(close_request, timeout=timeout_seconds) as response:
        if response.status != 204:
            raise RuntimeError(
                f"expected DELETE /api/v1/chat/{session_id} to return 204, got {response.status}"
            )


def expect_resume_failure(
    opener: urllib.request.OpenerDirector,
    base_url: str,
    timeout_seconds: int,
    payload: dict[str, Any],
    headers: dict[str, str],
    expected_status: int,
    expected_code: str,
) -> None:
    request = urllib.request.Request(
        base_url + "/api/v1/chat",
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Accept": "application/json",
            "Content-Type": "application/json",
            **headers,
        },
        method="POST",
    )
    try:
        with opener.open(request, timeout=timeout_seconds):
            raise RuntimeError("expected chat resume request to fail")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        if exc.code != expected_status or expected_code not in body:
            raise RuntimeError(
                f"expected resume failure {expected_status}/{expected_code}, got {exc.code}: {body}"
            ) from exc


def main() -> int:
    args = build_parser().parse_args()
    base_url = args.base_url.rstrip("/")
    stamp = time.strftime("%Y%m%d%H%M%S")
    opener = build_opener()
    chat_headers: dict[str, str] = {}
    fake_launcher_path: str | None = None

    try:
        print(f"[1/9] health check against {base_url}")
        request_json(opener, base_url, "GET", "/healthz")
        request_json(opener, base_url, "GET", "/api/v1/healthz")

        print("[2/9] create isolated organization and project")
        org = request_json(
            opener,
            base_url,
            "POST",
            "/api/v1/orgs",
            {
                "name": f"Ephemeral Chat Blackbox {stamp}",
                "slug": f"ephemeral-chat-blackbox-{stamp}",
            },
        )["organization"]
        project = request_json(
            opener,
            base_url,
            "POST",
            f"/api/v1/orgs/{org['id']}/projects",
            {
                "name": f"Ephemeral Chat Project {stamp}",
                "slug": f"ephemeral-chat-project-{stamp}",
                "description": "Temporary project for local ephemeral chat verification.",
                "status": "In Progress",
                "max_concurrent_agents": 1,
            },
        )["project"]

        if args.provider_mode == "fake-codex":
            print("[3/9] seed a deterministic fake Codex Ephemeral Chat provider")
            fake_codex_path = os.path.abspath(args.fake_codex_script)
            if not os.path.exists(fake_codex_path):
                raise RuntimeError(f"fake codex app server not found at {fake_codex_path}")
            local_machine = require_single_local_machine(opener, base_url, org["id"])
            fake_launcher_path = create_fake_codex_launcher(args.python_bin, fake_codex_path)
            fake_provider = request_json(
                opener,
                base_url,
                "POST",
                f"/api/v1/orgs/{org['id']}/providers",
                {
                    "machine_id": local_machine["id"],
                    "name": "Fake Codex Ephemeral Chat",
                    "adapter_type": "codex-app-server",
                    "cli_command": fake_launcher_path,
                    "cli_args": [args.python_bin, fake_codex_path],
                    "auth_config": {},
                    "model_name": "gpt-5.4",
                },
            )["provider"]
            chat_providers = [wait_for_provider(opener, base_url, org["id"], fake_provider["id"], args.timeout_seconds)]
        else:
            print("[3/9] wait for available Ephemeral Chat providers")
            chat_providers = wait_for_chat_providers(opener, base_url, org["id"], args.timeout_seconds)
        default_chat_provider = chat_providers[0]

        print("[4/9] set the selected provider as the default project provider")
        project = request_json(
            opener,
            base_url,
            "PATCH",
            f"/api/v1/projects/{project['id']}",
            {
                "default_agent_provider_id": default_chat_provider["id"],
            },
        )["project"]

        print("[5/9] create a ticket for ticket-detail chat coverage")
        ticket = request_json(
            opener,
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/tickets",
            {
                "title": "Ephemeral Chat ticket detail verification",
                "description": "Created by the blackbox test to exercise ticket_detail chat entry.",
            },
        )["ticket"]

        print("[6/9] run explicit-provider project-sidebar coverage against every available provider")
        explicit_results: list[dict[str, Any]] = []
        skipped_providers: list[dict[str, Any]] = []
        for provider in chat_providers:
            explicit_payload = {
                "message": (
                    f"Reply with one short sentence confirming explicit provider selection works for {provider['name']}."
                ),
                "source": "project_sidebar",
                "provider_id": provider["id"],
                "context": {
                    "project_id": project["id"],
                },
                "session_id": None,
            }
            try:
                explicit_result = start_chat_turn(
                    opener,
                    base_url,
                    args.timeout_seconds,
                    explicit_payload,
                    headers=chat_headers,
                )
            except ChatTurnError as exc:
                if not is_skippable_provider_error(exc):
                    raise RuntimeError(
                        f"chat stream returned error payload: {json.dumps(exc.payload, ensure_ascii=False)}"
                    ) from exc
                skipped_providers.append(
                    {
                        "provider": provider,
                        "reason": exc.payload,
                    }
                )
                continue
            explicit_session_id = str(explicit_result["done"].get("session_id", "")).strip()
            close_chat_session(opener, base_url, args.timeout_seconds, explicit_session_id, chat_headers)
            expect_resume_failure(
                opener,
                base_url,
                args.timeout_seconds,
                {**explicit_payload, "session_id": explicit_session_id},
                chat_headers,
                404,
                "CHAT_SESSION_NOT_FOUND",
            )
            explicit_results.append(
                {
                    "provider": provider,
                    "assistant_text": explicit_result["assistant_text"],
                    "done": explicit_result["done"],
                }
            )
        if not explicit_results:
            raise RuntimeError(
                "no available provider completed the explicit-provider chat probe: "
                + json.dumps(skipped_providers, ensure_ascii=False)
            )
        successful_providers = [item["provider"] for item in explicit_results]
        default_chat_provider = successful_providers[0]
        if project.get("default_agent_provider_id") != default_chat_provider["id"]:
            project = request_json(
                opener,
                base_url,
                "PATCH",
                f"/api/v1/projects/{project['id']}",
                {
                    "default_agent_provider_id": default_chat_provider["id"],
                },
            )["project"]

        print("[7/9] verify same-user replacement closes the previous session deterministically")
        replacement_first_payload = {
            "message": "Reply with one short sentence for the first replacement-session probe.",
            "source": "project_sidebar",
            "provider_id": default_chat_provider["id"],
            "context": {
                "project_id": project["id"],
            },
            "session_id": None,
        }
        replacement_first = start_chat_turn(
            opener,
            base_url,
            args.timeout_seconds,
            replacement_first_payload,
            headers=chat_headers,
        )
        replacement_first_session = str(replacement_first["done"].get("session_id", "")).strip()
        replacement_provider = successful_providers[1] if len(successful_providers) > 1 else default_chat_provider
        replacement_second_payload = {
            "message": "Reply with one short sentence for the replacement-session probe after switching context.",
            "source": "project_sidebar",
            "provider_id": replacement_provider["id"],
            "context": {
                "project_id": project["id"],
            },
            "session_id": None,
        }
        replacement_second = start_chat_turn(
            opener,
            base_url,
            args.timeout_seconds,
            replacement_second_payload,
            headers=chat_headers,
        )
        replacement_second_session = str(replacement_second["done"].get("session_id", "")).strip()
        if replacement_first_session == replacement_second_session:
            raise RuntimeError("expected replacement session to allocate a new session id")
        expect_resume_failure(
            opener,
            base_url,
            args.timeout_seconds,
            {**replacement_first_payload, "session_id": replacement_first_session},
            chat_headers,
            404,
            "CHAT_SESSION_NOT_FOUND",
        )
        close_chat_session(opener, base_url, args.timeout_seconds, replacement_second_session, chat_headers)

        print("[8/9] start ticket-detail chat with explicit provider selection")
        ticket_payload = {
            "message": "Reply with one short sentence confirming this ticket-detail chat is working.",
            "source": "ticket_detail",
            "provider_id": default_chat_provider["id"],
            "context": {
                "project_id": project["id"],
                "ticket_id": ticket["id"],
            },
            "session_id": None,
        }
        ticket_result = start_chat_turn(opener, base_url, args.timeout_seconds, ticket_payload, headers=chat_headers)
        ticket_session_id = str(ticket_result["done"].get("session_id", "")).strip()
        close_chat_session(opener, base_url, args.timeout_seconds, ticket_session_id, chat_headers)
        expect_resume_failure(
            opener,
            base_url,
            args.timeout_seconds,
            {**ticket_payload, "session_id": ticket_session_id},
            chat_headers,
            404,
            "CHAT_SESSION_NOT_FOUND",
        )

        print("[9/9] start project-sidebar chat via default-provider fallback")
        fallback_payload = {
            "message": "Reply with one short sentence confirming default provider fallback works.",
            "source": "project_sidebar",
            "context": {
                "project_id": project["id"],
            },
            "session_id": None,
        }
        fallback_result = start_chat_turn(
            opener,
            base_url,
            args.timeout_seconds,
            fallback_payload,
            headers=chat_headers,
        )
        fallback_session_id = str(fallback_result["done"].get("session_id", "")).strip()
        close_chat_session(opener, base_url, args.timeout_seconds, fallback_session_id, chat_headers)
        expect_resume_failure(
            opener,
            base_url,
            args.timeout_seconds,
            {**fallback_payload, "session_id": fallback_session_id},
            chat_headers,
            404,
            "CHAT_SESSION_NOT_FOUND",
        )

        print("[9/9] summarize results")
        print(
            json.dumps(
                {
                    "base_url": base_url,
                    "organization": org,
                    "project": project,
                    "providers": chat_providers,
                    "skipped_providers": skipped_providers,
                    "ticket": ticket,
                    "explicit_provider_runs": explicit_results,
                    "replacement": {
                        "first": replacement_first,
                        "second": replacement_second,
                    },
                    "ticket_detail": {
                        "assistant_text": ticket_result["assistant_text"],
                        "done": ticket_result["done"],
                    },
                    "fallback": {
                        "assistant_text": fallback_result["assistant_text"],
                        "done": fallback_result["done"],
                    },
                },
                indent=2,
                ensure_ascii=False,
            )
        )
        return 0
    finally:
        if fake_launcher_path is not None:
            try:
                os.unlink(fake_launcher_path)
            except FileNotFoundError:
                pass


if __name__ == "__main__":
    raise SystemExit(main())
