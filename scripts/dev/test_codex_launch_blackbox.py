#!/usr/bin/env python3

import argparse
import json
import os
import queue
import threading
import time
import urllib.error
import urllib.request
from pathlib import Path


def build_parser() -> argparse.ArgumentParser:
    repo_root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(
        description="Verify claimed -> running/ready Codex launch readiness against a local OpenASE deployment."
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("OPENASE_BASE_URL", "http://127.0.0.1:19836"),
        help="OpenASE base URL, for example http://127.0.0.1:19836",
    )
    parser.add_argument(
        "--claim-timeout-seconds",
        type=int,
        default=30,
        help="Maximum seconds to wait for the claimed signal.",
    )
    parser.add_argument(
        "--ready-timeout-seconds",
        type=int,
        default=30,
        help="Maximum seconds to wait for the ready signal.",
    )
    parser.add_argument(
        "--workspace-path",
        default=str(repo_root),
        help="Absolute workspace path passed into the created agent.",
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
        with urllib.request.urlopen(request, timeout=10) as response:
            raw = response.read().decode("utf-8")
            return json.loads(raw)
    except urllib.error.HTTPError as exc:
        payload_text = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} returned {exc.code}: {payload_text}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"{method} {path} failed: {exc}") from exc


def require_by_name(items, key: str, want: str):
    for item in items:
        if item.get(key) == want:
            return item
    raise RuntimeError(f"could not find item with {key}={want!r} in {items!r}")


def agent_stream_worker(base_url: str, project_id: str, event_queue: queue.Queue, stop_event: threading.Event):
    request = urllib.request.Request(
        base_url.rstrip("/") + f"/api/v1/projects/{project_id}/agents/stream",
        headers={"Accept": "text/event-stream"},
        method="GET",
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        current_event = None
        data_lines = []
        while not stop_event.is_set():
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
                    payload = json.loads("\n".join(data_lines))
                    event_queue.put({"event": current_event, "payload": payload})
                current_event = None
                data_lines = []


def wait_for_claim(base_url: str, project_id: str, agent_id: str, event_queue: queue.Queue, timeout_seconds: int):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        while True:
            try:
                event = event_queue.get_nowait()
            except queue.Empty:
                break

            payload = event.get("payload", {})
            agent = payload.get("payload", {}).get("agent", {}) if isinstance(payload, dict) else {}
            if event.get("event") == "agent.claimed" and agent.get("id") == agent_id:
                return

        agents = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/agents")["agents"]
        current = require_by_name(agents, "id", agent_id)
        if current["status"] == "claimed":
            return
        time.sleep(1)

    raise RuntimeError("timed out waiting for agent.claimed")


def wait_for_ready(
    base_url: str,
    project_id: str,
    agent_id: str,
    event_queue: queue.Queue,
    timeout_seconds: int,
):
    deadline = time.time() + timeout_seconds
    saw_ready_event = False

    while time.time() < deadline:
        while True:
            try:
                event = event_queue.get_nowait()
            except queue.Empty:
                break

            payload = event.get("payload", {})
            agent = payload.get("payload", {}).get("agent", {}) if isinstance(payload, dict) else {}
            if event.get("event") == "agent.ready" and agent.get("id") == agent_id:
                saw_ready_event = True

        agents = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/agents")["agents"]
        current = require_by_name(agents, "id", agent_id)
        if (
            current["status"] == "running"
            and current["runtime_phase"] == "ready"
            and current["session_id"]
            and current.get("runtime_started_at")
            and current.get("last_heartbeat_at")
            and saw_ready_event
        ):
            return current
        time.sleep(1)

    raise RuntimeError("timed out waiting for agent.ready with running/ready/session/heartbeat contract")


def main() -> int:
    args = build_parser().parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    workspace_path = Path(args.workspace_path).resolve()
    if not workspace_path.is_absolute():
        raise RuntimeError("--workspace-path must be absolute")

    fake_codex_path = (repo_root / "scripts" / "dev" / "fake_codex_app_server.py").resolve()
    if not fake_codex_path.exists():
        raise RuntimeError(f"fake codex script not found at {fake_codex_path}")

    base_url = args.base_url.rstrip("/")
    stamp = time.strftime("%Y%m%d%H%M%S")

    print(f"[1/7] health check against {base_url}")
    request_json(base_url, "GET", "/healthz")
    request_json(base_url, "GET", "/api/v1/healthz")

    print("[2/7] create isolated org and project")
    org = request_json(
        base_url,
        "POST",
        "/api/v1/orgs",
        {"name": f"Runtime Blackbox {stamp}", "slug": f"runtime-blackbox-{stamp}"},
    )["organization"]
    project = request_json(
        base_url,
        "POST",
        f"/api/v1/orgs/{org['id']}/projects",
        {
            "name": f"Runtime Launch {stamp}",
            "slug": f"runtime-launch-{stamp}",
            "description": "Temporary project for deterministic Codex launch verification.",
            "status": "active",
            "max_concurrent_agents": 1,
        },
    )["project"]

    print("[3/7] seed statuses and workflow pickup lane")
    statuses = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/statuses/reset",
    )["statuses"]
    todo = require_by_name(statuses, "name", "Todo")
    done = require_by_name(statuses, "name", "Done")
    workflow = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/workflows",
        {
            "name": f"Runtime Workflow {stamp}",
            "type": "coding",
            "pickup_status_ids": [todo["id"]],
            "finish_status_ids": [done["id"]],
            "harness_content": "---\nworkflow:\n  role: coding\n---\n\n# Runtime Launch Blackbox\n",
        },
    )["workflow"]

    print("[4/7] create provider, idle agent, and pickup ticket")
    provider = request_json(
        base_url,
        "POST",
        f"/api/v1/orgs/{org['id']}/providers",
        {
            "name": "Fake Codex Provider",
            "adapter_type": "codex-app-server",
            "cli_command": os.environ.get("PYTHON", "python3"),
            "cli_args": [str(fake_codex_path)],
            "auth_config": {},
            "model_name": "gpt-5.4",
        },
    )["provider"]
    agent = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/agents",
        {
            "provider_id": provider["id"],
            "name": f"runtime-agent-{stamp}",
            "capabilities": ["coding", "runtime-check"],
        },
    )["agent"]
    ticket = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/tickets",
        {
            "title": "Deterministic Codex launch verification",
            "description": "Created by the runtime blackbox verifier.",
            "priority": "high",
            "status_id": todo["id"],
            "workflow_id": workflow["id"],
            "created_by": "user:blackbox",
        },
    )["ticket"]

    stop_event = threading.Event()
    event_queue: queue.Queue = queue.Queue()
    stream_thread = threading.Thread(
        target=agent_stream_worker,
        args=(base_url, project["id"], event_queue, stop_event),
        daemon=True,
    )
    stream_thread.start()

    try:
        print("[5/7] wait for claimed state")
        wait_for_claim(base_url, project["id"], agent["id"], event_queue, args.claim_timeout_seconds)

        print("[6/7] wait for running + ready runtime contract")
        ready_agent = wait_for_ready(
            base_url,
            project["id"],
            agent["id"],
            event_queue,
            args.ready_timeout_seconds,
        )

        print("[7/7] verify mirrored runtime activity history")
        activity = request_json(
            base_url,
            "GET",
            f"/api/v1/projects/{project['id']}/activity?agent_id={agent['id']}&limit=20",
        )["events"]
        if not any(item.get("event_type") == "agent.ready" for item in activity):
            raise RuntimeError(f"expected mirrored agent.ready activity event, got {activity!r}")

        print("PASS")
        print(
            json.dumps(
                {
                    "project_id": project["id"],
                    "ticket_id": ticket["id"],
                    "agent_id": agent["id"],
                    "session_id": ready_agent["session_id"],
                    "runtime_phase": ready_agent["runtime_phase"],
                    "runtime_started_at": ready_agent["runtime_started_at"],
                    "last_heartbeat_at": ready_agent["last_heartbeat_at"],
                },
                indent=2,
            )
        )
        return 0
    finally:
        stop_event.set()


if __name__ == "__main__":
    raise SystemExit(main())
