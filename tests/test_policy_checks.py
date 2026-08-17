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


if __name__ == "__main__":
    unittest.main()
