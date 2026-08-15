#!/usr/bin/env python3
"""Validate shared Issue and Pull Request title conventions."""

from __future__ import annotations

import argparse
import re
import sys

TYPES = "Feature|Bug|Improvement|Documentation|Security|Maintenance|Release"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--title", required=True)
    args = parser.parse_args()
    release = r"\[Release\]: v[0-9]+\.[0-9]+\.[0-9]+"
    standard = rf"\[(?:{TYPES})\]: [A-Z].+"
    if re.fullmatch(rf"(?:{release}|{standard})", args.title):
        print("Work item title is valid.")
        return 0
    print(
        "error: title must be [Type]: Summary beginning with a capital letter; "
        "Release uses [Release]: vX.Y.Z",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
