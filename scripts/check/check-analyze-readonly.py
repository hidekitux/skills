#!/usr/bin/env python3
"""Reject Issue- or PR-creation instructions in analyze-* skill bodies.

Analyze-* skills are read-only: they inspect code and report findings but must
not instruct an agent to create Issues or Pull Requests. This deterministic
check scans only publishable skill bodies whose directory is named analyze-*.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# Only analyze-* skills are governed by the read-only contract.
ANALYZE_PREFIX = "analyze-"

# Frontmatter is stripped so a defensive summary such as "does not create
# issues" in the description does not trip the instruction detector.
FRONTMATTER_RE = re.compile(r"\A---\r?\n.*?\r?\n---\r?\n", re.DOTALL)

FORBIDDEN = {
    "Issue-creation instruction": re.compile(
        r"\b(?:create|open|file|submit|raise|log)\s+"
        r"(?:an?\s+|the\s+)?(?:github\s+)?issues?\b",
        re.IGNORECASE,
    ),
    "Pull-request-creation instruction": re.compile(
        r"\b(?:create|open|submit|raise)\s+"
        r"(?:an?\s+|the\s+)?(?:github\s+)?pull\s+requests?\b",
        re.IGNORECASE,
    ),
    "PR-creation instruction": re.compile(
        r"\b(?:create|open|submit|raise)\s+(?:an?\s+|the\s+)?pr\b",
        re.IGNORECASE,
    ),
}


def skill_body(path: Path) -> str:
    """Return the MARKDOWN body of a SKILL.md, excluding its frontmatter."""
    text = path.read_text(encoding="utf-8")
    match = FRONTMATTER_RE.match(text)
    return text[match.end() :] if match else text


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[2]
    )
    return main_with_root(parser.parse_args().root)


def main_with_root(root: Path) -> int:
    root = root.resolve()
    findings: list[str] = []
    for skill_file in sorted((root / "skills").rglob("SKILL.md")):
        if not skill_file.parent.name.startswith(ANALYZE_PREFIX):
            continue
        try:
            body = skill_body(skill_file)
        except (OSError, UnicodeError) as exc:
            findings.append(f"{skill_file.relative_to(root)}: cannot read: {exc}")
            continue
        for number, line in enumerate(body.splitlines(), start=1):
            for label, pattern in FORBIDDEN.items():
                if pattern.search(line):
                    findings.append(f"{skill_file.relative_to(root)}:{number}: {label}")
    if findings:
        print("Analyze read-only check failed:", file=sys.stderr)
        print(*[f"- {finding}" for finding in findings], sep="\n", file=sys.stderr)
        return 1
    print(
        "Analyze read-only check passed: no analyze-* skill instructs Issue or PR creation."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
