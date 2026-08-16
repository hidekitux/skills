"""Tests for the idempotent local Git-hooks and skills setup path."""

from __future__ import annotations

import importlib.util
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import tomllib

ROOT = Path(__file__).parents[1]


def load_script(path: str):
    path = ROOT / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


COMMIT_MESSAGE = load_script("scripts/lint/validate-commit-message.py")


def isolated_git_environment() -> dict[str, str]:
    """Prevent Git hooks from redirecting temporary repositories to this one."""
    return {
        key: value for key, value in os.environ.items() if not key.startswith("GIT_")
    }


class LocalSetupTests(unittest.TestCase):
    def test_setup_task_groups_the_local_prerequisites(self) -> None:
        config = tomllib.loads((ROOT / "mise.toml").read_text(encoding="utf-8"))
        self.assertEqual(
            config["tasks"]["setup"]["depends"],
            ["setup:local-skills", "setup:commitlint"],
        )
        self.assertNotIn("setup-local-skills", config["tasks"])
        self.assertNotIn("setup-commitlint", config["tasks"])

    def test_commitlint_has_no_trailer_exemption(self) -> None:
        config = (ROOT / ".commitlint.yaml").read_text(encoding="utf-8")
        self.assertNotIn("trailer-exists", config)

    def test_commit_message_requires_single_line_and_issue_number(self) -> None:
        self.assertEqual(COMMIT_MESSAGE.validate("feat: add login flow #61"), [])
        self.assertEqual(COMMIT_MESSAGE.validate("feat(scope): add login flow #61"), [])
        self.assertNotEqual(COMMIT_MESSAGE.validate("feat: add login flow"), [])
        self.assertNotEqual(
            COMMIT_MESSAGE.validate("feat: add login flow #61\n\nbody"), []
        )

    def test_lint_commits_requires_issue_number_suffix(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            shutil.copytree(
                ROOT / "scripts",
                root / "scripts",
                ignore=shutil.ignore_patterns("__pycache__"),
            )
            environment = isolated_git_environment()
            subprocess.run(
                ["git", "init", "--initial-branch", "main", root],
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=root,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=root,
                check=True,
                env=environment,
            )
            (root / "README").write_text("base\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=root, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "chore: add base"],
                cwd=root,
                check=True,
                env=environment,
            )
            (root / "README").write_text("feature\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=root, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "feat: add feature #1"],
                cwd=root,
                check=True,
                env=environment,
            )
            result = subprocess.run(
                [
                    "bash",
                    str(ROOT / "scripts" / "lint" / "lint-commits.sh"),
                ],
                cwd=root,
                check=False,
                capture_output=True,
                text=True,
                env={
                    **environment,
                    "COMMITLINT_BIN": "true",
                    "PR_BASE_SHA": "HEAD~1",
                    "PR_HEAD_SHA": "HEAD",
                },
            )
            self.assertEqual(result.returncode, 0)

            (root / "README").write_text("change\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=root, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "feat: change readme"],
                cwd=root,
                check=True,
                env=environment,
            )
            result = subprocess.run(
                [
                    "bash",
                    str(ROOT / "scripts" / "lint" / "lint-commits.sh"),
                ],
                cwd=root,
                check=False,
                capture_output=True,
                text=True,
                env={
                    **environment,
                    "COMMITLINT_BIN": "true",
                    "PR_BASE_SHA": "HEAD~1",
                    "PR_HEAD_SHA": "HEAD",
                },
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("numeric issue number", result.stderr)

    def test_commit_msg_hook_rejects_missing_issue_number(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            message_file = Path(temporary_directory) / "COMMIT_EDITMSG"
            message_file.write_text("feat: change readme\n", encoding="utf-8")
            result = subprocess.run(
                [str(ROOT / ".githooks" / "commit-msg"), str(message_file)],
                cwd=ROOT,
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 1)
            self.assertIn("numeric issue number", result.stderr)

            message_file.write_text("feat: change readme #61\n", encoding="utf-8")
            result = subprocess.run(
                [str(ROOT / ".githooks" / "commit-msg"), str(message_file)],
                cwd=ROOT,
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0)

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
                    ["bash", "scripts/setup/setup-local-skills.sh"],
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

    def test_post_checkout_runs_setup_for_branch_checkouts(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=0)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "post-checkout"), "old", "new", "1"],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0)
            self.assertEqual(log.read_text(encoding="utf-8"), "run setup\n")

    def test_post_checkout_skips_setup_for_file_checkouts(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=0)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "post-checkout"), "old", "new", "0"],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0)
            self.assertFalse(log.exists())

    def test_post_checkout_warns_without_blocking_when_setup_fails(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=1)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "post-checkout"), "old", "new", "1"],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0)
            self.assertEqual(log.read_text(encoding="utf-8"), "run setup\n")
            self.assertIn("Warning: Local setup did not complete.", result.stderr)

    def test_pre_commit_runs_local_checks(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=0)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "pre-commit")],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0)
            self.assertEqual(log.read_text(encoding="utf-8"), "run check:local\n")

    def test_pre_push_runs_full_validation(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=0)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "pre-push")],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0)
            self.assertEqual(log.read_text(encoding="utf-8"), "run validate\n")

    def test_pre_commit_blocks_when_local_checks_fail(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=1)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "pre-commit")],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 1)
            self.assertEqual(log.read_text(encoding="utf-8"), "run check:local\n")

    def test_pre_push_blocks_when_validation_fails(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            log = root / "mise.log"
            environment = self.mise_environment(root, log, exit_code=1)

            result = subprocess.run(
                [str(ROOT / ".githooks" / "pre-push")],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 1)
            self.assertEqual(log.read_text(encoding="utf-8"), "run validate\n")

    def mise_environment(self, root: Path, log: Path, exit_code: int) -> dict[str, str]:
        bin_directory = root / "bin"
        bin_directory.mkdir()
        mise = bin_directory / "mise"
        mise.write_text(
            '#!/bin/sh\nprintf \'%s\\n\' "$*" >> "$SETUP_LOG"\nexit "$MISE_EXIT_CODE"\n',
            encoding="utf-8",
        )
        mise.chmod(0o755)
        return {
            **os.environ,
            "MISE_EXIT_CODE": str(exit_code),
            "PATH": f"{bin_directory}:{os.environ['PATH']}",
            "SETUP_LOG": str(log),
        }


if __name__ == "__main__":
    unittest.main()
