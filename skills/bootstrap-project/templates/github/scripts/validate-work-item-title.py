#!/usr/bin/env python3
"""Validate shared Issue and Pull Request title conventions.

Rules:
- Change work uses `[Type]: Summary` in sentence case.
- Type is one of Feature, Bug, Improvement, Documentation, Security, or
  Maintenance, with the first letter capitalized.
- Summary begins with a capitalized imperative verb; later words are
  lowercase unless ordinary English requires a capital letter, such as for
  proper nouns, acronyms, or literal identifiers.
- Releases use the explicit exception `[Release]: vX.Y.Z` or a build
  identifier `[Release]: vX.Y.Z+N`.
"""

from __future__ import annotations

import argparse
import re
import sys

TYPES = "Feature|Bug|Improvement|Documentation|Security|Maintenance"
SIMPLE_TITLE_WORD = re.compile(r"\A[A-Z][a-z]+\Z")
PRESERVED_WORDS = {
    "Actions",
    "Android",
    "Apple",
    "Codex",
    "GitHub",
    "Google",
    "iPhone",
    "iOS",
    "macOS",
    "OpenAI",
    "Windows",
}


def sentence_case_summary(summary: str) -> list[str]:
    """Flag sentence-case violations that are mechanically detectable.

    A run of three or more ordinary capitalized words after the leading
    imperative verb is the strongest title-case signal. Shorter runs can be
    legitimate proper nouns, Acronyms (ALL CAPS), canonical mixed-case names,
    and literal identifiers are preserved, so this check flags only accidental
    title casing with high confidence.
    """

    problems: list[str] = []
    words = [word for word in re.split(r"[^A-Za-z0-9]+", summary) if word]
    if not words:
        return problems
    run = 0
    run_start = 1
    for index, word in enumerate(words[1:], start=1):
        if SIMPLE_TITLE_WORD.fullmatch(word) and word not in PRESERVED_WORDS:
            if run == 0:
                run_start = index
            run += 1
        else:
            if run >= 3:
                problems.append(
                    "capitalize later words only when ordinary English requires; "
                    f"title-case run: {words[run_start : run_start + run]!r}"
                )
            run = 0
    if run >= 3:
        problems.append(
            "capitalize later words only when ordinary English requires; "
            f"title-case run: {words[run_start : run_start + run]!r}"
        )
    return problems


def title_errors(title: str) -> list[str]:
    """Return the validation errors for a work item title."""
    release = r"\[Release\]: v[0-9]+\.[0-9]+\.[0-9]+(?:\+[0-9A-Za-z.-]+)?"
    standard = rf"\[(?:{TYPES})\]: [A-Z].+"
    errors: list[str] = []
    if not re.fullmatch(rf"(?:{release}|{standard})", title):
        errors.append(
            "title must be [Type]: Summary beginning with a capital letter; "
            "Release uses [Release]: vX.Y.Z or [Release]: vX.Y.Z+N"
        )
    else:
        summary = title.split(":", 1)[1].strip()
        errors.extend(sentence_case_summary(summary))
    return errors


def main_with_title(title: str) -> int:
    errors = title_errors(title)

    if errors:
        print("error: " + "; ".join(errors), file=sys.stderr)
        return 1
    print("Work item title is valid.")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--title", required=True)
    args = parser.parse_args()
    return main_with_title(args.title)


if __name__ == "__main__":
    raise SystemExit(main())
