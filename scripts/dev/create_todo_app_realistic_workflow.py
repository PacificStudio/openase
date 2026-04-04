#!/usr/bin/env python3

import argparse
import json
import os
import re
import subprocess
import textwrap
import time
import urllib.error
import urllib.request
from pathlib import Path

FIXED_GITHUB_REPO = "BetterAndBetterII/TodoApp"
FIXED_GITHUB_REPO_URL = "https://github.com/BetterAndBetterII/TodoApp.git"
FIXED_GITHUB_SSH_URL = "git@github.com:BetterAndBetterII/TodoApp.git"
GIT_AUTHOR_NAME = "Codex"
GIT_AUTHOR_EMAIL = "codex@openai.com"
README_BASELINE = (
    "# TodoApp\n\n"
    "OpenASE validation repository baseline.\n"
)
PROJECT_STATUS_IN_PROGRESS = "In Progress"


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
        "--workspace-parent",
        default=str(repo_root.parent),
        help="Parent directory used for temporary GitHub checkout preparation.",
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
        help="Maximum seconds to wait for the scheduler to claim a ticket and observe runtime activity.",
    )
    parser.add_argument(
        "--wait-for-workspace-seconds",
        type=int,
        default=180,
        help="Maximum seconds to wait for real coding activity to land in the Todo app workspace repo.",
    )
    parser.add_argument(
        "--skip-github",
        action="store_true",
        help=(
            "Skip GitHub Project and issue creation. The validation still prepares "
            f"the fixed repository {FIXED_GITHUB_REPO_URL} and links every ticket to a PR."
        ),
    )
    parser.add_argument(
        "--github-owner",
        default="@me",
        help='Owner for the GitHub Project. Use "@me" for the authenticated user.',
    )
    parser.add_argument(
        "--platform-skill-timeout-seconds",
        type=int,
        default=180,
        help="Maximum seconds to wait for the agent to post or update the current ticket workpad via the platform skill.",
    )
    parser.add_argument(
        "--require-platform-skill",
        action="store_true",
        help=(
            "Fail unless the claimed ticket receives an agent-owned Workpad comment update. "
            "This is enabled implicitly for real-codex mode."
        ),
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


def slugify(raw: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", raw.lower()).strip("-")
    slug = re.sub(r"-{2,}", "-", slug)
    return slug or "item"


def upsert_ticket_repo_scope(
    base_url: str,
    project_id: str,
    ticket_id: str,
    repo_id: str,
    branch_name: str,
    pull_request_url: str,
    pr_status: str,
    ci_status: str,
) -> dict:
    payload = {
        "branch_name": branch_name,
        "pull_request_url": pull_request_url,
        "pr_status": pr_status,
        "ci_status": ci_status,
    }
    current_scopes = request_json(
        base_url,
        "GET",
        f"/api/v1/projects/{project_id}/tickets/{ticket_id}/repo-scopes",
    ).get("repo_scopes", [])
    existing_scope = next((scope for scope in current_scopes if scope.get("repo_id") == repo_id), None)
    if existing_scope is None:
        response = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project_id}/tickets/{ticket_id}/repo-scopes",
            {
                "repo_id": repo_id,
                **payload,
            },
        )
    else:
        response = request_json(
            base_url,
            "PATCH",
            f"/api/v1/projects/{project_id}/tickets/{ticket_id}/repo-scopes/{existing_scope['id']}",
            payload,
        )
    repo_scope = response.get("repo_scope")
    if not isinstance(repo_scope, dict):
        raise RuntimeError(f"repo scope mutation returned an unexpected payload: {response!r}")
    return repo_scope


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


def ensure_git_identity(repo_dir: Path) -> None:
    run_cli(["git", "config", "user.name", GIT_AUTHOR_NAME], cwd=repo_dir, check=True)
    run_cli(["git", "config", "user.email", GIT_AUTHOR_EMAIL], cwd=repo_dir, check=True)


def ensure_main_checkout(repo_dir: Path) -> None:
    branch_result = run_cli(["git", "rev-parse", "--verify", "main"], cwd=repo_dir, check=False)
    if branch_result.returncode == 0:
        run_cli(["git", "checkout", "main"], cwd=repo_dir, check=True)
        return
    run_cli(["git", "checkout", "-b", "main"], cwd=repo_dir, check=True)


def ensure_repo_readme(repo_dir: Path) -> None:
    readme_path = repo_dir / "README.md"
    if not readme_path.exists():
        write_text(readme_path, README_BASELINE)


def build_ticket_branch_name(ticket_identifier: str) -> str:
    return f"agent/{ticket_identifier}"


def derive_ticket_workspace_repo_path(org_slug: str, project_slug: str, ticket_identifier: str, repo_name: str) -> Path:
    return Path.home() / ".openase" / "workspace" / org_slug / project_slug / ticket_identifier / repo_name


def prepare_github_validation_repository(repo: str, checkout_parent: Path, stamp: str) -> dict:
    repo_meta = run_cli_json(["gh", "api", f"repos/{repo}"])
    checkout_dir = checkout_parent / f"todoapp-github-validation-{stamp}"
    repo_dir = checkout_dir / "repo"
    default_branch = repo_meta.get("default_branch") or "main"
    checkout_dir.mkdir(parents=True, exist_ok=False)
    remote_default_branch = run_cli(
        ["git", "ls-remote", "--heads", FIXED_GITHUB_SSH_URL, default_branch],
        check=False,
    )
    remote_has_default_branch = bool(remote_default_branch.stdout.strip())

    if not remote_has_default_branch:
        repo_dir.mkdir(parents=True, exist_ok=False)
        run_cli(["git", "init", "-b", default_branch], cwd=repo_dir, check=True)
        ensure_git_identity(repo_dir)
        ensure_repo_readme(repo_dir)
        run_cli(["git", "add", "README.md"], cwd=repo_dir, check=True)
        run_cli(["git", "commit", "-m", "docs: bootstrap todoapp readme"], cwd=repo_dir, check=True)
        run_cli(["git", "remote", "add", "origin", FIXED_GITHUB_SSH_URL], cwd=repo_dir, check=True)
        run_cli(["git", "push", "-u", "origin", f"HEAD:{default_branch}"], cwd=repo_dir, check=True)
    else:
        run_cli(["git", "clone", FIXED_GITHUB_SSH_URL, str(repo_dir)], cwd=checkout_dir, check=True)
        ensure_git_identity(repo_dir)
        ensure_main_checkout(repo_dir)
        run_cli(["git", "pull", "--ff-only", "origin", default_branch], cwd=repo_dir, check=True)

    ensure_repo_readme(repo_dir)
    run_cli(["git", "add", "README.md"], cwd=repo_dir, check=True)
    run_cli(
        ["git", "commit", "--allow-empty", "-m", f"docs: baseline readme checkpoint {stamp}"],
        cwd=repo_dir,
        check=True,
    )
    run_cli(["git", "push", "origin", f"HEAD:{default_branch}"], cwd=repo_dir, check=True)
    baseline_head = run_cli(["git", "rev-parse", "HEAD"], cwd=repo_dir, check=True).stdout.strip()
    return {
        "checkout_dir": checkout_dir,
        "repo_dir": repo_dir,
        "baseline_head": baseline_head,
        "default_branch": default_branch,
        "clone_url": repo_meta.get("clone_url") or FIXED_GITHUB_REPO_URL,
        "ssh_url": repo_meta.get("ssh_url") or FIXED_GITHUB_SSH_URL,
    }


def create_github_pull_request(
    repo: str,
    repo_dir: Path,
    branch_name: str,
    base_sha: str,
    title: str,
    body: str,
) -> dict:
    ensure_main_checkout(repo_dir)
    run_cli(["git", "checkout", "-B", branch_name, base_sha], cwd=repo_dir, check=True)
    seed_rel_path = Path(".openase") / "pr-seeds" / f"{branch_name}.md"
    write_text(
        repo_dir / seed_rel_path,
        textwrap.dedent(
            f"""\
            # OpenASE Validation PR Seed

            Branch: `{branch_name}`

            This commit exists so the validation script can open a deterministic pull request
            for the matching OpenASE ticket.
            """
        ),
    )
    run_cli(["git", "add", str(seed_rel_path)], cwd=repo_dir, check=True)
    run_cli(["git", "commit", "-m", f"chore: seed validation pr for {branch_name}"], cwd=repo_dir, check=True)
    run_cli(["git", "push", "--force-with-lease", "-u", "origin", branch_name], cwd=repo_dir, check=True)
    existing_pull_requests = run_cli_json(
        ["gh", "pr", "list", "-R", repo, "--head", branch_name, "--state", "open", "--json", "number,url"],
        cwd=repo_dir,
    )
    if existing_pull_requests:
        existing = existing_pull_requests[0]
        return {
            "number": existing["number"],
            "url": existing["url"],
            "external_id": f"{repo}#{existing['number']}",
            "branch_name": branch_name,
        }
    result = run_cli(
        [
            "gh",
            "pr",
            "create",
            "-R",
            repo,
            "--base",
            "main",
            "--head",
            branch_name,
            "--title",
            title,
            "--body",
            body,
            "--draft",
        ],
        cwd=repo_dir,
        check=True,
    )
    pr_url = result.stdout.strip().splitlines()[-1].strip()
    if not pr_url.startswith("https://github.com/"):
        raise RuntimeError(f"unexpected pull request create output: {result.stdout!r}")
    pr_number = int(pr_url.rstrip("/").rsplit("/", 1)[-1])
    return {
        "number": pr_number,
        "url": pr_url,
        "external_id": f"{repo}#{pr_number}",
        "branch_name": branch_name,
    }


def wait_for_agent_claim(base_url: str, project_id: str, agent_id: str, timeout_seconds: int) -> dict | None:
    deadline = time.time() + timeout_seconds
    last_seen = None
    while time.time() < deadline:
        agents = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/agents").get("agents", [])
        current = require_by_name(agents, "id", agent_id)
        last_seen = current
        runtime = current.get("runtime") if isinstance(current.get("runtime"), dict) else {}
        status = runtime.get("status") or current.get("status")
        current_ticket_id = runtime.get("current_ticket_id") or current.get("current_ticket_id")
        if status in ("claimed", "running") and current_ticket_id:
            return current
        time.sleep(1)
    return last_seen


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def wait_for_agent_execution(base_url: str, project_id: str, agent_id: str, timeout_seconds: int) -> dict | None:
    deadline = time.time() + timeout_seconds
    last_seen = None
    while time.time() < deadline:
        agents = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/agents").get("agents", [])
        current = require_by_name(agents, "id", agent_id)
        last_seen = current
        runtime = current.get("runtime") if isinstance(current.get("runtime"), dict) else {}
        if runtime.get("runtime_phase") == "executing" or current.get("total_tokens_used", 0) > 0:
            return current
        time.sleep(2)
    return last_seen


def wait_for_workspace_activity(workspace_path: Path, baseline_head: str, timeout_seconds: int) -> dict | None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        if not workspace_path.exists() or not (workspace_path / ".git").exists():
            time.sleep(5)
            continue
        status_output = run_cli(["git", "status", "--short"], cwd=workspace_path, check=True).stdout.strip()
        current_head = run_cli(["git", "rev-parse", "HEAD"], cwd=workspace_path, check=True).stdout.strip()
        if status_output or current_head != baseline_head:
            diff_stat = run_cli(["git", "diff", "--stat"], cwd=workspace_path, check=False).stdout.strip()
            last_commit = run_cli(["git", "log", "--oneline", "-1"], cwd=workspace_path, check=True).stdout.strip()
            recent_files = run_cli(["git", "status", "--short"], cwd=workspace_path, check=True).stdout.strip().splitlines()
            return {
                "observed_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "head": current_head,
                "head_changed": current_head != baseline_head,
                "git_status": status_output,
                "git_diff_stat": diff_stat,
                "last_commit": last_commit,
                "changed_entries": recent_files,
            }
        time.sleep(5)
    return None


def wait_for_ticket_workpad_comment_by_agent(
    base_url: str,
    project_id: str,
    ticket_id: str,
    expected_actor: str,
    timeout_seconds: int,
) -> dict | None:
    deadline = time.time() + timeout_seconds
    last_seen = None
    while time.time() < deadline:
        detail = request_json(base_url, "GET", f"/api/v1/projects/{project_id}/tickets/{ticket_id}/detail")
        ticket = detail.get("ticket")
        comments = detail.get("comments")
        if not isinstance(ticket, dict) or not isinstance(comments, list):
            time.sleep(2)
            continue
        last_seen = {"ticket": ticket, "comments": comments}
        for comment in comments:
            if not isinstance(comment, dict):
                continue
            if comment.get("is_deleted"):
                continue
            body_markdown = comment.get("body_markdown")
            if not isinstance(body_markdown, str):
                continue
            if not body_markdown.lstrip().startswith("## Workpad"):
                continue
            if comment.get("created_by") != expected_actor and comment.get("last_edited_by") != expected_actor:
                continue
            return {
                "detected": True,
                "detected_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "ticket": ticket,
                "comment": comment,
            }
        time.sleep(2)
    if last_seen is None:
        return None
    return {
        "detected": False,
        "detected_at": None,
        "ticket": last_seen.get("ticket"),
        "comments": last_seen.get("comments"),
    }


def build_validation_workflow_harness(project_name: str) -> str:
    return textwrap.dedent(
        f"""\
        ---
        workflow:
          role: coding
        skills:
          - openase-platform
          - ticket-workpad
          - commit
          - push
          - create-pr
        ---

        你正在处理 OpenASE 分配的 {project_name} 工单 `{{{{ ticket.identifier }}}}`。

        {{% if attempt > 1 %}}
        续跑上下文：

        - 当前是第 {{{{ attempt }}}} 次尝试，最大允许 {{{{ max_attempts }}}} 次。
        - 直接基于当前工作区继续，不要重新初始化项目，不要重复已经完成的排查、实现和验证。
        - 如果之前已经有部分实现、测试或注释，优先在原有基础上收敛并补齐当前工单目标。
        {{% endif %}}

        项目上下文：

        - 当前项目名称：`{{{{ project.name }}}}`。
        - 当前 Workflow：`{{{{ workflow.name }}}}`（type=`{{{{ workflow.type }}}}`，pickup=`{{{{ workflow.pickup_status }}}}`，finish=`{{{{ workflow.finish_status }}}}`）。
        - 对这条 seeded coding workflow 来说，`{{{{ workflow.finish_status }}}}` 表示“代码已经提交并推送，关联 PR 已准备好进入 review”，不表示最终已经 merged。
        - 这是一个用于 OpenASE 端到端验证的独立 {project_name} 仓库，不是 OpenASE 平台仓库本体。
        - 目标是在一个轻量、无复杂构建步骤的前端仓库里完成真实编码任务，验证工单 -> workflow -> agent -> workspace 这条链路能稳定落地。
        - 默认期望保留当前“纯静态页面 + 原生 JavaScript + `node --test`”的轻量结构，除非当前工单明确要求，否则不要引入额外框架、打包器或重型依赖。

        工单信息：

        - 编号：`{{{{ ticket.identifier }}}}`
        - 标题：`{{{{ ticket.title }}}}`
        - 当前状态：`{{{{ ticket.status }}}}`
        - 优先级：`{{{{ ticket.priority }}}}`
        - 类型：`{{{{ ticket.type }}}}`
        - 创建者：`{{{{ ticket.created_by }}}}`
        - OpenASE 工单链接：`{{{{ ticket.url | default("未提供") }}}}`
        - 外部主关联：`{{{{ ticket.external_ref | default("无") }}}}`

        工单描述：
        {{% if ticket.description %}}
        {{{{ ticket.description }}}}
        {{% else %}}
        未提供额外描述。
        {{% endif %}}

        {{% if ticket.links %}}
        外部链接：
        {{% for link in ticket.links %}}
        - [{{{{ link.type }}}}] {{{{ link.title | default("untitled") }}}} (status=`{{{{ link.status | default("unknown") }}}}`, relation=`{{{{ link.relation | default("related") }}}}`): {{{{ link.url }}}}
        {{% endfor %}}
        {{% endif %}}

        {{% if ticket.dependencies %}}
        依赖工单：
        {{% for dependency in ticket.dependencies %}}
        - `{{{{ dependency.identifier }}}}` {{{{ dependency.title }}}} (type=`{{{{ dependency.type }}}}`, status=`{{{{ dependency.status }}}}`)
        {{% endfor %}}
        {{% endif %}}

        工作区与仓库：

        - 工作区根目录：`{{{{ workspace }}}}`
        - 当前执行机器：`{{{{ machine.name }}}}` @ `{{{{ machine.host }}}}`
        - 当前 Agent：`{{{{ agent.name }}}}`（provider=`{{{{ agent.provider }}}}`, model=`{{{{ agent.model }}}}`）

        {{% if repos %}}
        当前工单涉及以下仓库：
        {{% for repo in repos %}}
        - `{{{{ repo.name }}}}`
          path=`{{{{ repo.path }}}}`
          branch=`{{{{ repo.branch | default(repo.default_branch) }}}}`
          default_branch=`{{{{ repo.default_branch }}}}`
          labels=`{{{{ repo.labels | join(", ") | default("none") }}}}`
        {{% endfor %}}
        {{% else %}}
        当前工单没有显式 repo scope；如果项目只有一个仓库，可直接使用该仓库；否则必须先明确 repo scope。
        {{% endif %}}

        执行目标：

        - 只解决当前工单要求的那一块，不要把整个 Todo App 一次性重写到超出工单范围。
        - 交付必须是“可运行、可验证、可读”的真实代码，而不是停留在方案或注释层。
        - 优先形成最小完整垂直切片：实现功能、补齐必要样式/DOM、补充或更新测试，然后验证。
        - 如果当前工单只要求其中一个子能力，例如 app shell、storage model、add/toggle/delete、filter/count、regression tests，就只把这一块做扎实。

        工作台要求：

        - 当前 harness 已绑定 `ticket-workpad` 和 `openase-platform` skill；开始执行前，先用 skill 在当前工单下创建或更新一条标题为 `## Workpad` 的评论。
        - `## Workpad` 是当前工单唯一的持久化进度板；计划、当前进展、验证结果、剩余风险和阻塞都持续更新到这一条评论，不要每次新建评论。
        - 第一版 workpad 至少包含：
          - `Environment`：`<host>:<abs-workdir>@<short-sha>`
          - `Plan`
          - `Progress`
          - `Validation`
          - `Notes`
        - 在开始改代码前先写第一版 workpad；每完成一个关键阶段后都刷新它，至少覆盖：完成阅读、完成实现、完成测试、准备结束工单。
        - 如果执行过程中发现假设、scope 调整或阻塞，先更新 workpad，再继续动作或结束执行。

        全局规则：

        1. 这是无人值守执行，不要等待人类额外输入。
        2. 只在当前 Todo App 工作区及其相关仓库中修改文件，不要去改 OpenASE 仓库本体。
        3. 开工前先阅读与当前工单直接相关的文件，至少包括 `README.md`、`package.json`、`src/`、`test/`。
        4. 默认遵守现有技术路线：原生 HTML/CSS/JS、小而清晰的模块边界、最少依赖、无额外构建步骤。
        5. 保持实现面向真实用户体验：交互清晰、命名明确、结构可维护，不要只做能糊过测试的最小字符串修改。
        6. 任何行为变更都应同时考虑测试；修改功能时，优先补或改 `node --test` 覆盖。
        7. 不要引入与工单无关的大规模重构，不要顺手重命名整个项目或重排所有文件。
        8. 如果发现现有代码与工单目标冲突，优先做局部、可解释的调整，并在最终说明中写清取舍。
        9. 如果你新增存储、过滤、计数或派生状态逻辑，优先让数据模型和渲染逻辑保持清晰，而不是把判断散落到多个事件处理器中。
        10. 若需要命令验证，优先使用仓库现有命令，例如 `npm test`；如果增加新的验证方式，必须保持轻量且与当前仓库结构匹配。
        11. 对 Git 交付链路要有完整收口：代码改动完成后，要形成真实 commit、推送当前 ticket branch，并确认关联 PR 已包含最新变更；不要把仅存在于脏工作区的未提交改动当成交付完成。

        平台状态控制要求：

        - 需要操作 OpenASE 平台时，优先通过 skill 提供的 `./.openase/bin/openase ...` 包装命令完成，而不是自己拼接原始 HTTP 请求。
        - 当前工单状态控制是交付的一部分，不要只改代码不回写平台。
        - 当且仅当当前工单的代码实现已经完成、相关验证已经通过、已经形成至少一个覆盖本工单范围的 git commit、当前 ticket branch 已推送、并且关联 PR 已创建或更新到包含本次提交时，使用 platform skill 将当前工单状态更新到 `{{{{ workflow.finish_status }}}}`。
        - 对这个 workflow，不要把 `{{{{ workflow.finish_status }}}}` 理解成最终完成；它代表“ready for review”。PR merged 之后才应该进入最终 `Done`。
        - 如果你只验证了已有脏工作区、但没有形成新的提交和 PR 更新，不要推进状态到 `{{{{ workflow.finish_status }}}}`；应在 workpad 中明确说明仍未完成交付收口。
        - 不要在实现尚未完成时提前把当前 ticket 改到非 pickup 状态；一旦移出 pickup，当前 workflow 会结束这张工单的领取与执行。
        - 如果在执行中发现还需要额外的后续工作，可使用 platform skill 创建 follow-up ticket，但不要因为顺手拆分任务就提前结束当前 ticket。

        Todo App 质量要求：

        - 页面加载后应能直接在浏览器中使用，不依赖额外服务。
        - UI 可以保持简单，但不能明显粗糙；基本布局、层次、按钮状态和文本反馈要清楚。
        - 代码应易于继续扩展后续工单，例如在后续加入筛选、计数、持久化或回归测试时不需要推倒重来。
        - 测试应覆盖当前工单引入的关键行为或稳定契约，而不是只断言无关常量。

        建议执行顺序：

        1. 先创建或更新 `## Workpad`，记录环境戳、计划和初始判断。
        2. 再读取工单描述、README、当前源码和现有测试，确认当前切片的真实边界。
        3. 找到最小实现路径，再动手改代码。
        4. 实现后立即补齐或更新测试，并刷新 workpad。
        5. 运行相关验证命令，确认结果，并把结果写入 workpad。
        6. 最终只输出简洁的完成情况、变更点、验证命令与剩余风险。

        输出要求：

        - 不要写长篇空话。
        - 最终总结必须包含：改了什么、跑了什么验证、结果如何、还有什么剩余风险或未覆盖点。
        - 如果被阻塞，只报告真实阻塞原因，不要编造完成状态。
        """
    )


def main() -> int:
    args = build_parser().parse_args()
    stamp = time.strftime("%Y%m%d%H%M%S")
    base_url = args.base_url.rstrip("/")
    project_name = args.project_name.strip() or "Todo App"
    org_slug = f"{slugify(project_name)}-validation-{stamp}"
    project_slug = f"{slugify(project_name)}-{stamp}"
    workflow_name = f"{project_name} Coding Workflow"
    github_repo = FIXED_GITHUB_REPO
    github_project = None
    github_items = []
    github_repo_preparation = None
    github_pull_requests = []
    require_platform_skill = args.require_platform_skill or args.provider_mode == "real-codex"
    workspace_parent = Path(args.workspace_parent).resolve()
    if not workspace_parent.is_absolute():
        raise RuntimeError("--workspace-parent must be absolute")

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

    try:
        print(f"[1/12] health check against {base_url}")
        request_json(base_url, "GET", "/healthz")
        request_json(base_url, "GET", "/api/v1/healthz")

        print(f"[2/12] prepare fixed GitHub repository {FIXED_GITHUB_REPO_URL}")
        run_cli(["gh", "auth", "status"], check=True)
        github_repo_preparation = prepare_github_validation_repository(github_repo, workspace_parent, stamp)
        if not args.skip_github:
            print("[3/12] create a dedicated GitHub Project and linked issues")
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
            print("[3/12] skip GitHub Project and issue creation")

        print("[4/12] using the fixed GitHub repository and a local seed clone for the project repo")
        print(f"repository={FIXED_GITHUB_REPO_URL}")

        print("[5/12] create isolated OpenASE organization and project")
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
                "status": PROJECT_STATUS_IN_PROGRESS,
                "max_concurrent_agents": 1,
            },
        )["project"]

        print("[6/12] seed default statuses, verify local machine, then create provider/agent/workflow")
        statuses = request_json(base_url, "POST", f"/api/v1/projects/{project['id']}/statuses/reset")["statuses"]
        todo = require_by_name(statuses, "name", "Todo")
        in_review = require_by_name(statuses, "name", "In Review")
        local_machine = require_single_local_machine(base_url, org["id"])
        if args.provider_mode == "fake-codex":
            repo_root = Path(__file__).resolve().parents[2]
            fake_codex_path = repo_root / "scripts" / "dev" / "fake_codex_app_server.py"
            provider_payload = {
                "machine_id": local_machine["id"],
                "name": "Fake Codex Validation Provider",
                "adapter_type": "codex-app-server",
                "cli_command": os.environ.get("PYTHON", "python3"),
                "cli_args": [str(fake_codex_path)],
                "auth_config": {},
                "model_name": "gpt-5.4",
            }
        else:
            provider_payload = {
                "machine_id": local_machine["id"],
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
            },
        )["agent"]
        project_repo = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/repos",
            {
                "name": slugify(project_name),
                "repository_url": github_repo_preparation["ssh_url"],
                "default_branch": github_repo_preparation["default_branch"],
                "labels": ["todo-app", "validation", "github"],
            },
        )["repo"]
        workflow = request_json(
            base_url,
            "POST",
            f"/api/v1/projects/{project['id']}/workflows",
            {
                "agent_id": agent["id"],
                "name": workflow_name,
                "type": "coding",
                "pickup_status_ids": [todo["id"]],
                "finish_status_ids": [in_review["id"]],
                "harness_content": build_validation_workflow_harness(project_name),
            },
        )["workflow"]

        print("[7/12] set project defaults after creating the primary workspace repo")
        request_json(
            base_url,
            "PATCH",
            f"/api/v1/projects/{project['id']}",
            {
                "default_agent_provider_id": provider["id"],
            },
        )
        print("[8/12] create linked OpenASE tickets, repo scopes, and draft PRs")
        tickets = []
        ticket_workspace_paths: dict[str, str] = {}
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
            ticket_workspace_paths[ticket["id"]] = str(
                derive_ticket_workspace_repo_path(org_slug, project_slug, ticket["identifier"], project_repo["name"])
            )

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
            branch_name = build_ticket_branch_name(ticket["identifier"])
            pr_body = textwrap.dedent(
                f"""\
                Validation PR seeded by `scripts/dev/create_todo_app_realistic_workflow.py`.

                OpenASE project: {project_name}
                OpenASE ticket: {ticket["identifier"]} {ticket["title"]}
                Linked GitHub issue: {github_issue["url"] if github_issue else "none"}
                """
            )
            github_pr = create_github_pull_request(
                github_repo,
                github_repo_preparation["repo_dir"],
                branch_name,
                github_repo_preparation["baseline_head"],
                f"{ticket['identifier']}: {ticket['title']}",
                pr_body,
            )
            github_pull_requests.append(github_pr)
            upsert_ticket_repo_scope(
                base_url,
                project["id"],
                ticket["id"],
                project_repo["id"],
                github_pr["branch_name"],
                github_pr["url"],
                "open",
                "pending",
            )
            request_json(
                base_url,
                "POST",
                f"/api/v1/tickets/{ticket['id']}/external-links",
                {
                    "type": "github_pr",
                    "url": github_pr["url"],
                    "external_id": github_pr["external_id"],
                    "title": ticket["title"],
                    "status": "open",
                    "relation": "related",
                },
            )
            ticket = request_json(base_url, "GET", f"/api/v1/tickets/{ticket['id']}")["ticket"]
            tickets[-1] = ticket

        print("[9/12] add one realistic dependency edge")
        request_json(
            base_url,
            "POST",
            f"/api/v1/tickets/{tickets[0]['id']}/dependencies",
            {
                "target_ticket_id": tickets[1]["id"],
                "type": "blocks",
            },
        )

        print("[10/12] wait for the scheduler to claim work")
        agent_after_claim = wait_for_agent_claim(base_url, project["id"], agent["id"], args.wait_seconds)

        print("[11/12] wait for runtime execution, workspace activity, and platform skill completion")
        agent_after_execution = wait_for_agent_execution(base_url, project["id"], agent["id"], args.wait_seconds)
        workspace_activity = None
        platform_skill_result = None
        claimed_ticket_id = None
        claimed_ticket_workspace_path = None
        if isinstance(agent_after_claim, dict):
            runtime = agent_after_claim.get("runtime") if isinstance(agent_after_claim.get("runtime"), dict) else {}
            claimed_ticket_id = runtime.get("current_ticket_id") or agent_after_claim.get("current_ticket_id")
        if claimed_ticket_id:
            claimed_ticket_workspace_path = ticket_workspace_paths.get(claimed_ticket_id)
            if claimed_ticket_workspace_path:
                workspace_activity = wait_for_workspace_activity(
                    Path(claimed_ticket_workspace_path),
                    github_repo_preparation["baseline_head"],
                    args.wait_for_workspace_seconds,
                )
            if args.provider_mode == "fake-codex" and not args.require_platform_skill:
                platform_skill_result = {
                    "detected": False,
                    "required": False,
                    "skipped": True,
                    "reason": "fake-codex mode does not execute workspace commands or platform skills",
                    "ticket_id": claimed_ticket_id,
                }
            else:
                platform_skill_result = wait_for_ticket_workpad_comment_by_agent(
                    base_url,
                    project["id"],
                    claimed_ticket_id,
                    f"agent:{agent['name']}",
                    args.platform_skill_timeout_seconds,
                )
                if platform_skill_result is None:
                    platform_skill_result = {
                        "detected": False,
                        "required": require_platform_skill,
                        "skipped": False,
                        "reason": "ticket lookup did not return a usable payload",
                        "ticket_id": claimed_ticket_id,
                    }
                else:
                    platform_skill_result["required"] = require_platform_skill
                    platform_skill_result["skipped"] = False
                    platform_skill_result["ticket_id"] = claimed_ticket_id
                    if not platform_skill_result["detected"]:
                        ticket = platform_skill_result.get("ticket", {})
                        platform_skill_result["reason"] = (
                            "claimed ticket did not receive an agent-owned Workpad comment update"
                        )
                        platform_skill_result["observed_status_id"] = ticket.get("status_id")
                        platform_skill_result["observed_status_name"] = ticket.get("status_name")
                        platform_skill_result["observed_comments"] = len(platform_skill_result.get("comments", []))
        else:
            platform_skill_result = {
                "detected": False,
                "required": require_platform_skill,
                "skipped": False,
                "reason": "agent never exposed a claimed current_ticket_id",
                "ticket_id": None,
            }

        if require_platform_skill and not platform_skill_result.get("detected", False):
            raise RuntimeError(
                "expected the agent to post or update a Workpad comment via openase-platform skill, "
                f"but no qualifying ticket comment activity was observed: {json.dumps(platform_skill_result, ensure_ascii=False)}"
            )

        print("[12/12] summarize created resources")
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
                "agent_after_execution": agent_after_execution,
                "tickets": tickets,
                "platform_skill": platform_skill_result,
            },
            "github": {
                "repo_prepared": True,
                "owner": args.github_owner,
                "repo": github_repo,
                "repo_url": FIXED_GITHUB_REPO_URL,
                "baseline_head": github_repo_preparation["baseline_head"],
                "validation_branch_prefix": "agent",
                "seed_repo_path": str(github_repo_preparation["repo_dir"]),
                "project": github_project,
                "issues": github_items,
                "pull_requests": github_pull_requests,
            },
            "workspace": {
                "root": str(Path.home() / ".openase" / "workspace"),
                "claimed_ticket_repo_path": claimed_ticket_workspace_path,
                "ticket_repo_paths": ticket_workspace_paths,
                "baseline_head": github_repo_preparation["baseline_head"],
                "activity": workspace_activity,
            },
            "notes": [
                "OpenASE project-facing connector CRUD is not exported yet.",
                "Project repo workspaces are now materialized directly from repository_url and workspace_dirname; the legacy mirror registration API is no longer used.",
                "The OpenASE project repo points at BetterAndBetterII/TodoApp, while the local seed clone is used to prepare baseline commits plus linked GitHub issues and PRs.",
                "The script creates a fresh empty baseline checkpoint on main before opening ticket PRs.",
                "Each ticket gets a GitHub draft PR, a repo scope branch binding, and a github_pr external link.",
                f"Provider mode: {args.provider_mode}",
                "Platform skill detection is based on the claimed ticket receiving a Workpad comment created or edited by agent:<agent-name>.",
            ],
        }
        print(json.dumps(summary, indent=2))
        return 0
    finally:
        pass

if __name__ == "__main__":
    raise SystemExit(main())
