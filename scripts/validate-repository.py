#!/usr/bin/env python3
"""Validate repository-wide metadata and skill authoring conventions."""

from __future__ import annotations

import argparse
import hashlib
import re
import sys
from pathlib import Path
from typing import Any

import yaml

EXPECTED_LICENSE = "Apache-2.0"
EXPECTED_LICENSE_SHA256 = (
    "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
)
EXPECTED_NOTICE = """hidekitux/skills
Copyright 2026 Hideki Tanaka

This repository and its published skills are licensed under the Apache License,
Version 2.0. See the LICENSE file for the complete license text.
"""
VALID_STATUSES = {"experimental", "stable", "deprecated"}
VERSION_PATTERN = re.compile(r"^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$")
FRONTMATTER_PATTERN = re.compile(r"\A---\n(.*?)\n---\n", re.DOTALL)
TODO_HEADING_PATTERN = re.compile(r"(?im)^#{1,6}\s+.*todo list.*$")


def parse_frontmatter(path: Path) -> tuple[dict[str, Any], str]:
    text = path.read_text(encoding="utf-8")
    match = FRONTMATTER_PATTERN.match(text)
    if not match:
        raise ValueError("missing YAML frontmatter")
    metadata = yaml.safe_load(match.group(1))
    if not isinstance(metadata, dict):
        raise TypeError("frontmatter must be a YAML mapping")
    return metadata, text[match.end() :]


def validate_license(path: Path, errors: list[str]) -> None:
    if not path.is_file():
        errors.append("LICENSE is missing")
        return
    try:
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
    except OSError as exc:
        errors.append(f"LICENSE cannot be read: {exc}")
        return
    if digest != EXPECTED_LICENSE_SHA256:
        errors.append("LICENSE must contain the unmodified Apache-2.0 text")


def validate_notice(path: Path, errors: list[str]) -> None:
    if not path.is_file():
        errors.append("NOTICE is missing")
        return
    try:
        notice = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as exc:
        errors.append(f"NOTICE cannot be read: {exc}")
        return
    if notice != EXPECTED_NOTICE:
        errors.append(
            "NOTICE must retain the confirmed repository attribution and "
            "Apache-2.0 notice"
        )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parent.parent
    )
    args = parser.parse_args()
    root = args.root.resolve()
    errors: list[str] = []

    catalog_path = root / "CATALOG.yml"
    validate_license(root / "LICENSE", errors)
    validate_notice(root / "NOTICE", errors)
    if not catalog_path.is_file():
        errors.append("CATALOG.yml is missing")
        print_errors(errors)
        return 1

    try:
        catalog = yaml.safe_load(catalog_path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError) as exc:
        errors.append(f"CATALOG.yml cannot be parsed: {exc}")
        print_errors(errors)
        return 1

    if not isinstance(catalog, dict):
        errors.append("CATALOG.yml must contain a YAML mapping")
        print_errors(errors)
        return 1
    if catalog.get("license") != EXPECTED_LICENSE:
        errors.append("CATALOG.yml must declare license: Apache-2.0")
    entries = catalog.get("skills")
    if not isinstance(entries, list):
        errors.append("CATALOG.yml must contain a skills list")
        print_errors(errors)
        return 1

    skill_files = sorted((root / "skills").rglob("SKILL.md"))
    skill_by_name: dict[str, list[Path]] = {}
    for skill_path in skill_files:
        skill_by_name.setdefault(skill_path.parent.name, []).append(skill_path)

        try:
            metadata, body = parse_frontmatter(skill_path)
        except (OSError, ValueError, yaml.YAMLError) as exc:
            errors.append(f"{relative(skill_path, root)}: {exc}")
            continue

        relative_path = relative(skill_path, root)
        if metadata.get("name") != skill_path.parent.name:
            errors.append(f"{relative_path}: frontmatter name must match its directory")
        if (
            not isinstance(metadata.get("description"), str)
            or not metadata["description"].strip()
        ):
            errors.append(f"{relative_path}: description is required")
        if metadata.get("license") != EXPECTED_LICENSE:
            errors.append(f"{relative_path}: license must be Apache-2.0")
        body_lower = body.lower()
        if not TODO_HEADING_PATTERN.search(body):
            errors.append(f"{relative_path}: a Todo List heading is required")
        for required_term in ("complete", "handoff"):
            if required_term not in body_lower:
                errors.append(
                    f"{relative_path}: Todo List guidance must explain {required_term!r}"
                )

    catalog_names: set[str] = set()
    for index, entry in enumerate(entries, start=1):
        prefix = f"CATALOG.yml skills[{index}]"
        if not isinstance(entry, dict):
            errors.append(f"{prefix} must be a mapping")
            continue
        name = entry.get("name")
        if not isinstance(name, str) or not name:
            errors.append(f"{prefix}.name is required")
            continue
        if name in catalog_names:
            errors.append(f"{prefix}: duplicate skill name {name!r}")
        catalog_names.add(name)
        matches = skill_by_name.get(name, [])
        if not matches:
            errors.append(f"{prefix}: no skills/**/{name}/SKILL.md found")
        elif len(matches) > 1:
            errors.append(
                f"{prefix}: skill name is ambiguous; add a unique catalog path"
            )

        for field in ("summary", "owner", "status", "license", "version"):
            if not entry.get(field):
                errors.append(f"{prefix}.{field} is required")
        if entry.get("status") not in VALID_STATUSES:
            errors.append(f"{prefix}.status must be one of {sorted(VALID_STATUSES)}")
        if entry.get("license") != EXPECTED_LICENSE:
            errors.append(f"{prefix}.license must be Apache-2.0")
        if not isinstance(entry.get("version"), str) or not VERSION_PATTERN.fullmatch(
            entry["version"]
        ):
            errors.append(f"{prefix}.version must be semantic version text")

        adapters = entry.get("host_adapters", [])
        if adapters is None:
            adapters = []
        if not isinstance(adapters, list):
            errors.append(f"{prefix}.host_adapters must be a list")
        elif len(matches) == 1:
            skill_dir = matches[0].parent
            for host in adapters:
                if not isinstance(host, str) or not host:
                    errors.append(f"{prefix}.host_adapters contains an invalid host")
                    continue
                adapter = skill_dir / "references" / "hosts" / f"{host}.md"
                if not adapter.is_file():
                    errors.append(
                        f"{prefix}: missing host adapter {relative(adapter, root)}"
                    )

    discovered_names = set(skill_by_name)
    missing_catalog = discovered_names - catalog_names
    for name in sorted(missing_catalog):
        errors.append(f"skill {name!r} is missing from CATALOG.yml")

    if errors:
        print_errors(errors)
        return 1
    print(
        f"Repository metadata is valid: {len(skill_files)} skill(s), {len(entries)} catalog entr{'y' if len(entries) == 1 else 'ies'}."
    )
    return 0


def relative(path: Path, root: Path) -> str:
    return str(path.relative_to(root))


def print_errors(errors: list[str]) -> None:
    print("Repository validation failed:", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())
