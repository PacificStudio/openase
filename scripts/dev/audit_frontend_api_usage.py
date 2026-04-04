#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path


HTTP_METHODS = ("get", "post", "patch", "delete")
RUNTIME_SUFFIXES = (".ts", ".js", ".svelte")
TEST_SUFFIXES = (".test.ts", ".test.js", ".spec.ts", ".spec.js")
REPORT_CATEGORIES = ("backend_only", "contract_only", "wrapped_but_unused", "direct_only", "used")
DEFAULT_IGNORE_FILE = Path(__file__).resolve().with_name("frontend_api_audit_ignores.json")


@dataclass(frozen=True)
class OperationKey:
    method: str
    path: str


@dataclass(frozen=True)
class WrapperUsage:
    name: str
    file: str


@dataclass(frozen=True)
class IgnoreRule:
    categories: frozenset[str]
    reason: str


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Audit OpenASE frontend API coverage by comparing OpenAPI paths against "
            "contracts.ts, openase.ts wrappers, and runtime call sites."
        )
    )
    parser.add_argument(
        "--show-used",
        action="store_true",
        help="Include endpoints that already have runtime usage.",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Emit machine-readable JSON instead of a text report.",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=0,
        help="Limit the number of reported operations per category. Zero means no limit.",
    )
    parser.add_argument(
        "--fail-on",
        action="append",
        choices=REPORT_CATEGORIES,
        default=[],
        help="Exit non-zero when the report contains any operations in the selected category.",
    )
    parser.add_argument(
        "--ignore-file",
        help=(
            "Optional JSON file listing intentionally ignored operations. "
            f"Defaults to {DEFAULT_IGNORE_FILE.name} next to this script when present."
        ),
    )
    return parser


def load_openapi_operations(path: Path) -> dict[OperationKey, dict[str, str]]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    operations: dict[OperationKey, dict[str, str]] = {}
    for raw_path, methods in payload.get("paths", {}).items():
        for method, operation in methods.items():
            lowered = method.lower()
            if lowered not in HTTP_METHODS:
                continue
            operations[OperationKey(method=lowered, path=raw_path)] = {
                "operation_id": operation.get("operationId", ""),
                "summary": operation.get("summary", ""),
            }
    return operations


def normalize_template_path(raw: str) -> str:
    normalized = raw.strip()
    normalized = normalized.split("?", 1)[0]
    normalized = re.sub(
        r"\$\{\s*(?:encodeURIComponent|encodeURI)\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\}",
        r"{\1}",
        normalized,
    )
    normalized = re.sub(r"\$\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}", r"{\1}", normalized)
    normalized = re.sub(r"\$\{[^}]+\}", "{expr}", normalized)
    return normalized


def load_ignore_rules(path: Path) -> dict[OperationKey, IgnoreRule]:
    if not path.exists():
        return {}

    payload = json.loads(path.read_text(encoding="utf-8"))
    operations = payload.get("operations")
    if not isinstance(operations, list):
        raise ValueError(f"{path} must contain an 'operations' array")

    rules: dict[OperationKey, IgnoreRule] = {}
    for index, item in enumerate(operations):
        if not isinstance(item, dict):
            raise ValueError(f"{path} operations[{index}] must be an object")

        method = str(item.get("method", "")).lower()
        if method not in HTTP_METHODS:
            raise ValueError(f"{path} operations[{index}] has invalid method {method!r}")

        raw_path = str(item.get("path", "")).strip()
        if not raw_path.startswith("/"):
            raise ValueError(f"{path} operations[{index}] has invalid path {raw_path!r}")

        raw_categories = item.get("categories", REPORT_CATEGORIES)
        if not isinstance(raw_categories, list) or not raw_categories:
            raise ValueError(f"{path} operations[{index}] categories must be a non-empty array")

        categories = frozenset(str(category) for category in raw_categories)
        invalid_categories = sorted(category for category in categories if category not in REPORT_CATEGORIES)
        if invalid_categories:
            raise ValueError(
                f"{path} operations[{index}] has invalid categories: {', '.join(invalid_categories)}"
            )

        reason = str(item.get("reason", "")).strip()
        if not reason:
            raise ValueError(f"{path} operations[{index}] must include a non-empty reason")

        key = OperationKey(method=method, path=raw_path)
        if key in rules:
            raise ValueError(f"{path} declares duplicate ignore rule for {method.upper()} {raw_path}")

        rules[key] = IgnoreRule(categories=categories, reason=reason)
    return rules


def extract_response_refs(contracts_source: str) -> set[OperationKey]:
    refs: set[OperationKey] = set()
    pattern = re.compile(r"ResponseFor<'([^']+)',\s*'([^']+)'>")
    for path, method in pattern.findall(contracts_source):
        lowered = method.lower()
        if lowered in HTTP_METHODS:
            refs.add(OperationKey(method=lowered, path=path))
    return refs


def parse_exported_function_blocks(source: str) -> list[tuple[str, str]]:
    functions: list[tuple[str, str]] = []
    markers = ("export async function ", "export function ")
    offset = 0
    while True:
        start = -1
        marker = ""
        for candidate in markers:
            candidate_start = source.find(candidate, offset)
            if candidate_start >= 0 and (start < 0 or candidate_start < start):
                start = candidate_start
                marker = candidate
        if start < 0:
            return functions

        name_start = start + len(marker)
        name_end = name_start
        while name_end < len(source) and (source[name_end].isalnum() or source[name_end] == "_"):
            name_end += 1
        name = source[name_start:name_end]

        params_start = source.find("(", name_end)
        if params_start < 0:
            return functions

        params_depth = 0
        cursor = params_start
        while cursor < len(source):
            char = source[cursor]
            if char == "(":
                params_depth += 1
            elif char == ")":
                params_depth -= 1
                if params_depth == 0:
                    break
            cursor += 1
        else:
            return functions

        body_start = source.find("{", cursor + 1)
        if body_start < 0:
            return functions

        body_depth = 0
        cursor = body_start
        while cursor < len(source):
            char = source[cursor]
            if char == "{":
                body_depth += 1
            elif char == "}":
                body_depth -= 1
                if body_depth == 0:
                    functions.append((name, source[start : cursor + 1]))
                    offset = cursor + 1
                    break
            cursor += 1
        else:
            return functions


def extract_named_call_blocks(source: str, callee_name: str) -> list[str]:
    calls: list[str] = []
    offset = 0
    while True:
        start = source.find(callee_name, offset)
        if start < 0:
            return calls
        after_name = start + len(callee_name)
        if after_name < len(source) and (source[after_name].isalnum() or source[after_name] == "_"):
            offset = after_name
            continue

        params_start = source.find("(", start + len(callee_name))
        if params_start < 0:
            return calls

        depth = 0
        cursor = params_start
        while cursor < len(source):
            char = source[cursor]
            if char == "(":
                depth += 1
            elif char == ")":
                depth -= 1
                if depth == 0:
                    calls.append(source[start : cursor + 1])
                    offset = cursor + 1
                    break
            cursor += 1
        else:
            return calls


def extract_wrappers(openase_source: str, file_label: str) -> dict[OperationKey, list[WrapperUsage]]:
    wrappers: dict[OperationKey, list[WrapperUsage]] = defaultdict(list)
    call_pattern = re.compile(
        r"api\.(get|post|patch|delete)(?:\s*<[^>]+>)?\(\s*([`'\"])(.*?)\2",
        flags=re.DOTALL,
    )
    quoted_api_path_pattern = re.compile(r"([`'\"])(/api/v1.*?)\1", flags=re.DOTALL)
    fetch_method_pattern = re.compile(r"method\s*:\s*['\"](GET|POST|PATCH|DELETE)['\"]", flags=re.IGNORECASE)
    for function_name, block in parse_exported_function_blocks(openase_source):
        for method, _, raw_path in call_pattern.findall(block):
            path = normalize_template_path(raw_path)
            wrappers[OperationKey(method=method.lower(), path=path)].append(
                WrapperUsage(name=function_name, file=file_label)
            )
        for callee_name in ("fetch", "fetchJSON"):
            for call in extract_named_call_blocks(block, callee_name):
                path_match = quoted_api_path_pattern.search(call)
                if not path_match:
                    continue
                method_match = fetch_method_pattern.search(call)
                method = method_match.group(1).lower() if method_match else "get"
                path = normalize_template_path(path_match.group(2))
                wrappers[OperationKey(method=method, path=path)].append(
                    WrapperUsage(name=function_name, file=file_label)
                )
    return wrappers


def iter_api_wrapper_files(api_root: Path) -> list[Path]:
    files: list[Path] = []
    for path in sorted(api_root.glob("*.ts")):
        if path.name in {"client.ts", "contracts.ts"}:
            continue
        if path.name.endswith(TEST_SUFFIXES):
            continue
        files.append(path)
    return files


def iter_runtime_source_files(root: Path) -> list[Path]:
    files: list[Path] = []
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if path.suffix not in {".ts", ".js", ".svelte"}:
            continue
        if path.name.endswith(TEST_SUFFIXES):
            continue
        if "generated" in path.parts:
            continue
        files.append(path)
    return sorted(files)


def extract_direct_endpoint_usages(root: Path, exclude: set[Path]) -> dict[OperationKey, list[str]]:
    usages: dict[OperationKey, list[str]] = defaultdict(list)
    quoted_path_pattern = re.compile(r"([`'\"])(/api/v1[^`'\"]+)\1")
    method_hint_pattern = re.compile(r"\b(GET|POST|PATCH|DELETE)\b")

    for path in iter_runtime_source_files(root):
        if path in exclude:
            continue
        source = path.read_text(encoding="utf-8")
        relative = path.relative_to(root.parent).as_posix()
        method_hints = {match.lower() for match in method_hint_pattern.findall(source)}
        for _, raw_path in quoted_path_pattern.findall(source):
            normalized_path = normalize_template_path(raw_path)
            candidate_methods = method_hints or {"get"}
            for method in candidate_methods:
                usages[OperationKey(method=method, path=normalized_path)].append(relative)
    return usages


def extract_wrapper_references(
    root: Path,
    wrapper_names: set[str],
    exclude: set[Path],
) -> dict[str, list[str]]:
    references: dict[str, list[str]] = defaultdict(list)
    if not wrapper_names:
        return references

    patterns = {
        name: re.compile(rf"\b{name}\s*\(")
        for name in wrapper_names
    }
    for path in iter_runtime_source_files(root):
        if path in exclude:
            continue
        source = path.read_text(encoding="utf-8")
        relative = path.relative_to(root.parent).as_posix()
        for name, pattern in patterns.items():
            if pattern.search(source):
                references[name].append(relative)
    return references


def categorize_operation(
    operation: OperationKey,
    has_contract: bool,
    wrappers: list[WrapperUsage],
    wrapper_refs: dict[str, list[str]],
    direct_refs: list[str],
) -> tuple[str, dict[str, object]]:
    wrapper_names = [wrapper.name for wrapper in wrappers]
    used_wrappers = {name: refs for name, refs in wrapper_refs.items() if name in wrapper_names and refs}
    runtime_files = sorted(set(direct_refs + [ref for refs in used_wrappers.values() for ref in refs]))

    details = {
        "contract": has_contract,
        "wrappers": wrapper_names,
        "wrapper_runtime_refs": used_wrappers,
        "direct_runtime_refs": sorted(set(direct_refs)),
        "runtime_files": runtime_files,
    }

    if runtime_files:
        if wrappers:
            return "used", details
        return "direct_only", details
    if wrappers:
        return "wrapped_but_unused", details
    if has_contract:
        return "contract_only", details
    return "backend_only", details


def render_text_report(report: dict[str, list[dict[str, object]]], limit: int) -> str:
    lines: list[str] = []
    for category in REPORT_CATEGORIES:
        items = report.get(category, [])
        if not items:
            continue
        shown = items[:limit] if limit > 0 else items
        lines.append(f"[{category}] {len(items)}")
        for item in shown:
            lines.append(f"- {item['method'].upper()} {item['path']}")
            if item["operation_id"]:
                lines.append(f"  operation_id: {item['operation_id']}")
            if item["summary"]:
                lines.append(f"  summary: {item['summary']}")
            lines.append(
                f"  contract={item['contract']} wrappers={item['wrappers']} runtime_files={item['runtime_files']}"
            )
        if limit > 0 and len(items) > limit:
            lines.append(f"  ... {len(items) - limit} more")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    args = build_parser().parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    web_root = repo_root / "web" / "src"
    openapi_path = repo_root / "api" / "openapi.json"
    contracts_path = web_root / "lib" / "api" / "contracts.ts"
    api_root = web_root / "lib" / "api"
    ignore_path = (
        Path(args.ignore_file).expanduser()
        if args.ignore_file
        else DEFAULT_IGNORE_FILE
    )

    operations = load_openapi_operations(openapi_path)
    contract_refs = extract_response_refs(contracts_path.read_text(encoding="utf-8"))
    wrapper_files = iter_api_wrapper_files(api_root)
    wrapper_map: dict[OperationKey, list[WrapperUsage]] = defaultdict(list)
    for wrapper_file in wrapper_files:
        for operation, wrappers in extract_wrappers(
            wrapper_file.read_text(encoding="utf-8"),
            wrapper_file.relative_to(web_root.parent).as_posix(),
        ).items():
            wrapper_map[operation].extend(wrappers)
    ignore_rules = load_ignore_rules(ignore_path)
    wrapper_names = {wrapper.name for wrappers in wrapper_map.values() for wrapper in wrappers}
    direct_refs = extract_direct_endpoint_usages(
        web_root,
        exclude={contracts_path, *wrapper_files},
    )
    wrapper_refs = extract_wrapper_references(web_root, wrapper_names, exclude=set(wrapper_files))

    report: dict[str, list[dict[str, object]]] = defaultdict(list)
    for operation, metadata in sorted(operations.items(), key=lambda item: (item[0].path, item[0].method)):
        category, details = categorize_operation(
            operation=operation,
            has_contract=operation in contract_refs,
            wrappers=wrapper_map.get(operation, []),
            wrapper_refs=wrapper_refs,
            direct_refs=direct_refs.get(operation, []),
        )
        ignore_rule = ignore_rules.get(operation)
        if ignore_rule and category in ignore_rule.categories:
            continue
        if category == "used" and not args.show_used:
            continue
        report[category].append(
            {
                "method": operation.method,
                "path": operation.path,
                "operation_id": metadata["operation_id"],
                "summary": metadata["summary"],
                **details,
            }
        )

    if args.json:
        json.dump(report, sys.stdout, indent=2, sort_keys=True)
        sys.stdout.write("\n")
    else:
        sys.stdout.write(render_text_report(report, args.limit))

    failing_categories = [category for category in args.fail_on if report.get(category)]
    if failing_categories:
        summary = ", ".join(f"{category}={len(report[category])}" for category in failing_categories)
        sys.stderr.write(f"frontend API audit failed: {summary}\n")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
