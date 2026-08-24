"""Tests for the open Issue triage label validator."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


def load_script(path: str):
    path = Path(__file__).parents[1] / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


ISSUE_LABELS = load_script("scripts/validate/validate-issue-labels.py")

VALID = ["priority:medium", "scope:improvement", "phase:backlog"]


class IssueLabelTests(unittest.TestCase):
    def test_accepts_valid_label_set(self) -> None:
        self.assertEqual(ISSUE_LABELS.main_with_labels(VALID), 0)

    def test_accepts_other_allowed_values(self) -> None:
        self.assertEqual(
            ISSUE_LABELS.main_with_labels(
                ["priority:high", "scope:bug", "phase:planned"]
            ),
            0,
        )
        self.assertEqual(
            ISSUE_LABELS.main_with_labels(
                ["priority:low", "scope:release", "phase:in-progress"]
            ),
            0,
        )

    def test_rejects_missing_priority(self) -> None:
        self.assertEqual(
            ISSUE_LABELS.main_with_labels(["scope:improvement", "phase:backlog"]), 1
        )

    def test_rejects_missing_scope(self) -> None:
        self.assertEqual(
            ISSUE_LABELS.main_with_labels(["priority:medium", "phase:backlog"]), 1
        )

    def test_rejects_missing_phase(self) -> None:
        self.assertEqual(
            ISSUE_LABELS.main_with_labels(["priority:medium", "scope:improvement"]), 1
        )

    def test_rejects_duplicate_priority(self) -> None:
        labels = VALID + ["priority:high"]
        self.assertEqual(ISSUE_LABELS.main_with_labels(labels), 1)

    def test_rejects_duplicate_scope(self) -> None:
        labels = VALID + ["scope:feature"]
        self.assertEqual(ISSUE_LABELS.main_with_labels(labels), 1)

    def test_rejects_duplicate_phase(self) -> None:
        labels = VALID + ["phase:planned"]
        self.assertEqual(ISSUE_LABELS.main_with_labels(labels), 1)

    def test_rejects_unknown_label(self) -> None:
        labels = ["priority:medium", "scope:security", "phase:backlog"]
        self.assertEqual(ISSUE_LABELS.main_with_labels(labels), 1)

    def test_rejects_unrecognized_label_name(self) -> None:
        labels = VALID + ["team:core"]
        self.assertEqual(ISSUE_LABELS.main_with_labels(labels), 1)

    def test_rejects_empty_label_set(self) -> None:
        self.assertEqual(ISSUE_LABELS.main_with_labels([]), 1)


if __name__ == "__main__":
    unittest.main()
