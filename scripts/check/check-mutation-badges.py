#!/usr/bin/env python3
"""Compare README FSL mutation badges with the recorded mutation summary."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

KILLED_PATTERN = re.compile(r"badge/mutants%20killed-(\d+)%2F(\d+)-[0-9a-f]{6}")
RATE_PATTERN = re.compile(r"badge/kill%20rate-(\d+)\.(\d{2})%25-[0-9a-f]{6}")
SURVIVED_PATTERN = re.compile(r"badge/surviving%20mutants-(\d+)-[0-9a-f]{6}")


def read_recorded_summary(root: Path) -> dict:
    path = root / "docs" / "fsl-mutation-summary.json"
    try:
        content = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        print(
            "FSL mutation-badge check failed: cannot read "
            f"{path.relative_to(root)}: {exc}",
            file=sys.stderr,
        )
        raise SystemExit(2) from exc
    return content


def parse_badges(readme: Path) -> tuple[int, int, int, int, int] | None:
    text = readme.read_text(encoding="utf-8")
    killed = KILLED_PATTERN.search(text)
    rate = RATE_PATTERN.search(text)
    survived = SURVIVED_PATTERN.search(text)
    if not all((killed, rate, survived)):
        print(
            "FSL mutation-badge check failed: README is missing one or more "
            "FSL mutation badges",
            file=sys.stderr,
        )
        return None
    return (
        int(killed.group(1)),
        int(killed.group(2)),
        int(rate.group(1)),
        int(rate.group(2)),
        int(survived.group(1)),
    )


def expected_kill_rate(killed: int, potential: int) -> tuple[int, int]:
    """Return the (whole, fraction) of the kill rate to two decimal places."""
    value = killed / potential * 100
    fraction = int(round(value * 100) % 100)
    return int(value), fraction


def main_with_root(root: Path) -> int:
    root = root.resolve()
    summary = read_recorded_summary(root)
    try:
        potential = int(summary["potential"])
        killed = int(summary["killed"])
        survived = int(summary["survived"])
    except (KeyError, TypeError, ValueError) as exc:
        print(
            f"FSL mutation-badge check failed: malformed summary: {exc}",
            file=sys.stderr,
        )
        return 1
    if potential <= 0:
        print(
            "FSL mutation-badge check failed: recorded potential must be positive",
            file=sys.stderr,
        )
        return 1

    expected_percent, expected_fraction = expected_kill_rate(killed, potential)

    badges = parse_badges(root / "README.md")
    if badges is None:
        return 1
    b_killed, b_potential, b_percent, b_fraction, b_survived = badges

    errors: list[str] = []
    if b_killed != killed:
        errors.append(f"mutants killed {b_killed} does not match recorded {killed}")
    if b_potential != potential:
        errors.append(
            f"mutants killed denominator {b_potential} does not match recorded {potential}"
        )
    if (b_percent, b_fraction) != (expected_percent, expected_fraction):
        errors.append(
            f"kill rate {b_percent}.{b_fraction:02d}% does not match recorded "
            f"{expected_percent}.{expected_fraction:02d}%"
        )
    if b_survived != survived:
        errors.append(
            f"surviving mutants {b_survived} does not match recorded {survived}"
        )
    if errors:
        print("FSL mutation-badge check failed:", file=sys.stderr)
        print(*[f"- {error}" for error in errors], sep="\n", file=sys.stderr)
        return 1
    print(
        f"FSL mutation badges are current: {killed}/{potential} killed, "
        f"{expected_percent}.{expected_fraction:02d}% kill rate, {survived} surviving."
    )
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[2]
    )
    return main_with_root(parser.parse_args().root)


if __name__ == "__main__":
    raise SystemExit(main())
