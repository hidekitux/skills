"""Tests for the idempotent local Git-hooks and skills setup path."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import tomllib

ROOT = Path(__file__).parents[1]


class LocalSetupTests(unittest.TestCase):
    def test_setup_task_groups_the_local_prerequisites(self) -> None:
        config = tomllib.loads((ROOT / "mise.toml").read_text(encoding="utf-8"))
        self.assertEqual(
            config["tasks"]["setup"]["depends"],
            ["setup:local-skills", "setup:commitlint"],
        )
        self.assertNotIn("setup-local-skills", config["tasks"])
        self.assertNotIn("setup-commitlint", config["tasks"])

    def test_task_groups_follow_distinct_workflows(self) -> None:
        tasks = tomllib.loads((ROOT / "mise.toml").read_text(encoding="utf-8"))["tasks"]
        self.assertEqual(
            tasks["check:local"]["depends"],
            ["check:repository", "check:branch-policy", "check:diff"],
        )
        self.assertEqual(
            tasks["lint"]["depends"],
            ["lint:actions", "lint:python", "lint:shell"],
        )

    def test_setup_local_skills_is_idempotent_and_configures_hooks(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            clone = Path(temporary_directory) / "skills"
            shutil.copytree(
                ROOT,
                clone,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            subprocess.run(["git", "init", "--quiet", str(clone)], check=True)

            for _ in range(2):
                subprocess.run(
                    ["bash", "scripts/setup-local-skills.sh"],
                    cwd=clone,
                    check=True,
                    capture_output=True,
                    text=True,
                )

            hook_path = subprocess.run(
                ["git", "config", "--get", "core.hooksPath"],
                cwd=clone,
                check=True,
                capture_output=True,
                text=True,
            ).stdout.strip()
            self.assertEqual(hook_path, ".githooks")
            self.assertTrue(
                (clone / ".agents" / "skills" / "create-issue").is_symlink()
            )
            self.assertTrue(
                (clone / ".claude" / "skills" / "create-issue").is_symlink()
            )


if __name__ == "__main__":
    unittest.main()
