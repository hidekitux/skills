#!/usr/bin/env python3
"""Validate the triage label set on an open Issue.

Rules:
- Exactly one `priority:` label from high, medium, or low.
- Exactly one `scope:` label from feature, bug, docs, maintenance,
  improvement, or release.
- Exactly one `phase:` label from backlog, planned, or in-progress.
- No label outside the defined triage set.
"""

from __future__ import annotations

import argparse
import sys

PRIORITIES = ("high", "medium", "low")
SCOPES = ("feature", "bug", "docs", "maintenance", "improvement", "release")
PHASES = ("backlog", "planned", "in-progress")

ALLOWED_LABELS = {
    "priority": PRIORITIES,
    "scope": SCOPES,
    "phase": PHASES,
}


def label_errors(labels: list[str]) -> list[str]:
    """Return the validation errors for a triage label set."""
    errors: list[str] = []
    if not labels:
        errors.append("at least one label is required (priority:, scope:, phase:)")
        return errors

    counts = {"priority": 0, "scope": 0, "phase": 0}
    for label in labels:
        prefix, sep, value = label.partition(":")
        allowed = ALLOWED_LABELS.get(prefix)
        if not sep or allowed is None:
            errors.append(f"unknown label: {label!r}")
            continue
        if value not in allowed:
            errors.append(f"unknown label value for {prefix}: {label!r}")
            continue
        counts[prefix] += 1

    for dimension in ("priority", "scope", "phase"):
        if counts[dimension] != 1:
            errors.append(
                f"exactly one {dimension}: label is required; found {counts[dimension]}"
            )
    return errors


def main_with_labels(labels: list[str]) -> int:
    errors = label_errors(labels)

    if errors:
        print("error: " + "; ".join(errors), file=sys.stderr)
        return 1
    print("Issue triage labels are valid.")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--labels",
        required=True,
        help="comma-separated label names from the issue event payload",
    )
    args = parser.parse_args()
    labels = [label.strip() for label in args.labels.split(",") if label.strip()]
    return main_with_labels(labels)


if __name__ == "__main__":
    raise SystemExit(main())
