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

    def test_lint_commits_uses_project_local_commitlint_when_configured(self) -> None:
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
            commitlint = root / ".mise" / "bin" / "commitlint"
            commitlint.parent.mkdir(parents=True)
            commitlint.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
            commitlint.chmod(0o755)
            (root / "README").write_text("base\n", encoding="utf-8")
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
                    "PR_BASE_SHA": "HEAD~1",
                    "PR_HEAD_SHA": "HEAD",
                },
            )
            self.assertEqual(result.returncode, 0)

    def test_lint_commits_errors_when_project_commitlint_is_missing(self) -> None:
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
                    "PR_BASE_SHA": "HEAD~1",
                    "PR_HEAD_SHA": "HEAD",
                },
            )
            self.assertEqual(result.returncode, 2)
            self.assertIn("commitlint is not installed", result.stderr)

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
            environment = isolated_git_environment()
            shutil.copytree(
                ROOT,
                clone,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            subprocess.run(
                ["git", "init", "--quiet", str(clone)], check=True, env=environment
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=clone,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=clone,
                check=True,
                env=environment,
            )
            (clone / "README").write_text("base\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=clone, check=True, env=environment
            )
            subprocess.run(
                [
                    "git",
                    "-c",
                    "core.hooksPath=/dev/null",
                    "-c",
                    "commit.gpgsign=false",
                    "commit",
                    "-qm",
                    "chore: add base #1",
                ],
                cwd=clone,
                check=True,
                env=environment,
            )

            for _ in range(2):
                subprocess.run(
                    ["bash", "scripts/setup/setup-local-skills.sh"],
                    cwd=clone,
                    check=True,
                    capture_output=True,
                    text=True,
                    env=environment,
                )

            hook_path = subprocess.run(
                ["git", "config", "--get", "core.hooksPath"],
                cwd=clone,
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            ).stdout.strip()
            self.assertEqual(hook_path, ".githooks")
            self.assertTrue(
                (clone / ".agents" / "skills" / "create-issue").is_symlink()
            )
            self.assertTrue(
                (clone / ".claude" / "skills" / "create-issue").is_symlink()
            )

    def test_register_local_skills_is_snapshot_dependent(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            clone = Path(temporary_directory) / "skills"
            shutil.copytree(
                ROOT,
                clone,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            environment = isolated_git_environment()
            subprocess.run(
                ["git", "init", "--quiet", "--initial-branch", "main", str(clone)],
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=clone,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=clone,
                check=True,
                env=environment,
            )
            (clone / "README").write_text("base\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=clone, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "chore: add base"],
                cwd=clone,
                check=True,
                env=environment,
            )
            first = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                cwd=clone,
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            ).stdout.strip()

            result = subprocess.run(
                ["bash", "scripts/setup/register-local-skills.sh"],
                cwd=clone,
                check=False,
                capture_output=True,
                text=True,
                env=environment,
            )
            self.assertEqual(result.returncode, 0)
            self.assertIn(first, result.stdout)
            stamp = clone / ".agents" / "worktree-snapshot"
            self.assertEqual(stamp.read_text(encoding="ascii").strip(), first)

            (clone / "README").write_text("feature\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=clone, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "feat: add feature #1"],
                cwd=clone,
                check=True,
                env=environment,
            )
            second = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                cwd=clone,
                check=True,
                capture_output=True,
                text=True,
                env=environment,
            ).stdout.strip()

            result = subprocess.run(
                ["bash", "scripts/setup/register-local-skills.sh"],
                cwd=clone,
                check=False,
                capture_output=True,
                text=True,
                env=environment,
            )
            self.assertEqual(result.returncode, 0)
            self.assertIn(second, result.stdout)
            self.assertEqual(stamp.read_text(encoding="ascii").strip(), second)

    def test_setup_commitlint_reuses_the_shared_binary(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / "repos"
            main = root / "main"
            shutil.copytree(
                ROOT,
                main,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            environment = isolated_git_environment()
            subprocess.run(
                ["git", "init", "--quiet", "--initial-branch", "main", str(main)],
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=main,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=main,
                check=True,
                env=environment,
            )
            (main / "README").write_text("base\n", encoding="utf-8")
            subprocess.run(["git", "add", "-A"], cwd=main, check=True, env=environment)
            subprocess.run(
                ["git", "commit", "-m", "chore: add base"],
                cwd=main,
                check=True,
                env=environment,
            )
            worktree = root / "feature"
            subprocess.run(
                [
                    "git",
                    "-C",
                    str(main),
                    "worktree",
                    "add",
                    "-b",
                    "issue/1",
                    str(worktree),
                ],
                check=True,
                env=environment,
            )
            shared_dir = main / ".git" / ".mise" / "bin"
            shared_dir.mkdir(parents=True)
            (shared_dir / "commitlint").write_bytes(b"#!/bin/sh\nexit 0\n")
            (shared_dir / "commitlint").chmod(0o755)

            for _ in range(2):
                result = subprocess.run(
                    ["bash", "scripts/setup/setup-commitlint.sh"],
                    cwd=worktree,
                    check=False,
                    capture_output=True,
                    text=True,
                    env=environment,
                )
                self.assertEqual(result.returncode, 0)
                self.assertIn("Using shared commitlint", result.stdout)
            self.assertTrue(
                worktree.joinpath(".mise", "bin", "commitlint").is_symlink()
            )
            self.assertEqual(
                Path(
                    worktree.joinpath(".mise", "bin", "commitlint").readlink()
                ).resolve(),
                (shared_dir / "commitlint").resolve(),
            )

    def test_setup_commitlint_links_shared_binary_from_primary_worktree(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            main = Path(temporary_directory) / "main"
            shutil.copytree(
                ROOT,
                main,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            environment = isolated_git_environment()
            subprocess.run(
                ["git", "init", "--quiet", "--initial-branch", "main", str(main)],
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=main,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=main,
                check=True,
                env=environment,
            )
            shared_dir = main / ".git" / ".mise" / "bin"
            shared_dir.mkdir(parents=True)
            (shared_dir / "commitlint").write_bytes(b"#!/bin/sh\nexit 0\n")
            (shared_dir / "commitlint").chmod(0o755)

            result = subprocess.run(
                ["bash", "scripts/setup/setup-commitlint.sh"],
                cwd=main,
                check=False,
                capture_output=True,
                text=True,
                env=environment,
            )
            self.assertEqual(result.returncode, 0)
            self.assertIn("Using shared commitlint", result.stdout)
            link = main / ".mise" / "bin" / "commitlint"
            self.assertTrue(link.is_symlink())
            self.assertEqual(link.resolve(), (shared_dir / "commitlint").resolve())
            self.assertTrue(os.access(link, os.X_OK))

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

    def test_post_checkout_records_setup_result(self) -> None:
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
            self.assertIn("Local setup complete", result.stdout)

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

    def test_worktree_diagnostic_reports_owner_and_setup_state(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            clone = root / "clone"
            shutil.copytree(
                ROOT,
                clone,
                ignore=shutil.ignore_patterns(".git", ".agents", ".claude", ".mise"),
            )
            environment = isolated_git_environment()
            subprocess.run(
                ["git", "init", "--quiet", "--initial-branch", "main", str(clone)],
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.email", "test" + "@" + "example.invalid"],
                cwd=clone,
                check=True,
                env=environment,
            )
            subprocess.run(
                ["git", "config", "user.name", "Test User"],
                cwd=clone,
                check=True,
                env=environment,
            )
            (clone / "README").write_text("base\n", encoding="utf-8")
            subprocess.run(
                ["git", "add", "README"], cwd=clone, check=True, env=environment
            )
            subprocess.run(
                ["git", "commit", "-m", "chore: add base"],
                cwd=clone,
                check=True,
                env=environment,
            )

            result = subprocess.run(
                [
                    "python",
                    str(ROOT / "scripts" / "diagnose" / "diagnose-worktree.py"),
                    "--branch",
                    "main",
                    "--base",
                    "main",
                ],
                cwd=clone,
                check=False,
                capture_output=True,
                text=True,
                env=environment,
            )

            self.assertEqual(result.returncode, 0)
            self.assertIn(str(clone), result.stdout)
            self.assertIn("setup", result.stdout)
            self.assertIn("git worktree add --detach", result.stdout)
            self.assertIn("git worktree add -b issue/<number>", result.stdout)
            self.assertTrue(clone.is_dir())

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
