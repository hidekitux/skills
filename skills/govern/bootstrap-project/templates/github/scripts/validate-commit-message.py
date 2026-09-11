#!/usr/bin/env python3
"""Validate commit message one-sentence and issue-number policy.

The commit message must be a single Conventional Commits header line:

    type(scope): summary #NNN

The header must end with ` #NNN` where NNN is an Issue number (the governing
Issue for that commit; one Pull Request may handle multiple Issues, so the
number need not match the branch), and the message must not contain a body or
footer. The header itself is validated by commitlint; this script checks the
message shape that commitlint cannot.
"""

from __future__ import annotations

import argparse
import re
import sys

HEADER_PATTERN = re.compile(r"\A[^:\n]+: [^\n]+ #\d+\Z")


def validate(message: str) -> list[str]:
    errors: list[str] = []
    lines = message.splitlines()
    if not lines:
        return ["commit message must not be empty"]
    if len(lines) != 1:
        errors.append("commit message must be exactly one line")
        return errors
    if not HEADER_PATTERN.fullmatch(lines[0]):
        errors.append(
            "header must be `type(scope): summary #NNN` with a numeric issue number"
        )
        return errors
    summary = lines[0].split(":", 1)[1].strip()
    if re.search(r"[.!?]\s*#\d+\Z", summary):
        errors.append("summary must be a single sentence without terminal punctuation")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--message", required=True)
    args = parser.parse_args()
    errors = validate(args.message)
    if errors:
        print("error: " + "; ".join(errors), file=sys.stderr)
        return 1
    print("Commit message shape is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
