#!/usr/bin/env python3
"""Lint commit messages with the project-local commitlint and message policy.

Resolves the commitlint binary, collects the commits in the current range, and
runs each message through commitlint and validate-commit-message.py. Replaces
the former lint-commits.sh while preserving its environment contract and exit
codes (0 success, 1 lint failure, 2 missing commitlint).
"""

from __future__ import annotations

import importlib.util
import os
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent


def _load_commit_message_validator():
    """Import validate-commit-message.py despite the hyphenated filename."""
    path = SCRIPT_DIR / "validate-commit-message.py"
    spec = importlib.util.spec_from_file_location("validate_commit_message", path)
    assert spec and spec.loader
    validator = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(validator)
    return validator


VALIDATE_COMMIT_MESSAGE = _load_commit_message_validator()


def resolve_commitlint(root, runner=subprocess) -> str:
    """Return the commitlint binary path, exiting 2 when none is available."""
    override = os.environ.get("COMMITLINT_BIN")
    if override:
        return override
    candidate = Path(root) / ".mise" / "bin" / "commitlint"
    if os.access(candidate, os.X_OK):
        return str(candidate)
    print(
        "commitlint is not installed for this project. Run: mise run setup:commitlint",
        file=sys.stderr,
    )
    raise SystemExit(2)


def collect_commits(base, head, runner=subprocess) -> list[str]:
    """Return commit SHAs in a base..head range, or from a single head."""
    argv = ["git", "rev-list", "--no-merges"]
    argv.append(f"{base}..{head}" if base else head)
    result = runner.run(argv, text=True, capture_output=True)
    return [line for line in result.stdout.splitlines() if line]


def message_for(commit, runner=subprocess) -> str:
    """Return the full commit message body for a commit.

    Trailing newlines are stripped just as Bash command substitution did in the
    former lint-commits.sh, so a single-line message is not treated as having a
    spurious empty body line.
    """
    return runner.check_output(
        ["git", "show", "-s", "--format=%B", commit], text=True
    ).rstrip("\n")


def lint_message(message, commitlint_bin, runner=subprocess) -> int:
    """Lint one message, returning 0 on success or the failing status."""
    result = runner.run([commitlint_bin, "lint"], input=message, text=True)
    if result.returncode != 0:
        return result.returncode
    errors = VALIDATE_COMMIT_MESSAGE.validate(message)
    if errors:
        print("error: " + "; ".join(errors), file=sys.stderr)
        return 1
    print("Commit message shape is valid.")
    return 0


def main(runner=subprocess) -> int:
    root = runner.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    commitlint_bin = resolve_commitlint(root, runner)
    if os.environ.get("PR_BASE_SHA") and os.environ.get("PR_HEAD_SHA"):
        commits = collect_commits(
            os.environ["PR_BASE_SHA"], os.environ["PR_HEAD_SHA"], runner
        )
    else:
        before = os.environ.get("PUSH_BEFORE_SHA", "")
        after = os.environ.get("GITHUB_SHA") or "HEAD"
        if before and not all(character == "0" for character in before):
            commits = collect_commits(before, after, runner)
        else:
            commits = collect_commits(None, after, runner)
    if not commits:
        print("No commits to lint.")
        return 0
    for commit in commits:
        print(f"Checking commit {commit}")
        message = message_for(commit, runner)
        status = lint_message(message, commitlint_bin, runner)
        if status != 0:
            return status
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
