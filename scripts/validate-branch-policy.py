#!/usr/bin/env python3
"""Validate an issue-based pull-request branch policy."""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path

import tomllib

H2_PATTERN = re.compile(r"(?m)^##[ \t]+([^\r\n]+?)[ \t]*$")
ANY_CLOSING_LINE_PATTERN = re.compile(
    r"(?im)^\s*(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#([1-9][0-9]*)\s*$"
)
ANY_REFERENCE_LINE_PATTERN = re.compile(
    r"(?im)^\s*(?:(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)|tracks)\s+#[1-9][0-9]*\s*$"
)


def without_comments(text: str) -> str:
    return re.sub(r"<!--.*?-->", "", text, flags=re.DOTALL)


def issue_references_at_start(body: str, keyword: str) -> list[int] | None:
    """Return the opening Issue-section links, or None for an invalid body."""
    reference_pattern = re.compile(rf"^{re.escape(keyword)} #([1-9][0-9]*)$")
    clean = without_comments(body)
    headings = list(H2_PATTERN.finditer(clean))
    if not headings or headings[0].group(1).strip() != "Issue":
        return None
    if clean[: headings[0].start()].strip():
        return None

    end = headings[1].start() if len(headings) > 1 else len(clean)
    section_lines = [
        line.strip()
        for line in clean[headings[0].end() : end].splitlines()
        if line.strip()
    ]
    matches = [reference_pattern.fullmatch(line) for line in section_lines]
    if not section_lines or not all(matches):
        return None

    links = [int(match.group(1)) for match in matches if match]
    all_references = ANY_REFERENCE_LINE_PATTERN.findall(clean)
    if len(all_references) != len(links):
        return None
    return links


def issue_links_at_start(body: str) -> list[int] | None:
    """Return the opening Closes links required by an Issue branch."""
    if ANY_CLOSING_LINE_PATTERN.search(body) is None:
        return None
    return issue_references_at_start(body, "Closes")


def matching_issue_link_starts_body(body: str, issue_number: int) -> bool:
    """Return whether the first Issue-section line closes the branch Issue."""
    issue_links = issue_links_at_start(body)
    return issue_links is not None and issue_links[0] == issue_number


def load(path: Path):
    with path.open("rb") as f:
        policy = tomllib.load(f)
    routes = policy.get("routes")
    if not isinstance(routes, list) or not routes:
        raise TypeError("policy must define one or more [[routes]]")
    for index, route in enumerate(routes, start=1):
        if not isinstance(route, dict):
            raise TypeError(f"routes[{index}] must be a table")
        for key in ("head_pattern", "base_pattern"):
            if not isinstance(route.get(key), str) or not route[key]:
                raise ValueError(f"routes[{index}].{key} must be non-empty text")
            re.compile(route[key])
        if "requires_issue_link" in route and not isinstance(
            route["requires_issue_link"], bool
        ):
            raise TypeError(f"routes[{index}].requires_issue_link must be boolean")
    return routes


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--config", type=Path, default=Path(".github/branch-policy.toml"))
    p.add_argument("--base", default=os.getenv("PR_BASE_REF"))
    p.add_argument("--head", default=os.getenv("PR_HEAD_REF"))
    p.add_argument("--body", default=os.getenv("PR_BODY", ""))
    p.add_argument("--validate-config", action="store_true")
    args = p.parse_args()
    try:
        routes = load(args.config)
    except (OSError, ValueError, tomllib.TOMLDecodeError) as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    if args.validate_config:
        print(f"Branch policy configuration is valid: {args.config}")
        return 0
    if not args.base or not args.head:
        print("error: --base and --head are required", file=sys.stderr)
        return 2
    ok = False
    for route in routes:
        if not re.fullmatch(route["head_pattern"], args.head) or not re.fullmatch(
            route["base_pattern"], args.base
        ):
            continue
        if route.get("requires_issue_link", False):
            issue_number = int(args.head.rsplit("/", 1)[1])
            if not matching_issue_link_starts_body(args.body, issue_number):
                continue
        ok = True
        break
    if not ok:
        print(
            "error: disallowed pull-request direction or issue linkage; "
            "an Issue-backed Pull Request must start with an Issue section "
            "whose first content line is the branch Issue's matching Closes "
            "line, with every additional Closes line kept in that section: "
            f"{args.head} -> {args.base}",
            file=sys.stderr,
        )
        return 1
    print(f"Allowed pull-request direction: {args.head} -> {args.base}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
