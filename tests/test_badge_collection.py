"""Tests for badge payload collection from mutation and test logs."""

from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).parents[1]


def load_script(path: str):
    path = ROOT / path
    spec = importlib.util.spec_from_file_location(path.stem.replace("-", "_"), path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    sys.modules[module.__name__] = module
    spec.loader.exec_module(module)
    return module


COLLECT = load_script("scripts/badges/collect-badges.py")

TWO_SPEC_MUTATE_LOG = """\
Mutating specs/review-flow.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 133, "killed": 90, "survived": 43, "invalid": 0}}
Mutating specs/branch-flow.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 59, "killed": 50, "survived": 9, "invalid": 0}}
"""

FULL_RUN_MUTATE_LOG = (
    TWO_SPEC_MUTATE_LOG
    + """\
Mutating specs/release-gate.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 127, "killed": 108, "survived": 19, "invalid": 0}}
Mutating specs/skills/create-issue/issue-creation.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 200, "killed": 200, "survived": 0, "invalid": 0}}
Mutating specs/skills/create-pr/pull-request-creation.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 123, "killed": 114, "survived": 9, "invalid": 0}}
Mutating specs/skills/debug-code/debug-loop.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 200, "killed": 164, "survived": 36, "invalid": 0}}
"""
)

FSLC_SCRIPT_TEXT = """\
fsl_version="4.2.0"
download_base="https://github.com/ymm-oss/fsl/releases/download/v${fsl_version}"
"""

TEST_LOG_OK = """\
[test] $ python -m unittest discover -s tests
.........
----------------------------------------------------------------------
Ran 85 tests in 4.119s

OK
"""

TEST_LOG_FAILED = """\
[test] $ python -m unittest discover -s tests
.E
----------------------------------------------------------------------
Ran 2 tests in 0.001s

FAILED (failures=1, errors=0)
"""

EXPECTED_FULL_RUN = COLLECT.parse_mutate_log(FULL_RUN_MUTATE_LOG)


class MutateLogParsingTests(unittest.TestCase):
    def test_aggregates_every_spec_summary(self) -> None:
        summary = COLLECT.parse_mutate_log(TWO_SPEC_MUTATE_LOG)
        self.assertEqual(summary.total, 192)
        self.assertEqual(summary.killed, 140)
        self.assertEqual(summary.survived, 52)

    def test_aggregates_a_full_six_spec_run(self) -> None:
        self.assertEqual(EXPECTED_FULL_RUN.total, 842)
        self.assertEqual(EXPECTED_FULL_RUN.killed, 726)
        self.assertEqual(EXPECTED_FULL_RUN.survived, 116)

    def test_rejects_a_log_without_mutation_documents(self) -> None:
        with self.assertRaises(ValueError):
            COLLECT.parse_mutate_log("Mutating specs/review-flow.fsl at depth 8\n")

    def test_rejects_a_document_without_a_summary_block(self) -> None:
        with self.assertRaises(TypeError):
            COLLECT.parse_mutate_log('{"fsl": "1.0", "result": "mutated"}\n')

    def test_rejects_a_malformed_summary_block(self) -> None:
        text = '{"summary": {"total": "many", "killed": 1, "survived": 0}}\n'
        with self.assertRaises(ValueError):
            COLLECT.parse_mutate_log(text)

    def test_rejects_a_truncated_trailing_document(self) -> None:
        text = (
            "Mutating specs/review-flow.fsl at depth 8\n"
            '{"fsl": "1.0", "result": "mutated", '
            '"summary": {"total": 133, "killed": 90, "survived": 43, "invalid": 0}}\n'
            "Mutating specs/branch-flow.fsl at depth 8\n"
            '{"fsl": "1.0", "result": "mutated", '
            '"summary": {"total": 59, "killed": 50, "survived": 9, "invalid": 0}}\n'
            "Mutating specs/release-gate.fsl at depth 8\n"
            '{"fsl": "1.0", "result": "mutated", "summary": {"total": '
        )
        with self.assertRaises(ValueError):
            COLLECT.parse_mutate_log(text)

    def test_rejects_a_missing_document_for_a_mutation_run(self) -> None:
        text = (
            "Mutating specs/review-flow.fsl at depth 8\n"
            '{"fsl": "1.0", "result": "mutated", '
            '"summary": {"total": 133, "killed": 90, "survived": 43, "invalid": 0}}\n'
            "Mutating specs/branch-flow.fsl at depth 8\n"
        )
        with self.assertRaises(ValueError):
            COLLECT.parse_mutate_log(text)


class KillRateRoundingTests(unittest.TestCase):
    def test_matches_the_recorded_six_spec_run(self) -> None:
        self.assertEqual(
            COLLECT.expected_kill_rate(
                EXPECTED_FULL_RUN.killed, EXPECTED_FULL_RUN.total
            ),
            (86, 22),
        )

    def test_rounds_fractions_to_two_decimals(self) -> None:
        self.assertEqual(COLLECT.expected_kill_rate(1, 3), (33, 33))
        self.assertEqual(COLLECT.expected_kill_rate(2, 3), (66, 67))

    def test_handles_a_perfect_run(self) -> None:
        self.assertEqual(COLLECT.expected_kill_rate(200, 200), (100, 0))

    def test_carries_rounding_into_the_whole_percent(self) -> None:
        self.assertEqual(COLLECT.expected_kill_rate(20000, 20001), (100, 0))
        self.assertEqual(COLLECT.expected_kill_rate(19999, 20000), (100, 0))


class FslcVersionParsingTests(unittest.TestCase):
    def test_parses_the_pinned_fslc_version(self) -> None:
        self.assertEqual(COLLECT.parse_fslc_version(FSLC_SCRIPT_TEXT), "4.2.0")

    def test_rejects_a_script_without_a_pinned_version(self) -> None:
        with self.assertRaises(ValueError):
            COLLECT.parse_fslc_version("download_base=https://example.com/fslc\n")


class TestLogParsingTests(unittest.TestCase):
    def test_parses_a_passing_run(self) -> None:
        tests = COLLECT.parse_test_log(TEST_LOG_OK)
        self.assertEqual(tests.count, 85)
        self.assertTrue(tests.ok)

    def test_parses_a_failing_run(self) -> None:
        tests = COLLECT.parse_test_log(TEST_LOG_FAILED)
        self.assertEqual(tests.count, 2)
        self.assertFalse(tests.ok)

    def test_rejects_a_log_without_a_run_count(self) -> None:
        with self.assertRaises(ValueError):
            COLLECT.parse_test_log("Traceback (most recent call last):\n")

    def test_rejects_a_log_without_an_outcome(self) -> None:
        with self.assertRaises(ValueError):
            COLLECT.parse_test_log("Ran 3 tests in 0.001s\n")


class PayloadRenderingTests(unittest.TestCase):
    def test_renders_the_six_endpoint_payloads(self) -> None:
        payloads = COLLECT.render_payloads(
            EXPECTED_FULL_RUN, "4.2.0", COLLECT.TestSummary(count=85, ok=True)
        )
        self.assertEqual(set(payloads), set(COLLECT.PAYLOAD_NAMES))
        killed = payloads["fsl-killed.json"]
        self.assertEqual(
            killed,
            {
                "schemaVersion": 1,
                "label": "mutants killed",
                "message": "726/842",
                "color": "2ea44f",
            },
        )
        self.assertEqual(payloads["fsl-kill-rate.json"]["message"], "86.22%")
        self.assertEqual(payloads["fsl-survived.json"]["message"], "116")
        self.assertEqual(payloads["fslc-version.json"]["message"], "v4.2.0")
        self.assertEqual(payloads["tests-status.json"]["message"], "passing")
        self.assertEqual(payloads["tests-run.json"]["message"], "85")

    def test_renders_a_failing_test_status(self) -> None:
        payloads = COLLECT.render_payloads(
            EXPECTED_FULL_RUN, "4.2.0", COLLECT.TestSummary(count=2, ok=False)
        )
        self.assertEqual(payloads["tests-status.json"]["message"], "failing")
        self.assertEqual(payloads["tests-status.json"]["color"], "d73a4a")


class CliExitCodeTests(unittest.TestCase):
    def _write_inputs(
        self, root: Path, mutate_log: str, test_log: str
    ) -> tuple[Path, Path]:
        mutate_path = root / "mutate.log"
        test_path = root / "test.log"
        mutate_path.write_text(mutate_log, encoding="utf-8")
        test_path.write_text(test_log, encoding="utf-8")
        return mutate_path, test_path

    def test_writes_all_six_payloads_and_returns_zero(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mutate_path, test_path = self._write_inputs(
                root, FULL_RUN_MUTATE_LOG, TEST_LOG_OK
            )
            output_dir = root / "payloads"
            exit_code = COLLECT.main(
                [
                    "--mutate-log",
                    str(mutate_path),
                    "--test-log",
                    str(test_path),
                    "--output-dir",
                    str(output_dir),
                ]
            )
            self.assertEqual(exit_code, 0)
            self.assertEqual(
                {path.name for path in output_dir.iterdir()}, set(COLLECT.PAYLOAD_NAMES)
            )
            payload = json.loads(
                (output_dir / "fsl-killed.json").read_text(encoding="utf-8")
            )
            self.assertEqual(payload["message"], "726/842")

    def test_truncated_trailing_document_exits_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mutate_log = (
                "Mutating specs/review-flow.fsl at depth 8\n"
                '{"fsl": "1.0", "result": "mutated", '
                '"summary": {"total": 133, "killed": 90, "survived": 43, '
                '"invalid": 0}}\n'
                "Mutating specs/branch-flow.fsl at depth 8\n"
                '{"fsl": "1.0", "result": "mutated", '
                '"summary": {"total": 59, "killed": 50, "survived": 9, '
                '"invalid": 0}}\n'
                "Mutating specs/release-gate.fsl at depth 8\n"
                '{"fsl": "1.0", "result": "mutated", "summary": {"total": '
            )
            mutate_path, test_path = self._write_inputs(root, mutate_log, TEST_LOG_OK)
            with self.assertRaises(SystemExit) as ctx:
                COLLECT.main(
                    [
                        "--mutate-log",
                        str(mutate_path),
                        "--test-log",
                        str(test_path),
                        "--output-dir",
                        str(root / "payloads"),
                    ]
                )
            self.assertEqual(ctx.exception.code, 2)

    def test_truncated_mutate_log_exits_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mutate_path, test_path = self._write_inputs(
                root, "Mutating specs/review-flow.fsl at depth 8\n", TEST_LOG_OK
            )
            with self.assertRaises(SystemExit) as ctx:
                COLLECT.main(
                    [
                        "--mutate-log",
                        str(mutate_path),
                        "--test-log",
                        str(test_path),
                        "--output-dir",
                        str(root / "payloads"),
                    ]
                )
            self.assertEqual(ctx.exception.code, 2)

    def test_unparseable_test_log_exits_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mutate_path, test_path = self._write_inputs(
                root, FULL_RUN_MUTATE_LOG, "Traceback (most recent call last):\n"
            )
            with self.assertRaises(SystemExit) as ctx:
                COLLECT.main(
                    [
                        "--mutate-log",
                        str(mutate_path),
                        "--test-log",
                        str(test_path),
                        "--output-dir",
                        str(root / "payloads"),
                    ]
                )
            self.assertEqual(ctx.exception.code, 2)

    def test_zero_aggregate_total_exits_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            zero_log = '{"summary": {"total": 0, "killed": 0, "survived": 0}}\n'
            mutate_path, test_path = self._write_inputs(root, zero_log, TEST_LOG_OK)
            with self.assertRaises(SystemExit) as ctx:
                COLLECT.main(
                    [
                        "--mutate-log",
                        str(mutate_path),
                        "--test-log",
                        str(test_path),
                        "--output-dir",
                        str(root / "payloads"),
                    ]
                )
            self.assertEqual(ctx.exception.code, 2)

    def test_missing_fslc_version_exits_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mutate_path, test_path = self._write_inputs(
                root, FULL_RUN_MUTATE_LOG, TEST_LOG_OK
            )
            fslc_script = root / "install-fslc.sh"
            fslc_script.write_text(
                "download_base=https://example.com/fslc\n", encoding="utf-8"
            )
            with self.assertRaises(SystemExit) as ctx:
                COLLECT.main(
                    [
                        "--mutate-log",
                        str(mutate_path),
                        "--test-log",
                        str(test_path),
                        "--fslc-script",
                        str(fslc_script),
                        "--output-dir",
                        str(root / "payloads"),
                    ]
                )
            self.assertEqual(ctx.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
