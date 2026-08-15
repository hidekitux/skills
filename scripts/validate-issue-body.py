#!/usr/bin/env python3
"""Validate required headings in repository Issue bodies."""

from __future__ import annotations

import argparse
import re
import sys


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--title", required=True)
    parser.add_argument("--body", required=True)
    args = parser.parse_args()
    required = ["Context", "Goal", "Scope", "Acceptance criteria", "Validation"]
    levels = [2] * len(required)
    if args.title.startswith("[Release]:"):
        required += ["Changelog", "Added", "Changed", "Fixed", "Removed"]
        levels += [2, 3, 3, 3, 3]
    missing = [
        heading
        for heading, level in zip(required, levels)
        if not re.search(rf"(?m)^{'#' * level} {re.escape(heading)}\s*$", args.body)
    ]
    if missing:
        print(
            f"error: Issue body is missing headings: {', '.join(missing)}",
            file=sys.stderr,
        )
        return 1
    print("Issue body is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
