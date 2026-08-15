#!/usr/bin/env python3
"""Validate required headings in repository Issue bodies."""

from __future__ import annotations

import argparse
import re
import sys

COMMON_H2 = ["Context", "Goal", "Scope", "Acceptance criteria", "Validation"]
CHANGELOG_H3 = ["Added", "Changed", "Fixed", "Removed"]
H2_PATTERN = re.compile(r"(?m)^##[ \t]+([^\r\n]+?)[ \t]*$")
H3_PATTERN = re.compile(r"(?m)^###[ \t]+([^\r\n]+?)[ \t]*$")
CHECKBOX_PATTERN = re.compile(r"(?m)^- \[[ xX]\][ \t]+\S")
SCOPE_MARKER_PATTERN = re.compile(r"(?m)^- (In|Out):(?:[ \t].*)?$")


def without_comments(text: str) -> str:
    return re.sub(r"<!--.*?-->", "", text, flags=re.DOTALL)


def section_content(body: str, matches: list[re.Match[str]], index: int) -> str:
    end = matches[index + 1].start() if index + 1 < len(matches) else len(body)
    return without_comments(body[matches[index].end() : end]).strip()


def validation_errors(title: str, body: str) -> list[str]:
    release = title.startswith("[Release]:")
    expected_h2 = [*COMMON_H2, "Changelog"] if release else COMMON_H2
    h2_matches = list(H2_PATTERN.finditer(body))
    actual_h2 = [match.group(1).strip() for match in h2_matches]
    if actual_h2 != expected_h2:
        return [
            "level-two headings must appear exactly once in this order: "
            + ", ".join(expected_h2)
        ]

    errors = []
    sections = {
        name: section_content(body, h2_matches, index)
        for index, name in enumerate(actual_h2)
    }
    for name in COMMON_H2:
        if not sections[name]:
            errors.append(f"{name} must contain non-comment content")

    scope = sections["Scope"]
    scope_matches = list(SCOPE_MARKER_PATTERN.finditer(scope))
    scope_markers = [match.group(1) for match in scope_matches]
    if scope_markers != ["In", "Out"]:
        errors.append("Scope must contain exactly one - In: then one - Out: marker")
    else:
        for index, match in enumerate(scope_matches):
            end = (
                scope_matches[index + 1].start()
                if index + 1 < len(scope_matches)
                else len(scope)
            )
            inline = match.group(0).split(":", 1)[1].strip()
            nested = scope[match.end() : end].strip()
            if not inline and not nested:
                errors.append(f"Scope {match.group(1)} must contain concrete content")

    for name in ("Acceptance criteria", "Validation"):
        if not CHECKBOX_PATTERN.search(sections[name]):
            errors.append(f"{name} must contain at least one non-empty checkbox")

    h3_matches = list(H3_PATTERN.finditer(body))
    actual_h3 = [match.group(1).strip() for match in h3_matches]
    expected_h3 = CHANGELOG_H3 if release else []
    if actual_h3 != expected_h3:
        errors.append(
            "level-three headings must appear exactly once in this order: "
            + (", ".join(expected_h3) if expected_h3 else "none")
        )
    elif release:
        for index, name in enumerate(actual_h3):
            if not section_content(body, h3_matches, index):
                errors.append(f"{name} must contain an entry or - None.")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--title", required=True)
    parser.add_argument("--body", required=True)
    args = parser.parse_args()
    errors = validation_errors(args.title, args.body)
    if errors:
        for error in errors:
            print(f"error: {error}", file=sys.stderr)
        return 1
    print("Issue body is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
