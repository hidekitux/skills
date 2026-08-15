#!/usr/bin/env python3
"""Require a reviewed license attestation for every mise-managed tool."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import tomllib


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parent.parent
    )
    return main_with_root(parser.parse_args().root)


def main_with_root(root: Path) -> int:
    root = root.resolve()
    try:
        mise = tomllib.loads((root / "mise.toml").read_text(encoding="utf-8"))
        registry = tomllib.loads(
            (root / "TOOL_LICENSES.toml").read_text(encoding="utf-8")
        )
    except (OSError, tomllib.TOMLDecodeError) as exc:
        print(f"Tool-license check failed: {exc}", file=sys.stderr)
        return 1
    tools = mise.get("tools", {})
    attestations = registry.get("tools", {}) if isinstance(registry, dict) else {}
    errors: list[str] = []
    for tool in tools:
        entry = attestations.get(tool)
        if not isinstance(entry, dict):
            errors.append(f"{tool}: missing license attestation")
            continue
        if not isinstance(entry.get("license"), str) or not entry["license"].strip():
            errors.append(f"{tool}: license must be non-empty")
        if not isinstance(entry.get("source"), str) or not entry["source"].startswith(
            "https://"
        ):
            errors.append(f"{tool}: source must be a public https URL")
    errors.extend(
        f"{tool}: attestation has no mise tool"
        for tool in sorted(set(attestations) - set(tools))
    )
    if errors:
        print("Tool-license check failed:", file=sys.stderr)
        print(*[f"- {error}" for error in errors], sep="\n", file=sys.stderr)
        return 1
    print(f"Tool-license check passed: {len(tools)} tool(s) attested.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
