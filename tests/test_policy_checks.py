"""Tests for deterministic repository-policy checks."""

from __future__ import annotations

import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]


def load_script(path: str):
    path = ROOT / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


SENSITIVE = load_script("scripts/check/check-sensitive-content.py")
TOOL_LICENSES = load_script("scripts/check/check-tool-licenses.py")
SCRIPT_TESTS = load_script("scripts/validate/validate-script-tests.py")
MUTATION_BADGES = load_script("scripts/check/check-mutation-badges.py")
ANALYZE_READONLY = load_script("scripts/check/check-analyze-readonly.py")


ENDPOINT_BADGES = """\
![FSL mutants killed](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-killed.json)
![FSL kill rate](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-kill-rate.json)
![FSL surviving mutants](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-survived.json)
![FSL verifier](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffslc-version.json)
![Tests status](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-status.json)
![Tests run](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-run.json)
"""


class OpenCodeModelConfigTests(unittest.TestCase):
    def test_opencode_config_parses(self) -> None:
        config = json.loads((ROOT / "opencode.json").read_text(encoding="utf-8"))
        self.assertIsInstance(config, dict)

    def test_opencode_config_declares_every_model_tier(self) -> None:
        config = json.loads((ROOT / "opencode.json").read_text(encoding="utf-8"))
        for tier in ("high", "mid", "low"):
            with self.subTest(tier=tier):
                model = config["agent"][tier]["model"]
                self.assertIsInstance(model, str)
                self.assertTrue(model)


class PolicyCheckTests(unittest.TestCase):
    def test_sensitive_content_check_accepts_public_text(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "README.md").write_text(
                "See https://example.com/docs.\n", encoding="utf-8"
            )
            self.assertEqual(SENSITIVE.main_with_root(root), 0)

    def test_sensitive_content_check_rejects_a_token_and_private_url(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            token = "gh" + "p_" + "a" * 20
            private_url = "https://" + "192.168.1.1/private"
            (root / "notes.md").write_text(
                f"{token}\n{private_url}\n", encoding="utf-8"
            )
            self.assertEqual(SENSITIVE.main_with_root(root), 1)

    def test_tool_license_check_requires_every_mise_tool(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "mise.toml").write_text(
                '[tools]\npython = "3.11"\n', encoding="utf-8"
            )
            (root / "TOOL_LICENSES.toml").write_text("[tools]\n", encoding="utf-8")
            self.assertEqual(TOOL_LICENSES.main_with_root(root), 1)

    def test_script_test_mapping_requires_every_script(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts").mkdir()
            (root / "scripts" / "new.py").write_text("print('ok')\n", encoding="utf-8")
            (root / "SCRIPT_TESTS.toml").write_text("[scripts]\n", encoding="utf-8")
            self.assertEqual(SCRIPT_TESTS.main_with_root(root), 1)

    def test_script_test_mapping_requires_nested_script_coverage(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            nested = root / "scripts" / "validate"
            nested.mkdir(parents=True)
            (nested / "new.py").write_text("print('ok')\n", encoding="utf-8")
            (root / "SCRIPT_TESTS.toml").write_text("[scripts]\n", encoding="utf-8")
            self.assertEqual(SCRIPT_TESTS.main_with_root(root), 1)


class MutationBadgeTests(unittest.TestCase):
    def test_badge_check_accepts_six_endpoint_badges(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "README.md").write_text(ENDPOINT_BADGES, encoding="utf-8")
            self.assertEqual(MUTATION_BADGES.main_with_root(root), 0)

    def test_badge_check_rejects_a_static_fsl_badge(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            static = ENDPOINT_BADGES.replace(
                "https://img.shields.io/endpoint?url="
                "https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills"
                "%2Fbadge-data%2Ffsl-killed.json",
                "https://img.shields.io/badge/mutants%20killed-164%2F200-2ea44f",
            )
            (root / "README.md").write_text(static, encoding="utf-8")
            self.assertEqual(MUTATION_BADGES.main_with_root(root), 1)

    def test_badge_check_rejects_a_foreign_endpoint_payload(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            foreign = ENDPOINT_BADGES.replace(
                "%2Fbadge-data%2Ffsl-killed.json", "%2Fimages%2Ffsl-killed.json"
            )
            (root / "README.md").write_text(foreign, encoding="utf-8")
            self.assertEqual(MUTATION_BADGES.main_with_root(root), 1)

    def test_badge_check_rejects_a_missing_test_badge(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run_line = (
                "![Tests run](https://img.shields.io/endpoint?url="
                "https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills"
                "%2Fbadge-data%2Ftests-run.json)\n"
            )
            (root / "README.md").write_text(
                ENDPOINT_BADGES.replace(run_line, ""), encoding="utf-8"
            )
            self.assertEqual(MUTATION_BADGES.main_with_root(root), 1)


class AnalyzeReadonlyCheckTests(unittest.TestCase):
    def test_accepts_analyze_skill_without_creation_instructions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill_dir = root / "skills" / "analyze-project"
            skill_dir.mkdir(parents=True)
            (skill_dir / "SKILL.md").write_text(
                "---\nname: analyze-project\n---\n"
                "Inspect the code and report findings.\n",
                encoding="utf-8",
            )
            self.assertEqual(ANALYZE_READONLY.main_with_root(root), 0)

    def test_rejects_analyze_skill_instructing_issue_creation(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill_dir = root / "skills" / "analyze-project"
            skill_dir.mkdir(parents=True)
            (skill_dir / "SKILL.md").write_text(
                "---\nname: analyze-project\n---\n"
                "Create a GitHub issue to track the finding.\n",
                encoding="utf-8",
            )
            self.assertEqual(ANALYZE_READONLY.main_with_root(root), 1)

    def test_rejects_analyze_skill_instructing_pr_creation(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill_dir = root / "skills" / "analyze-baseline"
            skill_dir.mkdir(parents=True)
            (skill_dir / "SKILL.md").write_text(
                "---\nname: analyze-baseline\n---\n"
                "Open a pull request with the suggested change.\n",
                encoding="utf-8",
            )
            self.assertEqual(ANALYZE_READONLY.main_with_root(root), 1)

    def test_ignores_creation_instructions_in_non_analyze_skill(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill_dir = root / "skills" / "create-issue"
            skill_dir.mkdir(parents=True)
            (skill_dir / "SKILL.md").write_text(
                "---\nname: create-issue\n---\nCreate a GitHub issue for the work.\n",
                encoding="utf-8",
            )
            self.assertEqual(ANALYZE_READONLY.main_with_root(root), 0)


if __name__ == "__main__":
    unittest.main()
