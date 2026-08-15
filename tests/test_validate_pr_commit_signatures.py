"""Tests for the pull-request commit signature verifier."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


def load_verifier():
    path = (
        Path(__file__).parents[1] / "scripts/validate/validate-pr-commit-signatures.py"
    )
    spec = importlib.util.spec_from_file_location("signature_verifier", path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


VERIFIER = load_verifier()


class InvalidCommitsTests(unittest.TestCase):
    def test_accepts_only_github_verified_commits(self) -> None:
        commits = [
            {"sha": "verified", "commit": {"verification": {"verified": True}}},
            {
                "sha": "unsigned",
                "commit": {"verification": {"verified": False, "reason": "unsigned"}},
            },
            {"sha": "missing", "commit": {}},
        ]

        self.assertEqual(
            VERIFIER.invalid_commits(commits),
            ["unsigned: unsigned", "missing: missing verification"],
        )


if __name__ == "__main__":
    unittest.main()
