"""Pure-function tests for the commit-message linting script.

These tests exercise scripts/lint/lint-commits.py through an injected
subprocess runner so they never create a Git repository or a commit.
"""

from __future__ import annotations

import contextlib
import importlib.util
import io
import os
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

ROOT = Path(__file__).parents[1]


def load_script(path: str):
    path = ROOT / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


LINT_COMMITS = load_script("scripts/lint/lint-commits.py")


class FakeRunner:
    """Scripted stand-in for subprocess that records Git and commitlint calls."""

    def __init__(
        self,
        *,
        root="repo-root",
        rev_list_stdout="",
        show_outputs=(),
        commitlint_returncode=0,
        author_outputs=(),
    ):
        self.root = root
        self.rev_list_stdout = rev_list_stdout
        self.show_outputs = list(show_outputs)
        self.commitlint_returncode = commitlint_returncode
        self.author_outputs = list(author_outputs)
        self.calls: list[tuple[list[str], str | None]] = []

    def check_output(self, argv, **kwargs):
        self.calls.append((list(argv), None))
        if argv[:2] == ["git", "rev-parse"]:
            return self.root
        if argv[:2] == ["git", "show"]:
            if "--format=%an" in argv:
                return self.author_outputs.pop(0)
            return self.show_outputs.pop(0)
        raise AssertionError(f"unexpected check_output: {argv}")

    def run(self, argv, *, input=None, **kwargs):
        self.calls.append((list(argv), input))
        if argv[:2] == ["git", "rev-list"]:
            return SimpleNamespace(stdout=self.rev_list_stdout)
        if len(argv) == 2 and argv[1] == "lint":
            return SimpleNamespace(returncode=self.commitlint_returncode)
        raise AssertionError(f"unexpected run: {argv}")


def no_signature_prompt_environment():
    """A CM clearing the environment so no signing config leaks into tests."""
    return mock.patch.dict(os.environ, {}, clear=True)


class CommitLintingTests(unittest.TestCase):
    def test_resolve_commitlint_uses_environment_override(self) -> None:
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(os.environ, {"COMMITLINT_BIN": "/custom/bin/commitlint"}),
        ):
            self.assertEqual(
                LINT_COMMITS.resolve_commitlint("repo-root", FakeRunner()),
                "/custom/bin/commitlint",
            )

    def test_resolve_commitlint_uses_project_local_binary(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            binary = Path(temporary_directory) / ".mise" / "bin" / "commitlint"
            binary.parent.mkdir(parents=True)
            binary.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
            binary.chmod(0o755)
            with no_signature_prompt_environment():
                self.assertEqual(
                    LINT_COMMITS.resolve_commitlint(temporary_directory, FakeRunner()),
                    str(binary),
                )

    def test_resolve_commitlint_exits_2_with_guidance_when_missing(self) -> None:
        with (
            tempfile.TemporaryDirectory() as temporary_directory,
            no_signature_prompt_environment(),
            contextlib.redirect_stderr(io.StringIO()) as stderr,
            self.assertRaises(SystemExit) as raised,
        ):
            LINT_COMMITS.resolve_commitlint(temporary_directory, FakeRunner())
        self.assertEqual(raised.exception.code, 2)
        self.assertIn("commitlint is not installed", stderr.getvalue())

    def test_collect_commits_runs_rev_list_without_merges_range(self) -> None:
        runner = FakeRunner(rev_list_stdout="abc123\ndef456\n")
        self.assertEqual(
            LINT_COMMITS.collect_commits("base", "head", runner),
            ["abc123", "def456"],
        )
        self.assertIn(
            (["git", "rev-list", "--no-merges", "base..head"], None),
            runner.calls,
        )

    def test_collect_commits_supports_single_side_after(self) -> None:
        runner = FakeRunner(rev_list_stdout="abc123\n")
        self.assertEqual(LINT_COMMITS.collect_commits(None, "HEAD", runner), ["abc123"])
        self.assertIn(
            (["git", "rev-list", "--no-merges", "HEAD"], None),
            runner.calls,
        )

    def test_lint_message_reports_commitlint_failure(self) -> None:
        runner = FakeRunner(commitlint_returncode=1)
        status = LINT_COMMITS.lint_message(
            "feat: bad header #1\n", "/bin/commitlint", runner
        )
        self.assertEqual(status, 1)
        self.assertEqual(
            runner.calls, [(["/bin/commitlint", "lint"], "feat: bad header #1\n")]
        )

    def test_lint_message_reports_validator_failure(self) -> None:
        runner = FakeRunner(commitlint_returncode=0)
        with contextlib.redirect_stderr(io.StringIO()) as stderr:
            status = LINT_COMMITS.lint_message(
                "feat: no issue number\n", "/bin/commitlint", runner
            )
        self.assertEqual(status, 1)
        self.assertIn("numeric issue number", stderr.getvalue())

    def test_lint_message_passes_a_valid_message(self) -> None:
        runner = FakeRunner(commitlint_returncode=0)
        self.assertEqual(
            LINT_COMMITS.lint_message(
                "feat: add feature #1\n", "/bin/commitlint", runner
            ),
            0,
        )

    def test_main_lints_pr_range_and_exits_zero(self) -> None:
        runner = FakeRunner(
            rev_list_stdout="abc123\n",
            author_outputs=("Jane Doe",),
            show_outputs=("feat: add feature #1\n\n",),
            commitlint_returncode=0,
        )
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "COMMITLINT_BIN": "/bin/commitlint",
                    "PR_BASE_SHA": "base",
                    "PR_HEAD_SHA": "head",
                },
            ),
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn(
            (["git", "rev-list", "--no-merges", "base..head"], None),
            runner.calls,
        )
        self.assertIn(
            (["git", "show", "-s", "--format=%B", "abc123"], None),
            runner.calls,
        )

    def test_main_exits_one_when_issue_number_is_missing(self) -> None:
        runner = FakeRunner(
            rev_list_stdout="abc123\n",
            author_outputs=("Jane Doe",),
            show_outputs=("feat: change readme\n",),
            commitlint_returncode=0,
        )
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "COMMITLINT_BIN": "/bin/commitlint",
                    "PR_BASE_SHA": "base",
                    "PR_HEAD_SHA": "head",
                },
            ),
            contextlib.redirect_stderr(io.StringIO()) as stderr,
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 1)
        self.assertIn("numeric issue number", stderr.getvalue())

    def test_main_reports_when_there_are_no_commits(self) -> None:
        runner = FakeRunner(rev_list_stdout="")
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(os.environ, {"COMMITLINT_BIN": "/bin/commitlint"}),
            contextlib.redirect_stdout(io.StringIO()) as stdout,
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn("No commits to lint.", stdout.getvalue())
        self.assertIn((["git", "rev-list", "--no-merges", "HEAD"], None), runner.calls)

    def test_main_prefers_push_before_sha_when_set(self) -> None:
        runner = FakeRunner(
            rev_list_stdout="abc123\n",
            author_outputs=("Jane Doe",),
            show_outputs=("feat: add feature #1\n\n",),
            commitlint_returncode=0,
        )
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "COMMITLINT_BIN": "/bin/commitlint",
                    "PUSH_BEFORE_SHA": "beef",
                    "GITHUB_SHA": "cafe",
                },
            ),
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn(
            (["git", "rev-list", "--no-merges", "beef..cafe"], None),
            runner.calls,
        )

    def test_main_treats_all_zero_push_before_as_single_side(self) -> None:
        runner = FakeRunner(rev_list_stdout="")
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "COMMITLINT_BIN": "/bin/commitlint",
                    "PUSH_BEFORE_SHA": "0000",
                    "GITHUB_SHA": "cafe",
                },
            ),
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn((["git", "rev-list", "--no-merges", "cafe"], None), runner.calls)

    def test_main_exits_2_when_commitlint_is_missing(self) -> None:
        runner = FakeRunner()
        with (
            no_signature_prompt_environment(),
            self.assertRaises(SystemExit) as raised,
        ):
            LINT_COMMITS.main(runner)
        self.assertEqual(raised.exception.code, 2)

    def test_is_dependabot_pull_request_matches_head_ref(self) -> None:
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {"PR_HEAD_REF": "dependabot/github_actions/actions-checkout"},
            ),
        ):
            self.assertTrue(LINT_COMMITS.is_dependabot_pull_request())

    def test_is_dependabot_pull_request_matches_author(self) -> None:
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(os.environ, {"PR_AUTHOR": "dependabot[bot]"}),
        ):
            self.assertTrue(LINT_COMMITS.is_dependabot_pull_request())

    def test_is_dependabot_pull_request_false_for_human_context(self) -> None:
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ, {"PR_HEAD_REF": "issue/123", "PR_AUTHOR": "octocat"}
            ),
        ):
            self.assertFalse(LINT_COMMITS.is_dependabot_pull_request())

    def test_main_skips_linting_for_dependabot_pull_request(self) -> None:
        runner = FakeRunner()
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "PR_HEAD_REF": "dependabot/example",
                    "PR_AUTHOR": "dependabot[bot]",
                },
            ),
            contextlib.redirect_stdout(io.StringIO()) as stdout,
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn(
            "Skipping commit lint for a Dependabot pull request.", stdout.getvalue()
        )
        self.assertEqual(
            runner.calls, [(["git", "rev-parse", "--show-toplevel"], None)]
        )

    def test_main_skips_dependabot_authored_commits_in_push_range(self) -> None:
        runner = FakeRunner(
            rev_list_stdout="deps123\nhuman456\n",
            author_outputs=("dependabot[bot]", "Jane Doe"),
            show_outputs=("feat: add feature #1\n\n",),
            commitlint_returncode=0,
        )
        with (
            no_signature_prompt_environment(),
            mock.patch.dict(
                os.environ,
                {
                    "COMMITLINT_BIN": "/bin/commitlint",
                    "PUSH_BEFORE_SHA": "beef",
                    "GITHUB_SHA": "cafe",
                },
            ),
            contextlib.redirect_stdout(io.StringIO()) as stdout,
        ):
            self.assertEqual(LINT_COMMITS.main(runner), 0)
        self.assertIn("Checking commit deps123", stdout.getvalue())
        self.assertIn("Skipping Dependabot-authored commit deps123", stdout.getvalue())
        self.assertIn("Checking commit human456", stdout.getvalue())
        self.assertIn(
            (["git", "show", "-s", "--format=%an", "deps123"], None), runner.calls
        )
        self.assertIn(
            (["git", "show", "-s", "--format=%an", "human456"], None), runner.calls
        )
        self.assertIn(
            (["git", "show", "-s", "--format=%B", "human456"], None), runner.calls
        )
        self.assertNotIn(
            (["git", "show", "-s", "--format=%B", "deps123"], None), runner.calls
        )


if __name__ == "__main__":
    unittest.main()
