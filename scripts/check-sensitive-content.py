#!/usr/bin/env python3
"""Reject tracked text that matches known credentials or private user context."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

PATTERNS = {
    "GitHub token": re.compile(
        r"\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b"
    ),
    "OpenAI-style API key": re.compile(r"\bsk-[A-Za-z0-9_-]{20,}\b"),
    "Slack token": re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{20,}\b"),
    "private network URL": re.compile(
        r"https?://(?:localhost|127(?:\.\d{1,3}){3}|10(?:\.\d{1,3}){3}|"
        r"192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?::\d+)?(?:[/?#]|\b)",
        re.IGNORECASE,
    ),
    "macOS user path": re.compile(r"/Users/[A-Za-z0-9._-]+(?:/|\b)"),
    "email address": re.compile(
        r"\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b", re.IGNORECASE
    ),
}


def candidate_files(root: Path) -> list[Path]:
    result = subprocess.run(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"],
        cwd=root,
        check=False,
        capture_output=True,
    )
    if result.returncode == 0:
        return [root / item.decode() for item in result.stdout.split(b"\0") if item]
    return [
        path for path in root.rglob("*") if path.is_file() and ".git" not in path.parts
    ]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parent.parent
    )
    return main_with_root(parser.parse_args().root)


def main_with_root(root: Path) -> int:
    root = root.resolve()
    findings: list[str] = []
    for path in candidate_files(root):
        try:
            text = path.read_text(encoding="utf-8")
        except (OSError, UnicodeError):
            continue
        for number, line in enumerate(text.splitlines(), start=1):
            for label, pattern in PATTERNS.items():
                if pattern.search(line):
                    findings.append(f"{path.relative_to(root)}:{number}: {label}")
    if findings:
        print("Sensitive-content check failed:", file=sys.stderr)
        print(*[f"- {finding}" for finding in findings], sep="\n", file=sys.stderr)
        return 1
    print("Sensitive-content check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
