"""Tests for deterministic Issue and Pull Request body structure."""

from __future__ import annotations

import contextlib
import importlib.util
import io
import os
import sys
import unittest
from pathlib import Path
from unittest import mock


def load_script(path: str):
    path = Path(__file__).parents[1] / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


BRANCH_POLICY = load_script("scripts/validate/validate-branch-policy.py")
ISSUE_BODY = load_script("scripts/validate/validate-issue-body.py")
WORK_ITEM_TITLE = load_script("scripts/validate/validate-work-item-title.py")

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

    def test_accepts_dependabot_route_without_issue_link(self) -> None:
        with (
            mock.patch.object(
                sys,
                "argv",
                [
                    "validate-branch-policy.py",
                    "--config",
                    ".github/branch-policy.toml",
                    "--base",
                    "main",
                    "--head",
                    "dependabot/example",
                    "--body",
                    "",
                ],
            ),
            contextlib.redirect_stdout(io.StringIO()) as stdout,
        ):
            self.assertEqual(BRANCH_POLICY.main(), 0)
        self.assertIn("Allowed pull-request direction", stdout.getvalue())

    def test_rejects_issue_branch_without_matching_link(self) -> None:
        with (
            mock.patch.object(
                sys,
                "argv",
                [
                    "validate-branch-policy.py",
                    "--config",
                    ".github/branch-policy.toml",
                    "--base",
                    "main",
                    "--head",
                    "issue/123",
                    "--body",
                    "",
                ],
            ),
            mock.patch.dict(os.environ, {}, clear=True),
            contextlib.redirect_stderr(io.StringIO()) as stderr,
        ):
            self.assertEqual(BRANCH_POLICY.main(), 1)
        self.assertIn("disallowed pull-request direction", stderr.getvalue())


class WorkItemTitleTests(unittest.TestCase):
    def test_accepts_sentence_case_with_proper_nouns(self) -> None:
        title = "[Feature]: Add GitHub Actions login flow"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 0)

    def test_accepts_acronyms_in_summary(self) -> None:
        title = "[Improvement]: Add CI validation for YAML workflows"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 0)

    def test_accepts_build_identifier_release(self) -> None:
        title = "[Release]: v1.2.3+42"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 0)

    def test_rejects_lowercase_summary_start(self) -> None:
        title = "[Feature]: add user login flow"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)

    def test_rejects_unknown_type(self) -> None:
        title = "[Enhancement]: Add user login flow"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)

    def test_rejects_title_case_summary(self) -> None:
        title = "[Feature]: Add User Login Flow"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)

    def test_accepts_concrete_summary_with_template_verb(self) -> None:
        title = "[Feature]: Add analyze-project skill"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 0)

    def test_rejects_empty_summary(self) -> None:
        for title in ("[Feature]:", "[Feature]: ", "[Feature]:    "):
            with self.subTest(title=title):
                self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)

    def test_rejects_bare_template_placeholder_summary(self) -> None:
        for title in ("[Feature]: Add", "[Feature]: Add ", "[Feature]: Add\t"):
            with self.subTest(title=title):
                self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)

    def test_accepts_dependabot_title_with_author(self) -> None:
        title = "Bump actions/checkout from 4 to 5"
        self.assertEqual(
            WORK_ITEM_TITLE.main_with_title(title, author="dependabot[bot]"), 0
        )

    def test_rejects_bot_style_title_without_dependabot_author(self) -> None:
        title = "Bump actions/checkout from 4 to 5"
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title), 1)
        self.assertEqual(WORK_ITEM_TITLE.main_with_title(title, author="octocat"), 1)

    def test_main_reads_dependabot_context_from_environment(self) -> None:
        with (
            mock.patch.dict(os.environ, {"PR_AUTHOR": "dependabot[bot]"}),
            mock.patch.object(
                sys,
                "argv",
                [
                    "validate-work-item-title.py",
                    "--title",
                    "Bump actions/checkout from 4 to 5",
                ],
            ),
            contextlib.redirect_stdout(io.StringIO()) as stdout,
        ):
            self.assertEqual(WORK_ITEM_TITLE.main(), 0)
        self.assertIn(
            "Dependabot pull request title exemption applies.", stdout.getvalue()
        )

    def test_main_rejects_human_author_title_with_dependabot_head_ref(self) -> None:
        with (
            mock.patch.dict(
                os.environ,
                {
                    "PR_HEAD_REF": "dependabot/example",
                    "PR_AUTHOR": "octocat",
                },
            ),
            mock.patch.object(
                sys,
                "argv",
                [
                    "validate-work-item-title.py",
                    "--title",
                    "Bump actions/checkout from 4 to 5",
                ],
            ),
            contextlib.redirect_stderr(io.StringIO()) as stderr,
        ):
            self.assertEqual(WORK_ITEM_TITLE.main(), 1)
        self.assertIn("[Type]: Summary", stderr.getvalue())


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
