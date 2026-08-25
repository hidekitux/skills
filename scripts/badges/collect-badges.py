#!/usr/bin/env python3
"""Collect FSL mutation and test results into shields.io endpoint payloads.

Reads a `mise run mutate-fsl` log, a `mise run test` log, and the pinned fslc
version from `scripts/fsl/install-fslc.sh`, then writes six shields.io endpoint
payloads (`schemaVersion`/`label`/`message`/`color`) into an output directory:
`fsl-killed.json`, `fsl-kill-rate.json`, `fsl-survived.json`, `fslc-version.json`,
`tests-status.json`, and `tests-run.json`. The parsed values are printed so a
workflow run log shows exactly what the badges display, and the script exits 2
with a clear message when any value is missing or malformed.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

FSLC_VERSION_PATTERN = re.compile(r'\bfsl_version="([^"]+)"')
TEST_COUNT_PATTERN = re.compile(r"\bRan (\d+) tests?\b")
TEST_FAILED_PATTERN = re.compile(r"\bFAILED\b")
TEST_OK_PATTERN = re.compile(r"^OK$", re.MULTILINE)

PAYLOAD_NAMES = (
    "fsl-killed.json",
    "fsl-kill-rate.json",
    "fsl-survived.json",
    "fslc-version.json",
    "tests-status.json",
    "tests-run.json",
)


@dataclass(frozen=True)
class MutationSummary:
    total: int
    killed: int
    survived: int


@dataclass(frozen=True)
class TestSummary:
    count: int
    ok: bool


def parse_mutate_log(text: str) -> MutationSummary:
    """Aggregate the summary block of every mutation document in the log."""
    decoder = json.JSONDecoder()
    summaries: list[MutationSummary] = []
    index = 0
    while index < len(text):
        try:
            document, index = decoder.raw_decode(text, index)
        except json.JSONDecodeError:
            newline = text.find("\n", index)
            if newline == -1:
                break
            index = newline + 1
            continue
        if not isinstance(document, dict):
            raise TypeError("mutate-fsl log contains a non-object JSON document")
        summary = document.get("summary")
        if not isinstance(summary, dict):
            raise TypeError("mutate-fsl document is missing its summary block")
        try:
            summaries.append(
                MutationSummary(
                    total=int(summary["total"]),
                    killed=int(summary["killed"]),
                    survived=int(summary["survived"]),
                )
            )
        except (KeyError, TypeError, ValueError) as exc:
            raise ValueError(f"malformed summary block: {exc}") from exc
    if not summaries:
        raise ValueError("mutate-fsl log contains no mutation documents")
    return MutationSummary(
        total=sum(s.total for s in summaries),
        killed=sum(s.killed for s in summaries),
        survived=sum(s.survived for s in summaries),
    )


def parse_fslc_version(text: str) -> str:
    match = FSLC_VERSION_PATTERN.search(text)
    if match is None:
        raise ValueError("cannot find the pinned fsl_version in install-fslc.sh")
    return match.group(1)


def parse_test_log(text: str) -> TestSummary:
    count_match = TEST_COUNT_PATTERN.search(text)
    if count_match is None:
        raise ValueError("test log does not report a 'Ran N tests' count")
    if TEST_FAILED_PATTERN.search(text):
        ok = False
    elif TEST_OK_PATTERN.search(text):
        ok = True
    else:
        raise ValueError("test log reports neither OK nor FAILED")
    return TestSummary(count=int(count_match.group(1)), ok=ok)


def expected_kill_rate(killed: int, total: int) -> tuple[int, int]:
    """Return the (whole, fraction) kill rate to two decimal places."""
    value = killed / total * 100
    fraction = int(round(value * 100) % 100)
    return int(value), fraction


def render_payloads(
    summary: MutationSummary, fslc_version: str, tests: TestSummary
) -> dict[str, dict]:
    percent, fraction = expected_kill_rate(summary.killed, summary.total)
    return {
        "fsl-killed.json": {
            "schemaVersion": 1,
            "label": "mutants killed",
            "message": f"{summary.killed}/{summary.total}",
            "color": "2ea44f",
        },
        "fsl-kill-rate.json": {
            "schemaVersion": 1,
            "label": "kill rate",
            "message": f"{percent}.{fraction:02d}%",
            "color": "2ea44f",
        },
        "fsl-survived.json": {
            "schemaVersion": 1,
            "label": "surviving mutants",
            "message": str(summary.survived),
            "color": "a371f7",
        },
        "fslc-version.json": {
            "schemaVersion": 1,
            "label": "fslc",
            "message": f"v{fslc_version}",
            "color": "0b6bcb",
        },
        "tests-status.json": {
            "schemaVersion": 1,
            "label": "tests",
            "message": "passing" if tests.ok else "failing",
            "color": "2ea44f" if tests.ok else "d73a4a",
        },
        "tests-run.json": {
            "schemaVersion": 1,
            "label": "tests run",
            "message": str(tests.count),
            "color": "2ea44f",
        },
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    repo_root = Path(__file__).resolve().parents[2]
    parser.add_argument("--mutate-log", type=Path, required=True)
    parser.add_argument("--test-log", type=Path, required=True)
    parser.add_argument(
        "--fslc-script",
        type=Path,
        default=repo_root / "scripts/fsl/install-fslc.sh",
    )
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args(argv)

    try:
        mutate_text = args.mutate_log.read_text(encoding="utf-8")
        test_text = args.test_log.read_text(encoding="utf-8")
        fslc_text = args.fslc_script.read_text(encoding="utf-8")
        summary = parse_mutate_log(mutate_text)
        tests = parse_test_log(test_text)
        fslc_version = parse_fslc_version(fslc_text)
        if summary.total <= 0:
            raise ValueError("aggregate mutation total must be positive")
    except (OSError, ValueError, TypeError) as exc:
        print(f"Badge collection failed: {exc}", file=sys.stderr)
        raise SystemExit(2) from exc

    percent, fraction = expected_kill_rate(summary.killed, summary.total)
    print(
        f"FSL mutation badges: {summary.killed}/{summary.total} killed, "
        f"{percent}.{fraction:02d}% kill rate, {summary.survived} surviving mutants."
    )
    print(f"FSL verifier: fslc {fslc_version}")
    print(f"Tests: {tests.count} run, {'passing' if tests.ok else 'failing'}.")

    payloads = render_payloads(summary, fslc_version, tests)
    args.output_dir.mkdir(parents=True, exist_ok=True)
    for name in PAYLOAD_NAMES:
        (args.output_dir / name).write_text(
            json.dumps(payloads[name], indent=2) + "\n", encoding="utf-8"
        )
    print(f"Wrote {len(payloads)} badge payloads to {args.output_dir}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
