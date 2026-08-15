#!/usr/bin/env python3
"""Check that a repository release tag matches the skills catalog."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

import yaml

TAG_PATTERN = re.compile(r"^v(?P<version>\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$")


def run_git(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="release tag in the form vX.Y.Z")
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[2]
    )
    args = parser.parse_args()
    root = args.root.resolve()
    errors: list[str] = []

    match = TAG_PATTERN.fullmatch(args.tag)
    if not match:
        errors.append("release tag must match vX.Y.Z (with an optional semver suffix)")
    version = match.group("version") if match else None

    catalog_path = root / "CATALOG.yml"
    try:
        catalog = yaml.safe_load(catalog_path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError) as exc:
        errors.append(f"CATALOG.yml cannot be parsed: {exc}")
        catalog = {}

    entries = catalog.get("skills") if isinstance(catalog, dict) else None
    if not isinstance(entries, list) or not entries:
        errors.append("CATALOG.yml must contain at least one skill for a release")
    else:
        for index, entry in enumerate(entries, start=1):
            if not isinstance(entry, dict):
                errors.append(f"CATALOG.yml skills[{index}] must be a mapping")
                continue
            if version and entry.get("version") != version:
                errors.append(
                    f"CATALOG.yml skills[{index}] version {entry.get('version')!r} "
                    f"does not match {version!r}"
                )

    try:
        if run_git(root, "diff", "--quiet") != "":
            errors.append("working tree has unstaged changes")
    except subprocess.CalledProcessError:
        errors.append("working tree has unstaged changes")
    try:
        if run_git(root, "diff", "--cached", "--quiet") != "":
            errors.append("index has staged changes not committed")
    except subprocess.CalledProcessError:
        errors.append("index has staged changes not committed")
    try:
        untracked = run_git(root, "ls-files", "--others", "--exclude-standard")
        if untracked:
            errors.append(
                "working tree has untracked files: " + ", ".join(untracked.splitlines())
            )
    except subprocess.CalledProcessError as exc:
        errors.append(f"could not inspect Git state: {exc}")

    try:
        existing = run_git(
            root, "rev-parse", "--verify", "--quiet", f"refs/tags/{args.tag}"
        )
        if existing:
            errors.append(f"tag {args.tag} already exists locally")
    except subprocess.CalledProcessError:
        pass

    try:
        remote = run_git(root, "remote", "get-url", "origin")
    except subprocess.CalledProcessError:
        errors.append("remote origin is required to verify the published tag")
    else:
        remote_result = subprocess.run(
            [
                "git",
                "ls-remote",
                "--exit-code",
                "--refs",
                remote,
                f"refs/tags/{args.tag}",
            ],
            cwd=root,
            capture_output=True,
            check=False,
            text=True,
        )
        if remote_result.returncode == 0:
            errors.append(f"tag {args.tag} already exists on the origin remote")
        elif remote_result.returncode != 2:
            errors.append(f"could not inspect tag {args.tag} on the origin remote")

    if errors:
        print("Release verification failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(
        f"Release contract is valid for {args.tag}: {len(entries)} skill(s) at version {version}."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
