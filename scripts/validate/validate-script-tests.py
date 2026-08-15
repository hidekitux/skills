#!/usr/bin/env python3
"""Require each executable repository script to name a representative test."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import tomllib


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[2]
    )
    return main_with_root(parser.parse_args().root)


def main_with_root(root: Path) -> int:
    root = root.resolve()
    try:
        mapping = tomllib.loads(
            (root / "SCRIPT_TESTS.toml").read_text(encoding="utf-8")
        )
    except (OSError, tomllib.TOMLDecodeError) as exc:
        print(f"Script-test mapping check failed: {exc}", file=sys.stderr)
        return 1
    entries = mapping.get("scripts", {}) if isinstance(mapping, dict) else {}
    scripts = {
        str(path.relative_to(root))
        for path in (root / "scripts").rglob("*")
        if path.is_file() and "__pycache__" not in path.parts
    }
    errors: list[str] = []
    for script in sorted(scripts):
        test = entries.get(script)
        if not isinstance(test, str) or not (root / test).is_file():
            errors.append(f"{script}: missing existing representative test")
    errors.extend(
        f"{script}: mapping has no repository script"
        for script in sorted(set(entries) - scripts)
    )
    if errors:
        print("Script-test mapping check failed:", file=sys.stderr)
        print(*[f"- {error}" for error in errors], sep="\n", file=sys.stderr)
        return 1
    print(f"Script-test mapping check passed: {len(scripts)} script(s) mapped.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
