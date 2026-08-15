#!/usr/bin/env python3
"""Validate an issue-based pull-request branch policy."""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path

import tomllib


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
    close = re.search(
        r"(?im)^\s*(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#([1-9][0-9]*)\s*$",
        args.body,
    )
    ok = False
    for route in routes:
        if not re.fullmatch(route["head_pattern"], args.head) or not re.fullmatch(
            route["base_pattern"], args.base
        ):
            continue
        if route.get("requires_issue_link", False):
            issue_number = int(args.head.rsplit("/", 1)[1])
            if not close or issue_number != int(close.group(1)):
                continue
        ok = True
        break
    if not ok:
        print(
            f"error: disallowed pull-request direction or issue linkage: {args.head} -> {args.base}",
            file=sys.stderr,
        )
        return 1
    print(f"Allowed pull-request direction: {args.head} -> {args.base}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
