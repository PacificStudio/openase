#!/usr/bin/env python3

import argparse
import json
import os
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


def build_parser() -> argparse.ArgumentParser:
    repo_root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(
        description="Run a black-box smoke test against a locally deployed OpenASE instance."
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("OPENASE_BASE_URL", "http://127.0.0.1:19836"),
        help="OpenASE base URL, for example http://127.0.0.1:19836",
    )
    parser.add_argument(
        "--openase-bin",
        default=os.environ.get("OPENASE_BIN", str(repo_root / "bin" / "openase")),
        help="Path to the openase binary.",
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


def require_single_local_machine(base_url: str, org_id: str) -> dict:
    machines = request_json(base_url, "GET", f"/api/v1/orgs/{org_id}/machines").get("machines", [])
    local_machine = next((item for item in machines if item.get("name") == "local"), None)
    if local_machine is None:
        raise RuntimeError(f"organization {org_id} does not expose a local machine")
    if local_machine.get("status") != "online":
        raise RuntimeError(
            f"organization {org_id} local machine is not healthy: status={local_machine.get('status')!r}"
        )
    return local_machine


def run_cli(command, env=None, check=True):
    result = subprocess.run(
        command,
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    if check and result.returncode != 0:
        raise RuntimeError(
            "command failed:\n"
            + " ".join(command)
            + "\nstdout:\n"
            + result.stdout
            + "\nstderr:\n"
            + result.stderr
        )
    return result


def run_cli_json(command, env=None):
    result = run_cli(command, env=env, check=True)
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(
            "expected JSON output from command:\n"
            + " ".join(command)
            + "\nstdout:\n"
            + result.stdout
            + "\nstderr:\n"
            + result.stderr
        ) from exc


def assert_idle_agent_defaults(agent: dict):
    if agent.get("runtime_control_state") != "active":
        raise RuntimeError(
            f"expected new agent runtime_control_state to be 'active', got {agent.get('runtime_control_state')!r}"
        )
    if agent.get("total_tokens_used") != 0:
        raise RuntimeError(f"expected new agent total_tokens_used to be 0, got {agent.get('total_tokens_used')!r}")
    if agent.get("total_tickets_completed") != 0:
        raise RuntimeError(
            "expected new agent total_tickets_completed to be 0, "
            f"got {agent.get('total_tickets_completed')!r}"
        )
    if agent.get("runtime") is not None:
        raise RuntimeError(f"expected new agent runtime summary to be absent, got {agent.get('runtime')!r}")


def cleanup_harness_artifacts(repo_root: Path | None, workflow: dict | None):
    if repo_root is None:
        return

    harnesses_root = (repo_root / ".openase" / "harnesses").resolve()
    cleanup_targets = []

    if workflow is not None:
        raw_harness_path = workflow.get("harness_path", "")
        if isinstance(raw_harness_path, str) and raw_harness_path.strip():
            cleanup_targets.append((repo_root / raw_harness_path).resolve())

    seen = set()
    for target in cleanup_targets:
        if target in seen:
            continue
        seen.add(target)

        if target.is_file():
            target.unlink(missing_ok=True)
            continue

        if target.is_dir() and harnesses_root in target.parents:
            shutil.rmtree(target, ignore_errors=True)


def main() -> int:
    args = build_parser().parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    base_url = args.base_url.rstrip("/")
    openase_bin = args.openase_bin
    project = None
    workflow = None
    primary_repo_root = None

    if not Path(openase_bin).exists():
        raise RuntimeError(f"openase binary not found at {openase_bin}")

    stamp = time.strftime("%Y%m%d%H%M%S")
    org_slug = f"openase-blackbox-{stamp}"
    project_slug = f"openase-blackbox-project-{stamp}"
    workflow_name = f"Blackbox Coding Workflow {stamp}"
    primary_repo_name = f"blackbox-main-{stamp}"
    agent_repo_name = f"agent-tools-{stamp}"

    try:
        print(f"[1/11] health check against {base_url}")
        request_json(base_url, "GET", "/healthz")
        request_json(base_url, "GET", "/api/v1/healthz")

        print("[2/11] create isolated organization and project")
        org = request_json(
            base_url,
            "POST",
            "/api/v1/orgs",
            {
                "name": f"OpenASE Blackbox {stamp}",
                "slug": org_slug,
            },
        )["organization"]
        project = request_json(
            base_url,
            "POST",
            f"/api/v1/orgs/{org['id']}/projects",
            {
                "name": f"OpenASE Blackbox Project {stamp}",
                "slug": project_slug,
                "description": "Temporary project for local deployment black-box validation.",
                "status": "In Progress",
                "max_concurrent_agents": 2,
            },
        )["project"]

        print("[3/11] seed default statuses and register a local project repo")
        statuses = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/statuses/reset",
        )["statuses"]
        todo = require_by_name(statuses, "name", "Todo")
        done = require_by_name(statuses, "name", "Done")
        primary_repo_root = (repo_root.parent / primary_repo_name).resolve()
        primary_repo_root.mkdir(parents=True, exist_ok=False)
        (primary_repo_root / ".git").mkdir()
        repo = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/repos",
            {
                "name": primary_repo_name,
                "repository_url": str(primary_repo_root),
                "default_branch": "main",
                "labels": ["smoke", "main"],
            },
        )["repo"]

        print("[4/11] verify local machine, then create provider, idle agent, workflow, and ticket")
        local_machine = require_single_local_machine(base_url, org["id"])
        provider = request_json(
            base_url,
            "POST",
            f"/api/v1/orgs/{org['id']}/providers",
            {
                "machine_id": local_machine["id"],
                "name": "Codex Smoke Provider",
                "adapter_type": "codex-app-server",
                "cli_command": "codex",
                "cli_args": ["app-server", "--listen", "stdio://"],
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
                "name": f"smoke-agent-{stamp}",
            },
        )["agent"]
        assert_idle_agent_defaults(agent)
        workflow = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/workflows",
            {
                "agent_id": agent["id"],
                "name": workflow_name,
                "type": "coding",
                "pickup_status_ids": [todo["id"]],
                "finish_status_ids": [done["id"]],
                "harness_content": "# Blackbox Workflow\n\nKeep the implementation focused on the current ticket.\n",
            },
        )["workflow"]
        ticket = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/tickets",
            {
                "title": "Blackbox coding ticket",
                "description": "Created by local black-box smoke test.",
                "priority": "high",
                "workflow_id": workflow["id"],
                "created_by": "user:blackbox",
            },
        )["ticket"]

        print("[5/11] attach repo scope and validate ticket detail")
        existing_scopes = request_json(
            base_url,
            "GET",
            f"/api/v1/projects/{project['id']}/tickets/{ticket['id']}/repo-scopes",
        )["repo_scopes"]
        repo_scope = next((item for item in existing_scopes if item.get("repo_id") == repo["id"]), None)
        if repo_scope is None:
            repo_scope = request_json(
                base_url,
                "POST",
                f"/api/v1/projects/{project['id']}/tickets/{ticket['id']}/repo-scopes",
                {
                    "repo_id": repo["id"],
                    "branch_name": f"blackbox/{stamp}",
                },
            )["repo_scope"]
        detail = request_json(
            base_url,
            "GET",
            f"/api/v1/projects/{project['id']}/tickets/{ticket['id']}/detail",
        )
        if not any(scope["id"] == repo_scope["id"] for scope in detail["repo_scopes"]):
            raise RuntimeError("ticket detail did not include the repo scope that was just created")

        print("[6/11] issue default agent token")
        token_payload = run_cli_json(
            [
                openase_bin,
                "issue-agent-token",
                "--agent-id",
                agent["id"],
                "--project-id",
                project["id"],
                "--ticket-id",
                ticket["id"],
                "--api-url",
                base_url + "/api/v1/platform",
            ]
        )

        agent_env = os.environ.copy()
        agent_env.update(token_payload["environment"])

        print("[7/11] verify agent CLI ticket list/create/update")
        ticket_list = run_cli_json([openase_bin, "ticket", "list"], env=agent_env)
        if not any(item["id"] == ticket["id"] for item in ticket_list["tickets"]):
            raise RuntimeError("agent ticket list did not include the current ticket")

        created_by_agent = run_cli_json(
            [
                openase_bin,
                "ticket",
                "create",
                "--title",
                "Agent-created follow-up",
                "--description",
                "Created via agent platform CLI smoke test.",
                "--priority",
                "medium",
            ],
            env=agent_env,
        )["ticket"]
        if created_by_agent["created_by"] != f"agent:{agent['name']}":
            raise RuntimeError(
                f"expected agent-created ticket marker agent:{agent['name']}, got {created_by_agent['created_by']}"
            )

        updated_current = run_cli_json(
            [
                openase_bin,
                "ticket",
                "update",
                "--description",
                "Updated via agent platform CLI smoke test.",
                "--external-ref",
                f"blackbox/{stamp}",
            ],
            env=agent_env,
        )["ticket"]
        if updated_current["description"] != "Updated via agent platform CLI smoke test.":
            raise RuntimeError(f"unexpected updated ticket payload: {updated_current!r}")

        print("[8/11] verify default token cannot mutate project")
        forbidden = run_cli(
            [
                openase_bin,
                "project",
                "update",
                "--description",
                "This should be rejected for a default-scoped token.",
            ],
            env=agent_env,
            check=False,
        )
        if forbidden.returncode == 0:
            raise RuntimeError("default-scoped token unexpectedly updated the project")
        forbidden_output = forbidden.stdout + forbidden.stderr
        if "missing required scope projects.update" not in forbidden_output:
            raise RuntimeError(
                "expected scope failure for default token, got:\n"
                + forbidden_output
            )

        print("[9/11] issue privileged token")
        privileged_token_payload = run_cli_json(
            [
                openase_bin,
                "issue-agent-token",
                "--agent-id",
                agent["id"],
                "--project-id",
                project["id"],
                "--ticket-id",
                ticket["id"],
                "--api-url",
                base_url + "/api/v1/platform",
                "--scope",
                "projects.update",
                "--scope",
                "projects.add_repo",
                "--scope",
                "tickets.create",
                "--scope",
                "tickets.list",
                "--scope",
                "tickets.update.self",
            ]
        )
        privileged_env = os.environ.copy()
        privileged_env.update(privileged_token_payload["environment"])

        print("[10/11] verify privileged token can update project and add repo")
        updated_project = run_cli_json(
            [
                openase_bin,
                "project",
                "update",
                "--description",
                "Updated by black-box privileged agent token.",
            ],
            env=privileged_env,
        )["project"]
        if updated_project["description"] != "Updated by black-box privileged agent token.":
            raise RuntimeError(f"unexpected project update payload: {updated_project!r}")

        created_repo = run_cli_json(
            [
                openase_bin,
                "project",
                "add-repo",
                "--name",
                agent_repo_name,
                "--url",
                "https://github.com/acme/agent-tools.git",
                "--label",
                "smoke",
                "--label",
                "agent",
            ],
            env=privileged_env,
        )["repo"]
        if created_repo["name"] != agent_repo_name:
            raise RuntimeError(f"unexpected repo create payload: {created_repo!r}")

        print("[11/11] final readback checks")
        project_after = request_json(base_url, "GET", f"/api/v1/projects/{project['id']}")["project"]
        if project_after["description"] != "Updated by black-box privileged agent token.":
            raise RuntimeError(f"project readback mismatch: {project_after!r}")
        repos_after = request_json(base_url, "GET", f"/api/v1/projects/{project['id']}/repos")["repos"]
        if not any(item["id"] == created_repo["id"] for item in repos_after):
            raise RuntimeError("repo created by privileged token was not returned by project repo list")

        summary = {
            "organization_id": org["id"],
            "project_id": project["id"],
            "workflow_id": workflow["id"],
            "provider_id": provider["id"],
            "agent_id": agent["id"],
            "ticket_id": ticket["id"],
            "repo_id": repo["id"],
            "agent_created_ticket_id": created_by_agent["id"],
            "privileged_repo_id": created_repo["id"],
        }
        print(json.dumps(summary, indent=2))
        return 0
    finally:
        cleanup_harness_artifacts(primary_repo_root, workflow)
        if primary_repo_root is not None:
            shutil.rmtree(primary_repo_root, ignore_errors=True)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
