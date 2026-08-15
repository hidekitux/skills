"""Tests for the release publication gate wrapper."""

from __future__ import annotations

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]


class PublishReleaseTests(unittest.TestCase):
    def test_usage_requires_exactly_one_tag(self) -> None:
        result = subprocess.run(
            ["bash", "scripts/publish-release.sh"],
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
                ["bash", "scripts/publish-release.sh", "v1.2.3"],
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


if __name__ == "__main__":
    unittest.main()
