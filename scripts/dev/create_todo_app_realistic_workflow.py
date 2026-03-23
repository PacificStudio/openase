#!/usr/bin/env python3

import argparse
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


def build_parser() -> argparse.ArgumentParser:
    repo_root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(
        description=(
            "Create a realistic OpenASE validation workflow: project, coding workflow, coding agent, "
            "GitHub project, GitHub issues, and linked OpenASE tickets."
        )
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("OPENASE_BASE_URL", "http://127.0.0.1:19836"),
        help="OpenASE base URL, for example http://127.0.0.1:19836",
    )
    parser.add_argument(
        "--project-name",
        default="Todo App",
        help="Display name for the created OpenASE project.",
    )
    parser.add_argument(
        "--workspace-path",
        default=str(repo_root),
        help="Absolute workspace path passed into the created agent.",
    )
    parser.add_argument(
        "--provider-mode",
        choices=("fake-codex", "real-codex"),
        default="fake-codex",
        help=(
            "Provider implementation for the created coding agent. "
            "fake-codex is deterministic; real-codex uses `codex app-server --listen stdio://`."
        ),
    )
    parser.add_argument(
        "--wait-seconds",
        type=int,
        default=45,
        help="Maximum seconds to wait for the scheduler to claim a ticket.",
    )
    parser.add_argument(
        "--skip-github",
        action="store_true",
        help="Skip GitHub project and issue creation; only create OpenASE resources.",
    )
    parser.add_argument(
        "--github-owner",
        default="@me",
        help='Owner for the GitHub Project. Use "@me" for the authenticated user.',
    )
    parser.add_argument(
        "--github-repo",
        default="",
        help="Repository used for GitHub issues, in OWNER/REPO form. Defaults to origin when possible.",
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


def run_cli(command: list[str], cwd: Path | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        command,
        cwd=str(cwd) if cwd else None,
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


def run_cli_json(command: list[str], cwd: Path | None = None):
    result = run_cli(command, cwd=cwd, check=True)
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


def require_by_name(items, key: str, want: str):
    for item in items:
        if item.get(key) == want:
            return item
    raise RuntimeError(f"could not find item with {key}={want!r} in {items!r}")


def slugify(raw: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", raw.lower()).strip("-")
    slug = re.sub(r"-{2,}", "-", slug)
    return slug or "item"


def parse_github_repo(raw: str) -> str:
    trimmed = raw.strip()
    if not trimmed:
        raise RuntimeError("github repo must not be empty")
    if trimmed.endswith(".git"):
        trimmed = trimmed[:-4]
    if trimmed.startswith("https://github.com/"):
        trimmed = trimmed.removeprefix("https://github.com/")
    if trimmed.startswith("git@github.com:"):
        trimmed = trimmed.removeprefix("git@github.com:")
    if len(trimmed.split("/")) != 2:
        raise RuntimeError(f"expected OWNER/REPO GitHub repo, got {raw!r}")
    return trimmed


def detect_origin_github_repo(repo_root: Path) -> str | None:
    result = run_cli(["git", "config", "--get", "remote.origin.url"], cwd=repo_root, check=False)
    if result.returncode != 0:
        return None
    raw = result.stdout.strip()
    if "github.com" not in raw:
        return None
    try:
        return parse_github_repo(raw)
    except RuntimeError:
        return None


def create_github_project(owner: str, title: str) -> dict:
    run_cli(["gh", "project", "create", "--owner", owner, "--title", title], check=True)
    return get_github_project_by_title(owner, title)


def get_github_project_by_title(owner: str, title: str) -> dict:
    projects_payload = run_cli_json(["gh", "project", "list", "--owner", owner, "--format", "json"])
    for project in projects_payload.get("projects", []):
        if project.get("title") == title:
            return project
    raise RuntimeError(f"could not find newly created GitHub project {title!r}")


def create_github_issue(repo: str, title: str, body: str, project_title: str | None) -> dict:
    command = ["gh", "issue", "create", "-R", repo, "--title", title, "--body", body]
    if project_title:
        command.extend(["--project", project_title])
    result = run_cli(command, check=True)
    issue_url = result.stdout.strip().splitlines()[-1].strip()
    if not issue_url.startswith("https://github.com/"):
        raise RuntimeError(f"unexpected issue create output: {result.stdout!r}")
    issue_number = int(issue_url.rstrip("/").rsplit("/", 1)[-1])
    return {
        "number": issue_number,
        "url": issue_url,
        "external_id": f"{repo}#{issue_number}",
    }


def wait_for_agent_claim(base_url: str, project_id: str, agent_id: str, timeout_seconds: int) -> dict | None:
    deadline = time.time() + timeout_seconds
    last_seen = None
    while time.time() < deadline:
        agents = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/agents").get("agents", [])
        current = require_by_name(agents, "id", agent_id)
        last_seen = current
        if current.get("status") in ("claimed", "running") and current.get("current_ticket_id"):
            return current
        time.sleep(1)
    return last_seen


def main() -> int:
    args = build_parser().parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    workspace_path = Path(args.workspace_path).resolve()
    if not workspace_path.is_absolute():
        raise RuntimeError("--workspace-path must be absolute")

    stamp = time.strftime("%Y%m%d%H%M%S")
    base_url = args.base_url.rstrip("/")
    project_name = args.project_name.strip() or "Todo App"
    org_slug = f"{slugify(project_name)}-validation-{stamp}"
    project_slug = f"{slugify(project_name)}-{stamp}"
    workflow_name = f"{project_name} Coding Workflow"
    github_repo = ""
    github_project = None
    github_items = []

    if not args.skip_github:
        github_repo = args.github_repo.strip() or detect_origin_github_repo(repo_root) or ""
        if not github_repo:
            raise RuntimeError("unable to determine --github-repo from origin; pass --github-repo explicitly")
        github_repo = parse_github_repo(github_repo)

    todo_issue_specs = [
        {
            "title": f"[{project_name}] Scaffold app shell and storage model",
            "body": (
                "Create the initial Todo app shell, define the task storage model, and document the "
                "basic architecture choices."
            ),
            "priority": "high",
        },
        {
            "title": f"[{project_name}] Implement add / toggle / delete flows",
            "body": (
                "Implement the core task interactions for creating todos, marking them complete, and "
                "removing them."
            ),
            "priority": "high",
        },
        {
            "title": f"[{project_name}] Add filtering, counts, and regression tests",
            "body": (
                "Support active/completed filters, item counters, and add focused regression coverage "
                "for the Todo workflow."
            ),
            "priority": "medium",
        },
    ]

    print(f"[1/10] health check against {base_url}")
    request_json(base_url, "GET", "/healthz")
    request_json(base_url, "GET", "/api/v1/healthz")

    if not args.skip_github:
        print("[2/10] verify GitHub CLI auth and create a dedicated GitHub Project")
        run_cli(["gh", "auth", "status"], check=True)
        github_project_title = f"OpenASE {project_name} Validation {stamp}"
        github_project = create_github_project(args.github_owner, github_project_title)
        for spec in todo_issue_specs:
            body = (
                spec["body"]
                + "\n\n"
                + "Created by `scripts/dev/create_todo_app_realistic_workflow.py` for end-to-end "
                + "OpenASE workflow validation."
            )
            github_issue = create_github_issue(github_repo, spec["title"], body, github_project_title)
            github_items.append(github_issue)
        github_project = get_github_project_by_title(args.github_owner, github_project_title)
    else:
        print("[2/10] skip GitHub project and issue creation")

    print("[3/10] create isolated OpenASE organization and project")
    org = request_json(
        base_url,
        "POST",
        "/api/v1/orgs",
        {
            "name": f"{project_name} Validation {stamp}",
            "slug": org_slug,
        },
    )["organization"]
    project = request_json(
        base_url,
        "POST",
        f"/api/v1/orgs/{org['id']}/projects",
        {
            "name": project_name,
            "slug": project_slug,
            "description": "Realistic Todo app workflow validation created by the dev seed script.",
            "status": "active",
            "max_concurrent_agents": 1,
        },
    )["project"]

    print("[4/10] seed default statuses and create coding workflow")
    statuses = request_json(base_url, "POST", f"/api/v1/projects/{project['id']}/statuses/reset")["statuses"]
    todo = require_by_name(statuses, "name", "Todo")
    done = require_by_name(statuses, "name", "Done")
    workflow = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/workflows",
        {
            "name": workflow_name,
            "type": "coding",
            "pickup_status_id": todo["id"],
            "finish_status_id": done["id"],
            "harness_content": (
                "---\n"
                "workflow:\n"
                "  role: coding\n"
                "---\n\n"
                f"# {project_name}\n\n"
                "You are responsible for implementing coding tasks for the Todo app project.\n"
                "Prefer concrete code changes, tests, and short execution notes.\n"
            ),
        },
    )["workflow"]

    print("[5/10] create provider and coding agent")
    if args.provider_mode == "fake-codex":
        fake_codex_path = repo_root / "scripts" / "dev" / "fake_codex_app_server.py"
        provider_payload = {
            "name": "Fake Codex Validation Provider",
            "adapter_type": "codex-app-server",
            "cli_command": os.environ.get("PYTHON", "python3"),
            "cli_args": [str(fake_codex_path)],
            "auth_config": {},
            "model_name": "gpt-5.4",
        }
    else:
        provider_payload = {
            "name": "Codex Validation Provider",
            "adapter_type": "codex-app-server",
            "cli_command": "codex",
            "cli_args": ["app-server", "--listen", "stdio://"],
            "auth_config": {},
            "model_name": "gpt-5.4",
        }

    provider = request_json(
        base_url,
        "POST",
        f"/api/v1/orgs/{org['id']}/providers",
        provider_payload,
    )["provider"]
    agent = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/agents",
        {
            "provider_id": provider["id"],
            "name": f"{slugify(project_name)}-coding-01",
            "workspace_path": str(workspace_path),
            "capabilities": ["coding", "todo-app", "validation"],
        },
    )["agent"]

    print("[6/10] set project defaults and register the Git repo for the workflow")
    request_json(
        base_url,
        "PATCH",
        f"/api/v1/projects/{project['id']}",
        {
            "default_agent_provider_id": provider["id"],
            "default_workflow_id": workflow["id"],
        },
    )
    project_repo = request_json(
        base_url,
        "POST",
        f"/api/v1/projects/{project['id']}/repos",
        {
            "name": slugify(project_name),
            "repository_url": f"https://github.com/{github_repo}.git" if github_repo else "https://github.com/BetterAndBetterII/openase.git",
            "default_branch": "main",
            "is_primary": True,
            "labels": ["todo-app", "validation"],
        },
    )["repo"]

    print("[7/10] create linked OpenASE tickets")
    tickets = []
    for index, spec in enumerate(todo_issue_specs):
        github_issue = github_items[index] if index < len(github_items) else None
        description = spec["body"]
        if github_issue is not None:
            description += f"\n\nLinked GitHub issue: {github_issue['url']}"
        ticket = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/tickets",
            {
                "title": spec["title"],
                "description": description,
                "status_id": todo["id"],
                "priority": spec["priority"],
                "workflow_id": workflow["id"],
                "created_by": "user:workflow-seed",
                "external_ref": github_issue["external_id"] if github_issue else None,
            },
        )["ticket"]
        tickets.append(ticket)

        if github_issue is not None:
            request_json(
                base_url,
                "POST",
                f"/api/v1/tickets/{ticket['id']}/external-links",
                {
                    "type": "github_issue",
                    "url": github_issue["url"],
                    "external_id": github_issue["external_id"],
                    "title": spec["title"],
                    "status": "open",
                    "relation": "related",
                },
            )
            ticket = request_json(base_url, "GET", f"/api/v1/tickets/{ticket['id']}")["ticket"]
            tickets[-1] = ticket

    print("[8/10] add one realistic dependency edge")
    request_json(
        base_url,
        "POST",
        f"/api/v1/tickets/{tickets[0]['id']}/dependencies",
        {
            "target_ticket_id": tickets[1]["id"],
            "type": "blocks",
        },
    )

    print("[9/10] wait for the scheduler to claim work")
    agent_after_claim = wait_for_agent_claim(base_url, project["id"], agent["id"], args.wait_seconds)

    print("[10/10] summarize created resources")
    summary = {
        "openase": {
            "base_url": base_url,
            "organization": org,
            "project": project,
            "project_repo": project_repo,
            "workflow": workflow,
            "provider": provider,
            "agent": agent,
            "agent_after_wait": agent_after_claim,
            "tickets": tickets,
        },
        "github": {
            "enabled": not args.skip_github,
            "owner": args.github_owner,
            "repo": github_repo or None,
            "project": github_project,
            "issues": github_items,
        },
        "notes": [
            "OpenASE project-facing connector CRUD is not exported yet.",
            "This script creates real GitHub issues/projects, then links them to OpenASE tickets through external links.",
            f"Provider mode: {args.provider_mode}",
        ],
    }
    print(json.dumps(summary, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
