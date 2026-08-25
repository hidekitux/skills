"""Tests for the release publication gate wrapper."""

from __future__ import annotations

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]

import importlib.util


def load_script(path: str):
    path = ROOT / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


VERIFY_RELEASE = load_script("scripts/release/verify-release.py")


class PublishReleaseTests(unittest.TestCase):
    def test_usage_requires_exactly_one_tag(self) -> None:
        result = subprocess.run(
            ["bash", "scripts/release/publish-release.sh"],
            cwd=ROOT,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 2)
        self.assertIn("mise run release:publish -- vX.Y.Z", result.stderr)

    def test_runs_gates_before_publish(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bin_directory = root / "bin"
            bin_directory.mkdir()
            log = root / "release.log"
            self.write_command(
                bin_directory / "mise",
                '#!/bin/sh\nprintf "mise %s\\n" "$*" >> "$PUBLISH_LOG"\n',
            )
            self.write_command(
                bin_directory / "gh",
                '#!/bin/sh\nprintf "gh %s\\n" "$*" >> "$PUBLISH_LOG"\n',
            )
            environment = {
                **os.environ,
                "PATH": f"{bin_directory}:{os.environ['PATH']}",
                "PUBLISH_LOG": str(log),
                "SKILL_CREATOR_ROOT": str(root / "unavailable-skill-creator"),
            }
            result = subprocess.run(
                ["bash", "scripts/release/publish-release.sh", "v1.2.3"],
                cwd=ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0)
            self.assertEqual(
                log.read_text(encoding="utf-8").splitlines(),
                [
                    "mise run validate",
                    "mise run verify-release -- v1.2.3",
                    "gh skill publish --tag v1.2.3",
                ],
            )
            self.assertIn("skipping Codex-specific evidence", result.stderr)

    @staticmethod
    def write_command(path: Path, contents: str) -> None:
        path.write_text(contents, encoding="utf-8")
        path.chmod(0o755)


class ReferenceIntegrityTests(unittest.TestCase):
    def test_rejects_released_skill_referencing_skill_outside_catalog(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.write_skill(
                root, "plan-issue", "Plan and hand off to `implement-issue`.\n"
            )
            self.write_skill(root, "implement-issue", "Implement the plan.\n")
            errors = VERIFY_RELEASE.find_cross_skill_references(root, {"plan-issue"})
            self.assertTrue(any("implement-issue" in error for error in errors))

    def test_accepts_released_skill_referencing_catalog_skill(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.write_skill(root, "plan-issue", "Hand off to `implement-issue`.\n")
            self.write_skill(root, "implement-issue", "Implements.\n")
            errors = VERIFY_RELEASE.find_cross_skill_references(
                root, {"plan-issue", "implement-issue"}
            )
            self.assertEqual(errors, [])

    def test_ignores_self_reference(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.write_skill(
                root, "analyze-baseline", "See also the `analyze-baseline` notes.\n"
            )
            errors = VERIFY_RELEASE.find_cross_skill_references(
                root, {"analyze-baseline"}
            )
            self.assertEqual(errors, [])

    @staticmethod
    def write_skill(root: Path, name: str, body: str) -> None:
        skill_dir = root / "skills" / name
        skill_dir.mkdir(parents=True)
        (skill_dir / "SKILL.md").write_text(
            f"---\nname: {name}\n---\n{body}", encoding="utf-8"
        )


if __name__ == "__main__":
    unittest.main()
