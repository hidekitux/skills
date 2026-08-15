"""Tests for deterministic Issue and Pull Request body structure."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


def load_script(name: str):
    path = Path(__file__).parents[1] / "scripts" / name
    spec = importlib.util.spec_from_file_location(name.removesuffix(".py"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


BRANCH_POLICY = load_script("validate-branch-policy.py")
ISSUE_BODY = load_script("validate-issue-body.py")

CHANGE_BODY = """## Context

The generated work-item structure is ambiguous.

## Goal

Make generated bodies deterministic.

## Scope

- In:
  - Body ordering.
- Out:
  - Prose linting.

## Acceptance criteria

- [ ] Required sections are ordered.

## Validation

- [ ] Run the body validator tests.
"""

RELEASE_BODY = """## Context

The verified changes are ready to publish.

## Goal

Publish version v1.2.3.

## Scope

- In:
  - Version v1.2.3.
- Out:
  - Later changes.

## Acceptance criteria

- [ ] The release is published.

## Validation

- [ ] Run the release verification task.

## Changelog

### Added

- Deterministic body validation.

### Changed

- None.

### Fixed

- None.

### Removed

- None.
"""


class PullRequestBodyTests(unittest.TestCase):
    def test_accepts_matching_closes_block_as_first_section(self) -> None:
        body = """## Issue

Closes #35
Closes #36

## Summary

- Standardize generated bodies.
"""
        self.assertEqual(BRANCH_POLICY.issue_links_at_start(body), [35, 36])
        self.assertTrue(BRANCH_POLICY.matching_issue_link_starts_body(body, 35))

    def test_rejects_branch_issue_when_it_is_not_the_first_reference(self) -> None:
        body = """## Issue

Closes #36
Closes #35

## Summary

- Standardize generated bodies.
"""
        self.assertFalse(BRANCH_POLICY.matching_issue_link_starts_body(body, 35))

    def test_rejects_additional_closes_outside_issue_section(self) -> None:
        body = """## Issue

Closes #35

## Summary

- Standardize generated bodies.

Closes #36
"""
        self.assertFalse(BRANCH_POLICY.matching_issue_link_starts_body(body, 35))

    def test_rejects_closes_line_at_the_end(self) -> None:
        body = """## Summary

- Standardize generated bodies.

Closes #35
"""
        self.assertIsNone(BRANCH_POLICY.issue_links_at_start(body))

    def test_rejects_non_reference_content_in_issue_section(self) -> None:
        body = """## Issue

This work closes the linked Issue.
Closes #35

## Summary

- Standardize generated bodies.
"""
        self.assertIsNone(BRANCH_POLICY.issue_links_at_start(body))

    def test_accepts_release_tracks_block_as_first_section(self) -> None:
        body = """## Issue

Tracks #35

## Summary

- Prepare the verified release.
"""
        self.assertEqual(BRANCH_POLICY.issue_references_at_start(body, "Tracks"), [35])


class IssueBodyTests(unittest.TestCase):
    def test_accepts_ordered_change_body(self) -> None:
        self.assertEqual(
            ISSUE_BODY.validation_errors(
                "[Improvement]: Standardize bodies", CHANGE_BODY
            ),
            [],
        )

    def test_rejects_reordered_or_duplicate_change_headings(self) -> None:
        reordered = (
            CHANGE_BODY.replace("## Context", "## Temporary", 1)
            .replace("## Goal", "## Context", 1)
            .replace("## Temporary", "## Goal", 1)
        )
        duplicate = CHANGE_BODY + "\n## Validation\n\n- [ ] Run it again.\n"
        self.assertTrue(
            ISSUE_BODY.validation_errors("[Improvement]: Standardize bodies", reordered)
        )
        self.assertTrue(
            ISSUE_BODY.validation_errors("[Improvement]: Standardize bodies", duplicate)
        )

    def test_rejects_empty_checklist_items(self) -> None:
        body = CHANGE_BODY.replace("- [ ] Required sections are ordered.", "- [ ]")
        self.assertIn(
            "Acceptance criteria must contain at least one non-empty checkbox",
            ISSUE_BODY.validation_errors("[Improvement]: Standardize bodies", body),
        )

    def test_rejects_empty_scope_markers(self) -> None:
        body = CHANGE_BODY.replace(
            "- In:\n  - Body ordering.\n- Out:\n  - Prose linting.",
            "- In:\n- Out:",
        )
        errors = ISSUE_BODY.validation_errors("[Improvement]: Standardize bodies", body)
        self.assertIn("Scope In must contain concrete content", errors)
        self.assertIn("Scope Out must contain concrete content", errors)

    def test_accepts_ordered_release_body(self) -> None:
        self.assertEqual(
            ISSUE_BODY.validation_errors("[Release]: v1.2.3", RELEASE_BODY), []
        )

    def test_rejects_reordered_release_changelog(self) -> None:
        body = (
            RELEASE_BODY.replace("### Added", "### Temporary", 1)
            .replace("### Changed", "### Added", 1)
            .replace("### Temporary", "### Changed", 1)
        )
        self.assertTrue(ISSUE_BODY.validation_errors("[Release]: v1.2.3", body))


if __name__ == "__main__":
    unittest.main()
