#!/usr/bin/env python3
"""Check that a repository release tag matches the skills catalog."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

TAG_PATTERN = re.compile(r"^v(?P<version>\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$")


def find_skill_directories(root: Path) -> dict[str, Path]:
    """Map every discovered publishable skill name to its directory."""
    discovered: dict[str, Path] = {}
    for skill_file in sorted((root / "skills").rglob("SKILL.md")):
        discovered[skill_file.parent.name] = skill_file.parent
    return discovered


def find_cross_skill_references(root: Path, catalog_names: set[str]) -> list[str]:
    """Return errors when a released skill references a skill outside the catalog.

    A deterministic, standalone helper (no YAML dependency) so tests can import
    it directly. Every published (catalog) skill body is scanned for references
    to any other skill discovered in skills/; if that referenced skill is not
    part of this release's catalog, the release would ship a broken pointer, so
    verify-release rejects it.
    """
    skill_dirs = find_skill_directories(root)
    errors: list[str] = []
    for name in sorted(catalog_names):
        skill_dir = skill_dirs.get(name)
        if skill_dir is None:
            continue  # a catalog entry with no skill is reported elsewhere
        try:
            body = (skill_dir / "SKILL.md").read_text(encoding="utf-8")
        except (OSError, UnicodeError) as exc:
            errors.append(f"{name}: cannot read SKILL.md: {exc}")
            continue
        for other in sorted(skill_dirs):
            if other == name:
                continue
            if other not in catalog_names and re.search(
                rf"(?<![A-Za-z0-9-]){re.escape(other)}(?![A-Za-z0-9-])",
                body,
                re.IGNORECASE,
            ):
                errors.append(
                    f"{name} references skill {other!r} which is not "
                    "listed in the release catalog"
                )
    return errors


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
    import yaml

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
    catalog_names = (
        {entry.get("name") for entry in entries if isinstance(entry, dict)}
        if isinstance(entries, list)
        else set()
    )
    errors.extend(find_cross_skill_references(root, catalog_names))
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
