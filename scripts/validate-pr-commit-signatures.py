"""Require every commit in a pull request to have a GitHub-verified signature."""

from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", help="repository in OWNER/REPO form")
    parser.add_argument("--pull-request", type=int, help="positive pull request number")
    parser.add_argument(
        "--commits-json",
        type=Path,
        help="local API response fixture; use only for deterministic validation",
    )
    return parser.parse_args()


def load_fixture(path: Path) -> list[dict[str, Any]]:
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(f"cannot load commits fixture: {error}") from error
    if not isinstance(payload, list) or any(
        not isinstance(item, dict) for item in payload
    ):
        raise ValueError("commits fixture must be a JSON array of commit objects")
    return payload


def fetch_commits(repo: str, pull_request: int, token: str) -> list[dict[str, Any]]:
    commits: list[dict[str, Any]] = []
    page = 1
    while True:
        request = urllib.request.Request(
            f"https://api.github.com/repos/{repo}/pulls/{pull_request}/commits?per_page=100&page={page}",
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {token}",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        try:
            with urllib.request.urlopen(request) as response:
                payload = json.load(response)
        except urllib.error.URLError as error:
            raise ValueError(f"cannot fetch pull-request commits: {error}") from error
        if not isinstance(payload, list) or any(
            not isinstance(item, dict) for item in payload
        ):
            raise ValueError("GitHub returned an invalid pull-request commits response")
        commits.extend(payload)
        if len(payload) < 100:
            return commits
        page += 1


def invalid_commits(commits: list[dict[str, Any]]) -> list[str]:
    invalid: list[str] = []
    for commit in commits:
        sha = commit.get("sha", "unknown")
        commit_data = commit.get("commit")
        verification = (
            commit_data.get("verification", {}) if isinstance(commit_data, dict) else {}
        )
        if (
            not isinstance(verification, dict)
            or verification.get("verified") is not True
        ):
            reason = (
                verification.get("reason", "missing verification")
                if isinstance(verification, dict)
                else "missing verification"
            )
            invalid.append(f"{sha}: {reason}")
    return invalid


def main() -> int:
    args = parse_args()
    if args.commits_json:
        try:
            commits = load_fixture(args.commits_json)
        except ValueError as error:
            print(f"error: {error}", file=sys.stderr)
            return 2
    else:
        if (
            not args.repo
            or args.repo.count("/") != 1
            or any(not part for part in args.repo.split("/", 1))
        ):
            print("error: --repo must be OWNER/REPO", file=sys.stderr)
            return 2
        if not args.pull_request or args.pull_request < 1:
            print("error: --pull-request must be positive", file=sys.stderr)
            return 2
        token = os.getenv("GITHUB_TOKEN")
        if not token:
            print("error: GITHUB_TOKEN is required", file=sys.stderr)
            return 2
        try:
            commits = fetch_commits(args.repo, args.pull_request, token)
        except ValueError as error:
            print(f"error: {error}", file=sys.stderr)
            return 2

    if not commits:
        print("error: pull request contains no commits", file=sys.stderr)
        return 1
    invalid = invalid_commits(commits)
    if invalid:
        print("error: pull request contains unverified commits:", file=sys.stderr)
        print("\n".join(f"- {commit}" for commit in invalid), file=sys.stderr)
        return 1
    print(f"All {len(commits)} pull-request commits have GitHub-verified signatures.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
