#!/usr/bin/env python3
"""Configure rebase-only merging and a protected GitHub branch ruleset."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from typing import Any

DEFAULT_RULESET_NAME = "Require pull requests on protected branches"
DEFAULT_ISSUE_RULESET_NAME = "Require signed commits on issue branches"
REPOSITORY_SETTINGS = {
    "allow_merge_commit": False,
    "allow_squash_merge": False,
    "allow_rebase_merge": True,
    "delete_branch_on_merge": True,
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True, metavar="OWNER/REPO")
    parser.add_argument(
        "--branch",
        action="append",
        metavar="NAME",
        help="Protected branch; repeat to override the default main target.",
    )
    parser.add_argument(
        "--required-check",
        action="append",
        default=[],
        metavar="CONTEXT",
        help="Required status-check context; repeat for every CI check.",
    )
    parser.add_argument("--ruleset-name", default=DEFAULT_RULESET_NAME)
    parser.add_argument(
        "--issue-branch-pattern",
        action="append",
        metavar="PATTERN",
        help="Signed work-branch pattern; repeat to override the default issue/*.",
    )
    parser.add_argument("--issue-ruleset-name", default=DEFAULT_ISSUE_RULESET_NAME)
    parser.add_argument("--approvals", type=int, default=1)
    parser.add_argument(
        "--allow-merge-method", action="append", choices=("merge", "squash", "rebase")
    )
    parser.add_argument("--require-code-owner-review", action="store_true")
    parser.add_argument("--allow-last-push-approval", action="store_true")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Apply the change. Without this flag, print the payload only.",
    )
    return parser.parse_args()


def command(*args: str, input_text: str | None = None) -> str:
    result = subprocess.run(
        args,
        check=False,
        input=input_text,
        text=True,
        capture_output=True,
    )
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or "command failed")
    return result.stdout


def build_payload(args: argparse.Namespace) -> dict[str, Any]:
    if args.repo.count("/") != 1 or any(not value for value in args.repo.split("/", 1)):
        raise ValueError("--repo must be OWNER/REPO")
    if args.approvals < 0 or args.approvals > 6:
        raise ValueError("--approvals must be between 0 and 6")
    if args.approvals == 0 and not args.allow_last_push_approval:
        raise ValueError(
            "--approvals 0 requires --allow-last-push-approval so a solo workflow "
            "does not require another person's approval"
        )
    if not args.required_check:
        raise ValueError("supply at least one --required-check so merges require CI")

    branches = args.branch or ["main"]
    if any(not branch or branch.startswith("refs/") for branch in branches):
        raise ValueError("--branch values must be unqualified branch names")
    if len(set(branches)) != len(branches):
        raise ValueError("--branch values must be unique")
    if any(not context.strip() for context in args.required_check):
        raise ValueError("--required-check values must not be empty")

    merge_methods = args.allow_merge_method or ["rebase"]
    return {
        "name": args.ruleset_name,
        "target": "branch",
        "enforcement": "active",
        "bypass_actors": [],
        "conditions": {
            "ref_name": {
                "include": [f"refs/heads/{branch}" for branch in branches],
                "exclude": [],
            }
        },
        "rules": [
            {
                "type": "pull_request",
                "parameters": {
                    "allowed_merge_methods": merge_methods,
                    "dismiss_stale_reviews_on_push": True,
                    "require_code_owner_review": args.require_code_owner_review,
                    "require_last_push_approval": not args.allow_last_push_approval,
                    "required_approving_review_count": args.approvals,
                    "required_review_thread_resolution": True,
                },
            },
            {
                "type": "required_status_checks",
                "parameters": {
                    "strict_required_status_checks_policy": True,
                    "required_status_checks": [
                        {"context": context} for context in args.required_check
                    ],
                },
            },
            {"type": "non_fast_forward"},
            {"type": "deletion"},
            {"type": "required_linear_history"},
        ],
    }


def build_issue_payload(args: argparse.Namespace) -> dict[str, Any]:
    patterns = args.issue_branch_pattern or ["issue/*"]
    if any(
        not pattern or pattern.startswith("refs/") or pattern.strip() != pattern
        for pattern in patterns
    ):
        raise ValueError(
            "--issue-branch-pattern values must be nonempty branch patterns"
        )
    if len(set(patterns)) != len(patterns):
        raise ValueError("--issue-branch-pattern values must be unique")

    return {
        "name": args.issue_ruleset_name,
        "target": "branch",
        "enforcement": "active",
        "bypass_actors": [],
        "conditions": {
            "ref_name": {
                "include": [f"refs/heads/{pattern}" for pattern in patterns],
                "exclude": [],
            }
        },
        "rules": [{"type": "required_signatures"}],
    }


def find_existing_ruleset(repo: str, name: str) -> int | None:
    output = command("gh", "api", f"repos/{repo}/rulesets", "--paginate", "--slurp")
    pages = json.loads(output)
    if not isinstance(pages, list) or any(not isinstance(page, list) for page in pages):
        raise RuntimeError("ruleset listing returned an unexpected response shape")
    if any(not isinstance(item, dict) for page in pages for item in page):
        raise RuntimeError("ruleset listing contained an unexpected item")
    matches = [
        item["id"]
        for page in pages
        for item in page
        if item.get("name") == name and item.get("source_type") == "Repository"
    ]
    if len(matches) > 1:
        raise RuntimeError(
            f"multiple rulesets are named {name!r}; rename or remove the duplicates first"
        )
    return matches[0] if matches else None


def verify(actual: dict[str, Any], expected: dict[str, Any]) -> None:
    if actual.get("enforcement") != "active" or actual.get("bypass_actors"):
        raise RuntimeError(
            "ruleset verification failed: enforcement or bypass actors differ"
        )
    if actual.get("conditions") != expected["conditions"]:
        raise RuntimeError("ruleset verification failed: branch targets differ")

    actual_rules = {
        rule.get("type"): rule.get("parameters", {}) for rule in actual.get("rules", [])
    }
    for expected_rule in expected["rules"]:
        rule_type = expected_rule["type"]
        if rule_type not in actual_rules:
            raise RuntimeError(f"ruleset verification failed: {rule_type} is missing")
        for key, value in expected_rule.get("parameters", {}).items():
            if actual_rules[rule_type].get(key) != value:
                raise RuntimeError(
                    f"ruleset verification failed: {rule_type}.{key} differs"
                )


def configure_repository(repo: str) -> None:
    result = command(
        "gh",
        "api",
        "--method",
        "PATCH",
        f"repos/{repo}",
        *[
            argument
            for key, value in REPOSITORY_SETTINGS.items()
            for argument in ("-f", f"{key}={str(value).lower()}")
        ],
    )
    actual = json.loads(result)
    for key, value in REPOSITORY_SETTINGS.items():
        if actual.get(key) != value:
            raise RuntimeError(
                f"repository settings verification failed: {key} differs"
            )


def apply_ruleset(repo: str, payload: dict[str, Any]) -> tuple[str, int]:
    ruleset_id = find_existing_ruleset(repo, payload["name"])
    endpoint = (
        f"repos/{repo}/rulesets"
        if ruleset_id is None
        else f"repos/{repo}/rulesets/{ruleset_id}"
    )
    method = "POST" if ruleset_id is None else "PUT"
    result = command(
        "gh",
        "api",
        "--method",
        method,
        endpoint,
        "-H",
        "Accept: application/vnd.github+json",
        "-H",
        "X-GitHub-Api-Version: 2022-11-28",
        "--input",
        "-",
        input_text=json.dumps(payload, indent=2) + "\n",
    )
    created = json.loads(result)
    actual = json.loads(command("gh", "api", f"repos/{repo}/rulesets/{created['id']}"))
    verify(actual, payload)
    return ("Created" if ruleset_id is None else "Updated", created["id"])


def main() -> int:
    args = parse_args()
    try:
        payload = build_payload(args)
        issue_payload = build_issue_payload(args)
    except ValueError as error:
        print(f"error: {error}", file=sys.stderr)
        return 2

    if not args.apply:
        print(
            json.dumps(
                {
                    "repository_settings": REPOSITORY_SETTINGS,
                    "ruleset": payload,
                    "issue_ruleset": issue_payload,
                },
                indent=2,
            )
        )
        print(
            "Dry run only. Review the payload, then rerun with --apply.",
            file=sys.stderr,
        )
        return 0

    if shutil.which("gh") is None:
        print("error: GitHub CLI (gh) is required", file=sys.stderr)
        return 2

    try:
        command("gh", "auth", "status")
        configure_repository(args.repo)
        main_action, main_id = apply_ruleset(args.repo, payload)
        issue_action, issue_id = apply_ruleset(args.repo, issue_payload)
    except (RuntimeError, json.JSONDecodeError, KeyError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1

    print(f"{main_action} and verified ruleset {main_id} for {args.repo}.")
    print(f"{issue_action} and verified ruleset {issue_id} for {args.repo}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
