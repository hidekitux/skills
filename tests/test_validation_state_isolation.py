"""Regression tests for validation caches outside the repository tree."""

from __future__ import annotations

import unittest
from pathlib import Path

import tomllib

ROOT = Path(__file__).parents[1]
CACHE_ROOT = "${RUNNER_TEMP:-${TMPDIR:-/tmp}}"


class ValidationStateIsolationTests(unittest.TestCase):
    def test_validation_tasks_use_temporary_cache_roots(self) -> None:
        config = tomllib.loads((ROOT / "mise.toml").read_text(encoding="utf-8"))
        tasks = config["tasks"]
        for name in (
            "lint:python",
            "check:repository",
            "validate-skill-creator",
            "verify-release",
        ):
            run = tasks[name]["run"]
            self.assertIn(CACHE_ROOT, run, name)
            self.assertNotIn("$PWD/.mise", run, name)

    def test_fsl_scripts_use_temporary_cache_root(self) -> None:
        for name in ("install-fslc.sh", "verify-fsl.sh", "mutate-fsl.sh"):
            source = (ROOT / "scripts/fsl" / name).read_text(encoding="utf-8")
            self.assertIn(CACHE_ROOT, source, name)
            self.assertNotIn("$PWD/.mise", source, name)


if __name__ == "__main__":
    unittest.main()
