#!/usr/bin/env python3
"""Repository architecture guard for OpenASE.

This script enforces dependency direction using package-prefix checks and an
explicit temporary-debt allowlist. The allowlist is intentionally narrow:
each exception is tied to one file and one forbidden import prefix so CI
blocks new drift while existing debt is paid down incrementally.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import re
import sys


REPO_ROOT = Path(__file__).resolve().parents[2]
IMPORT_RE = re.compile(r'"([^"]+)"')
GO_IMPORT_ROOT = "github.com/BetterAndBetterII/openase/"


def go_import_prefix(path_prefix: str) -> str:
    return GO_IMPORT_ROOT + path_prefix


@dataclass(frozen=True)
class ExceptionEntry:
    path: str
    import_prefix: str
    rationale: str


@dataclass(frozen=True)
class Rule:
    name: str
    file_prefixes: tuple[str, ...]
    forbidden_import_prefixes: tuple[str, ...]
    exceptions: tuple[ExceptionEntry, ...] = ()

    def applies_to(self, relative_path: str) -> bool:
        return any(relative_path.startswith(prefix) for prefix in self.file_prefixes)


DOMAIN_OR_TYPES = ("internal/domain/", "internal/types/")
SERVICE_USECASE = (
    "internal/service/",
    "internal/ticket/",
    "internal/ticketstatus/",
    "internal/workflow/",
    "internal/chat/",
    "internal/notification/",
    "internal/issueconnector/",
    "internal/scheduledjob/",
    "internal/agentplatform/",
)
INTERFACE_ENTRY = ("internal/httpapi/", "internal/cli/", "cmd/openase/")
UPPER_DELIVERY = (
    go_import_prefix("internal/httpapi"),
    go_import_prefix("internal/app"),
    go_import_prefix("internal/runtime"),
    go_import_prefix("internal/setup"),
    go_import_prefix("internal/webui"),
)
SERVICE_LATERAL = tuple(go_import_prefix(prefix.rstrip("/")) for prefix in SERVICE_USECASE if prefix != "internal/service/")


RULES = (
    Rule(
        name="domain-core-no-ent",
        file_prefixes=DOMAIN_OR_TYPES,
        forbidden_import_prefixes=(go_import_prefix("ent"),),
    ),
    Rule(
        name="domain-core-no-upper-layers",
        file_prefixes=DOMAIN_OR_TYPES,
        forbidden_import_prefixes=(
            go_import_prefix("internal/repo"),
            go_import_prefix("internal/service"),
            go_import_prefix("internal/ticket"),
            go_import_prefix("internal/ticketstatus"),
            go_import_prefix("internal/workflow"),
            go_import_prefix("internal/chat"),
            go_import_prefix("internal/notification"),
            go_import_prefix("internal/issueconnector"),
            go_import_prefix("internal/scheduledjob"),
            go_import_prefix("internal/agentplatform"),
            go_import_prefix("internal/httpapi"),
            go_import_prefix("internal/app"),
            go_import_prefix("internal/runtime"),
            go_import_prefix("internal/orchestrator"),
            go_import_prefix("internal/setup"),
            go_import_prefix("internal/webui"),
            go_import_prefix("internal/infra"),
        ),
    ),
    Rule(
        name="provider-no-upper-layers",
        file_prefixes=("internal/provider/",),
        forbidden_import_prefixes=(
            go_import_prefix("internal/repo"),
            go_import_prefix("internal/service"),
            go_import_prefix("internal/ticket"),
            go_import_prefix("internal/ticketstatus"),
            go_import_prefix("internal/workflow"),
            go_import_prefix("internal/chat"),
            go_import_prefix("internal/notification"),
            go_import_prefix("internal/issueconnector"),
            go_import_prefix("internal/scheduledjob"),
            go_import_prefix("internal/agentplatform"),
            go_import_prefix("internal/httpapi"),
            go_import_prefix("internal/app"),
            go_import_prefix("internal/runtime"),
            go_import_prefix("internal/orchestrator"),
            go_import_prefix("internal/setup"),
            go_import_prefix("internal/webui"),
            go_import_prefix("internal/infra"),
        ),
    ),
    Rule(
        name="repo-no-upper-layers",
        file_prefixes=("internal/repo/",),
        forbidden_import_prefixes=(
            go_import_prefix("internal/service"),
            go_import_prefix("internal/ticket"),
            go_import_prefix("internal/ticketstatus"),
            go_import_prefix("internal/workflow"),
            go_import_prefix("internal/chat"),
            go_import_prefix("internal/notification"),
            go_import_prefix("internal/issueconnector"),
            go_import_prefix("internal/scheduledjob"),
            go_import_prefix("internal/agentplatform"),
            go_import_prefix("internal/httpapi"),
            go_import_prefix("internal/app"),
            go_import_prefix("internal/runtime"),
            go_import_prefix("internal/orchestrator"),
            go_import_prefix("internal/setup"),
            go_import_prefix("internal/webui"),
        ),
    ),
    Rule(
        name="service-usecase-no-ent",
        file_prefixes=SERVICE_USECASE,
        forbidden_import_prefixes=(go_import_prefix("ent"),),
        exceptions=(),
    ),
    Rule(
        name="service-usecase-no-delivery-or-wiring",
        file_prefixes=SERVICE_USECASE,
        forbidden_import_prefixes=UPPER_DELIVERY,
    ),
    Rule(
        name="internal-service-no-lateral-usecase",
        file_prefixes=("internal/service/",),
        forbidden_import_prefixes=SERVICE_LATERAL,
    ),
    Rule(
        name="interface-entry-no-ent",
        file_prefixes=INTERFACE_ENTRY,
        forbidden_import_prefixes=(go_import_prefix("ent"),),
        exceptions=(
        ),
    ),
)


def iter_go_files() -> list[Path]:
    files: list[Path] = []
    for prefix in {
        *DOMAIN_OR_TYPES,
        *SERVICE_USECASE,
        *INTERFACE_ENTRY,
        "internal/provider/",
        "internal/repo/",
        "internal/infra/",
    }:
        root = REPO_ROOT / prefix
        if not root.exists():
            continue
        files.extend(path for path in root.rglob("*.go") if not path.name.endswith("_test.go"))
    return sorted(set(files))


def parse_imports(path: Path) -> list[str]:
    return IMPORT_RE.findall(path.read_text())


def main() -> int:
    violations: list[str] = []
    used_exceptions: set[tuple[str, str]] = set()
    all_exceptions: dict[tuple[str, str], ExceptionEntry] = {}

    for rule in RULES:
        for entry in rule.exceptions:
            key = (entry.path, entry.import_prefix)
            if key in all_exceptions:
                raise SystemExit(f"duplicate exception entry: {entry.path} / {entry.import_prefix}")
            all_exceptions[key] = entry

    for path in iter_go_files():
        relative_path = path.relative_to(REPO_ROOT).as_posix()
        source = path.read_text()
        imports = parse_imports(path)
        for rule in RULES:
            if not rule.applies_to(relative_path):
                continue
            sorted_prefixes = sorted(
                rule.forbidden_import_prefixes,
                key=len,
                reverse=True,
            )
            for imported in imports:
                for forbidden_prefix in sorted_prefixes:
                    if not imported.startswith(forbidden_prefix):
                        continue
                    key = (relative_path, forbidden_prefix)
                    if key in all_exceptions:
                        used_exceptions.add(key)
                        break
                    violations.append(
                        f"{relative_path}: rule {rule.name} forbids import {imported}"
                    )
                    break
        if relative_path.startswith("internal/infra/"):
            logging_import = go_import_prefix("internal/logging")
            if logging_import not in imports:
                violations.append(
                    f"{relative_path}: infra files must import {logging_import}"
                )
            if "logging." not in source:
                violations.append(
                    f"{relative_path}: infra files must use the logging package at least once"
                )
    stale_exceptions = [
        all_exceptions[key]
        for key in sorted(all_exceptions)
        if key not in used_exceptions
    ]

    if violations or stale_exceptions:
        if violations:
            print("Architecture violations:", file=sys.stderr)
            for item in violations:
                print(f"  - {item}", file=sys.stderr)
        if stale_exceptions:
            print("Stale architecture exceptions:", file=sys.stderr)
            for entry in stale_exceptions:
                print(
                    f"  - {entry.path} no longer needs exception for {entry.import_prefix}"
                    f" ({entry.rationale})",
                    file=sys.stderr,
                )
        return 1

    print("Architecture guard passed.")
    if used_exceptions:
        print("Temporary debt exceptions still active:")
        for key in sorted(used_exceptions):
            entry = all_exceptions[key]
            print(f"  - {entry.path} -> {entry.import_prefix} ({entry.rationale})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
