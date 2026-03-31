#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable


CHAPTER_RE = re.compile(r"^##\s+(第[一二三四五六七八九十百零〇0-9]+章\s+.+)$")


@dataclass(frozen=True)
class Chapter:
    number: int
    title: str
    line_start: int
    line_end: int

    @property
    def file_name(self) -> str:
        return f"ch{self.number:02d}.txt"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run parallel Codex PRD-vs-code chapter alignment scans."
    )
    parser.add_argument(
        "--prd",
        default="OpenASE-PRD.md",
        help="Path to the PRD markdown file. Default: OpenASE-PRD.md",
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="Repo root used as Codex working directory. Default: current directory",
    )
    parser.add_argument(
        "--list-chapters",
        action="store_true",
        help="List top-level PRD chapters and exit",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Scan all top-level chapters",
    )
    parser.add_argument(
        "--chapters",
        default="",
        help="Comma-separated chapter numbers or ranges, for example 5,6,7-10",
    )
    parser.add_argument(
        "--match",
        default="",
        help="Case-sensitive regex applied to top-level chapter titles",
    )
    parser.add_argument(
        "--model",
        default="gpt-5.3-codex-spark",
        help="Codex model name. Default: gpt-5.3-codex-spark",
    )
    parser.add_argument(
        "--max-parallel",
        type=int,
        default=6,
        help="Maximum concurrent codex exec processes. Default: 6",
    )
    parser.add_argument(
        "--output-dir",
        default="",
        help="Output directory for chapter files. Default: /tmp/prd-alignment-scan-<timestamp>",
    )
    parser.add_argument(
        "--skip-synthesize",
        action="store_true",
        help="Skip the final synthesis Codex run",
    )
    parser.add_argument(
        "--extra-focus",
        default="",
        help="Optional extra focus text appended to each chapter prompt",
    )
    return parser.parse_args()


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        print(f"PRD file not found: {path}", file=sys.stderr)
        sys.exit(1)


def extract_chapters(prd_text: str) -> list[Chapter]:
    lines = prd_text.splitlines()
    starts: list[tuple[int, str]] = []
    for lineno, line in enumerate(lines, start=1):
        match = CHAPTER_RE.match(line)
        if match:
            starts.append((lineno, match.group(1)))

    chapters: list[Chapter] = []
    for index, (line_start, title) in enumerate(starts, start=1):
        line_end = starts[index][0] - 1 if index < len(starts) else len(lines)
        chapters.append(
            Chapter(
                number=index,
                title=title,
                line_start=line_start,
                line_end=line_end,
            )
        )
    return chapters


def parse_chapter_selector(raw: str, max_number: int) -> set[int]:
    selected: set[int] = set()
    if not raw.strip():
        return selected
    for token in raw.split(","):
        token = token.strip()
        if not token:
            continue
        if "-" in token:
            start_raw, end_raw = token.split("-", 1)
            start = int(start_raw)
            end = int(end_raw)
            if start > end:
                start, end = end, start
            for number in range(start, end + 1):
                if 1 <= number <= max_number:
                    selected.add(number)
        else:
            number = int(token)
            if 1 <= number <= max_number:
                selected.add(number)
    return selected


def select_chapters(
    chapters: list[Chapter], scan_all: bool, chapter_selector: str, title_regex: str
) -> list[Chapter]:
    if scan_all:
        return chapters

    selected_numbers = parse_chapter_selector(chapter_selector, len(chapters))
    regex = re.compile(title_regex) if title_regex.strip() else None

    result: list[Chapter] = []
    for chapter in chapters:
        if selected_numbers and chapter.number in selected_numbers:
            result.append(chapter)
            continue
        if regex and regex.search(chapter.title):
            result.append(chapter)

    deduped: dict[int, Chapter] = {chapter.number: chapter for chapter in result}
    return [deduped[number] for number in sorted(deduped)]


def build_chapter_prompt(prd_path: str, chapter: Chapter, extra_focus: str) -> str:
    prompt = (
        f"请只分析 {prd_path} 的{chapter.title}（行 {chapter.line_start}-{chapter.line_end}），"
        "再扫描当前仓库代码实现，判断该章节与代码是否存在偏差。\n"
        "请输出：\n"
        "1. 本章关键承诺（最多 8 条）\n"
        "2. 已实现且基本符合的点\n"
        "3. 明确存在的偏差（每条都要写 PRD 期望、当前实现、相关文件路径）\n"
        "4. 不确定项\n"
        "要求：\n"
        "- 只基于当前仓库\n"
        "- 使用中文，简洁\n"
        "- 如果该章节主要是愿景、路线图、视觉风格或产品主张，无法直接从代码验证时要明确说明，不要臆测\n"
        "- 尽量引用当前仓库里的具体文件路径\n"
    )
    if extra_focus.strip():
        prompt += f"- 额外关注点：{extra_focus.strip()}\n"
    return prompt


def build_summary_prompt(output_dir: Path, manifest_path: Path) -> str:
    return (
        f"请阅读 {manifest_path} 和同目录下所有 chNN.txt 章节扫描结果，"
        "整理一份去重后的 PRD vs 代码实现偏差清单。\n"
        "输出要求：\n"
        "1. 先给出扫描范围与覆盖章节\n"
        "2. 然后分为：高优先级偏差 / 中优先级偏差 / 低优先级或产品体验偏差 / 不确定项\n"
        "3. 每条偏差要尽量去重，按“PRD 期望 / 当前实现 / 相关文件”表达\n"
        "4. 单独列出“已经基本对齐的章节”\n"
        "5. 最后给出建议修复顺序\n"
        "要求：\n"
        "- 只基于当前目录中的章节结果文件\n"
        "- 使用中文\n"
        "- 优先总结核心语义偏差，而不是 UI 文案或风格差异\n"
        f"- 输出写入 {output_dir / 'summary.md'}\n"
    )


def run_command(command: list[str], cwd: Path) -> subprocess.Popen[str]:
    return subprocess.Popen(
        command,
        cwd=str(cwd),
        stdout=subprocess.DEVNULL,
        stderr=subprocess.STDOUT,
    )


def launch_chapter_scans(
    repo_root: Path,
    prd_path: str,
    selected: list[Chapter],
    output_dir: Path,
    model: str,
    max_parallel: int,
    extra_focus: str,
) -> None:
    queue = list(selected)
    running: list[tuple[Chapter, subprocess.Popen[str]]] = []
    completed = 0

    while queue or running:
        while queue and len(running) < max_parallel:
            chapter = queue.pop(0)
            output_path = output_dir / chapter.file_name
            prompt = build_chapter_prompt(prd_path, chapter, extra_focus)
            command = [
                "codex",
                "exec",
                "-m",
                model,
                "--dangerously-bypass-approvals-and-sandbox",
                "-C",
                str(repo_root),
                "-o",
                str(output_path),
                prompt,
            ]
            print(
                f"[scan] launch chapter {chapter.number:02d}: {chapter.title}",
                file=sys.stderr,
            )
            running.append((chapter, run_command(command, repo_root)))

        still_running: list[tuple[Chapter, subprocess.Popen[str]]] = []
        for chapter, process in running:
            if process.poll() is None:
                still_running.append((chapter, process))
                continue
            if process.returncode != 0:
                raise RuntimeError(
                    f"chapter {chapter.number:02d} scan failed with exit code {process.returncode}"
                )
            completed += 1
            print(
                f"[scan] completed {completed}/{len(selected)} chapters",
                file=sys.stderr,
            )
        running = still_running
        if queue or running:
            time.sleep(2)


def write_manifest(output_dir: Path, prd_path: Path, selected: Iterable[Chapter]) -> Path:
    manifest_path = output_dir / "manifest.json"
    payload = {
        "prd": str(prd_path),
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "chapters": [
            {
                **asdict(chapter),
                "output_file": chapter.file_name,
            }
            for chapter in selected
        ],
    }
    manifest_path.write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    return manifest_path


def run_summary(repo_root: Path, output_dir: Path, manifest_path: Path, model: str) -> None:
    summary_path = output_dir / "summary.md"
    prompt = build_summary_prompt(output_dir, manifest_path)
    command = [
        "codex",
        "exec",
        "-m",
        model,
        "--dangerously-bypass-approvals-and-sandbox",
        "-C",
        str(repo_root),
        "-o",
        str(summary_path),
        prompt,
    ]
    print("[summary] launch synthesis", file=sys.stderr)
    completed = subprocess.run(command, cwd=str(repo_root), check=False)
    if completed.returncode != 0:
        raise RuntimeError(f"summary synthesis failed with exit code {completed.returncode}")


def default_output_dir() -> Path:
    return Path("/tmp") / f"prd-alignment-scan-{time.strftime('%Y%m%d-%H%M%S')}"


def main() -> int:
    args = parse_args()
    repo_root = Path(args.repo_root).resolve()
    prd_path = Path(args.prd)
    prd_abs = (repo_root / prd_path).resolve() if not prd_path.is_absolute() else prd_path
    chapters = extract_chapters(read_text(prd_abs))

    if args.list_chapters:
        for chapter in chapters:
            print(
                f"{chapter.number:02d}\t{chapter.line_start}-{chapter.line_end}\t{chapter.title}"
            )
        return 0

    selected = select_chapters(chapters, args.all, args.chapters, args.match)
    if not selected:
        print(
            "No chapters selected. Use --all, --chapters, or --match.",
            file=sys.stderr,
        )
        return 2

    output_dir = Path(args.output_dir).resolve() if args.output_dir else default_output_dir()
    output_dir.mkdir(parents=True, exist_ok=True)

    manifest_path = write_manifest(output_dir, prd_abs, selected)
    print(f"[scan] output dir: {output_dir}", file=sys.stderr)
    print(f"[scan] manifest: {manifest_path}", file=sys.stderr)

    launch_chapter_scans(
        repo_root=repo_root,
        prd_path=prd_path.as_posix(),
        selected=selected,
        output_dir=output_dir,
        model=args.model,
        max_parallel=max(1, args.max_parallel),
        extra_focus=args.extra_focus,
    )

    if not args.skip_synthesize:
        run_summary(repo_root=repo_root, output_dir=output_dir, manifest_path=manifest_path, model=args.model)

    print(json.dumps({"output_dir": str(output_dir), "manifest": str(manifest_path)}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
