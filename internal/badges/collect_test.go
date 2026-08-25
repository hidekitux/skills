package badges

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const twoSpecMutateLog = `Mutating specs/review-flow.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 133, "killed": 90, "survived": 43, "invalid": 0}}
Mutating specs/branch-flow.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 59, "killed": 50, "survived": 9, "invalid": 0}}
`

const fullRunMutateLog = twoSpecMutateLog + `Mutating specs/release-gate.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 127, "killed": 108, "survived": 19, "invalid": 0}}
Mutating specs/skills/create-issue/issue-creation.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 200, "killed": 200, "survived": 0, "invalid": 0}}
Mutating specs/skills/create-pr/pull-request-creation.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 123, "killed": 114, "survived": 9, "invalid": 0}}
Mutating specs/skills/debug-code/debug-loop.fsl at depth 8
{"fsl": "1.0", "result": "mutated", "summary": {"total": 200, "killed": 164, "survived": 36, "invalid": 0}}
`

const fslcScriptText = `fsl_version="4.2.0"
download_base="https://github.com/ymm-oss/fsl/releases/download/v${fsl_version}"
`

const testJSONOK = `{"Time":"0001-01-01T00:00:00Z","Action":"run","Package":"p","Test":"TestExample"}
{"Time":"0001-01-01T00:00:00Z","Action":"output","Package":"p","Test":"TestExample","Output":"PASS\n"}
{"Time":"0001-01-01T00:00:00Z","Action":"pass","Package":"p","Test":"TestExample"}
{"Time":"0001-01-01T00:00:00Z","Action":"pass","Package":"p"}
`

const testJSONFailed = `{"Time":"0001-01-01T00:00:00Z","Action":"run","Package":"p","Test":"TestBad"}
{"Time":"0001-01-01T00:00:00Z","Action":"fail","Package":"p","Test":"TestBad"}
{"Time":"0001-01-01T00:00:00Z","Action":"fail","Package":"p"}
`

func TestParseMutateLogAggregatesEverySpecSummary(t *testing.T) {
	summary, err := ParseMutateLog(twoSpecMutateLog)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 192 || summary.Killed != 140 || summary.Survived != 52 {
		t.Fatalf("unexpected summary %+v", summary)
	}
	if summary, err := ParseMutateLog(fullRunMutateLog); err != nil || summary.Total != 842 || summary.Killed != 726 || summary.Survived != 116 {
		t.Fatalf("unexpected full summary %+v err=%v", summary, err)
	}
}

func TestParseMutateLogRejectsLogWithoutDocuments(t *testing.T) {
	if _, err := ParseMutateLog("Mutating specs/review-flow.fsl at depth 8\n"); err == nil {
		t.Fatal("expected error for missing documents")
	}
}

func TestParseMutateLogRejectsMissingSummaryBlock(t *testing.T) {
	if _, err := ParseMutateLog(`{"fsl": "1.0", "result": "mutated"}` + "\n"); err == nil {
		t.Fatal("expected error for missing summary block")
	}
}

func TestParseMutateLogRejectsTruncatedTrailingDocument(t *testing.T) {
	text := twoSpecMutateLog + "Mutating specs/release-gate.fsl at depth 8\n{\"summary\": {\"total\": "
	if _, err := ParseMutateLog(text); err == nil {
		t.Fatal("expected error for truncated document")
	}
}

func TestExpectedKillRate(t *testing.T) {
	cases := []struct {
		killed, total, percent, fraction int
	}{
		{726, 842, 86, 22},
		{1, 3, 33, 33},
		{2, 3, 66, 67},
		{200, 200, 100, 0},
		{20000, 20001, 100, 0},
		{19999, 20000, 100, 0},
	}
	for _, c := range cases {
		percent, fraction := ExpectedKillRate(c.killed, c.total)
		if percent != c.percent || fraction != c.fraction {
			t.Fatalf("ExpectedKillRate(%d,%d) = %d.%d, want %d.%d", c.killed, c.total, percent, fraction, c.percent, c.fraction)
		}
	}
}

func TestParseFSLCVersion(t *testing.T) {
	version, err := ParseFSLCVersion(fslcScriptText)
	if err != nil || version != "4.2.0" {
		t.Fatalf("version=%q err=%v", version, err)
	}
	if _, err := ParseFSLCVersion("download_base=https://example.com/fslc\n"); err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestParseTestJSONPassing(t *testing.T) {
	tests, err := ParseTestJSON(testJSONOK)
	if err != nil || tests.Count != 1 || !tests.OK {
		t.Fatalf("tests=%+v err=%v", tests, err)
	}
}

func TestParseTestJSONFailing(t *testing.T) {
	tests, err := ParseTestJSON(testJSONFailed)
	if err != nil || tests.Count != 1 || tests.OK {
		t.Fatalf("tests=%+v err=%v", tests, err)
	}
}

func TestParseTestJSONRejectsNoEvents(t *testing.T) {
	if _, err := ParseTestJSON("Traceback (most recent call last):\n"); err == nil {
		t.Fatal("expected error for empty event log")
	}
}

func TestRenderPayloadsSixEndpointPayloads(t *testing.T) {
	summary, _ := ParseMutateLog(fullRunMutateLog)
	payloads := RenderPayloads(summary, "4.2.0", TestSummary{Count: 85, OK: true})
	if len(payloads) != len(payloadNames) {
		t.Fatalf("expected %d payloads, got %d", len(payloadNames), len(payloads))
	}
	killed := payloads["fsl-killed.json"]
	expected := payload{SchemaVersion: 1, Label: "mutants killed", Message: "726/842", Color: "2ea44f"}
	if !reflect.DeepEqual(killed, expected) {
		t.Fatalf("unexpected killed payload %+v", killed)
	}
	if payloads["fsl-kill-rate.json"].Message != "86.22%" {
		t.Fatalf("unexpected kill-rate message %q", payloads["fsl-kill-rate.json"].Message)
	}
	if payloads["fslc-version.json"].Message != "v4.2.0" {
		t.Fatalf("unexpected fslc version %q", payloads["fslc-version.json"].Message)
	}
	if payloads["tests-status.json"].Message != "passing" {
		t.Fatalf("unexpected test status %q", payloads["tests-status.json"].Message)
	}
	failing := RenderPayloads(summary, "4.2.0", TestSummary{Count: 2, OK: false})
	if failing["tests-status.json"].Message != "failing" || failing["tests-status.json"].Color != "d73a4a" {
		t.Fatalf("unexpected failing status %+v", failing["tests-status.json"])
	}
}

func writeCollectInputs(t *testing.T, root string, mutateLog, testLog string) (string, string) {
	t.Helper()
	mutatePath := filepath.Join(root, "mutate.log")
	testPath := filepath.Join(root, "test.log")
	if err := os.WriteFile(mutatePath, []byte(mutateLog), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testPath, []byte(testLog), 0o644); err != nil {
		t.Fatal(err)
	}
	return mutatePath, testPath
}

func TestCollectBadgesWritesSixPayloads(t *testing.T) {
	root := t.TempDir()
	mutatePath, testPath := writeCollectInputs(t, root, fullRunMutateLog, testJSONOK)
	fslcPath := filepath.Join(root, "install-fslc.sh")
	if err := os.WriteFile(fslcPath, []byte(fslcScriptText), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(root, "payloads")
	var out, errOut bytes.Buffer
	if code := CollectBadges(mutatePath, testPath, fslcPath, outputDir, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, errOut.String())
	}
	entries, _ := os.ReadDir(outputDir)
	if len(entries) != len(payloadNames) {
		t.Fatalf("expected %d payload files, got %d", len(payloadNames), len(entries))
	}
	var killed payload
	content, _ := os.ReadFile(filepath.Join(outputDir, "fsl-killed.json"))
	if err := json.Unmarshal(content, &killed); err != nil || killed.Message != "726/842" {
		t.Fatalf("unexpected killed payload %+v err=%v", killed, err)
	}
}

func TestCollectBadgesTruncatedLogExitsTwo(t *testing.T) {
	root := t.TempDir()
	mutatePath, testPath := writeCollectInputs(t, root, "Mutating specs/review-flow.fsl at depth 8\n", testJSONOK)
	fslcPath := filepath.Join(root, "install-fslc.sh")
	if err := os.WriteFile(fslcPath, []byte(fslcScriptText), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := CollectBadges(mutatePath, testPath, fslcPath, filepath.Join(root, "payloads"), &out, &errOut); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCollectBadgesZeroAggregateTotalExitsTwo(t *testing.T) {
	root := t.TempDir()
	zeroLog := `{"summary": {"total": 0, "killed": 0, "survived": 0}}` + "\n"
	mutatePath, testPath := writeCollectInputs(t, root, zeroLog, testJSONOK)
	fslcPath := filepath.Join(root, "install-fslc.sh")
	if err := os.WriteFile(fslcPath, []byte(fslcScriptText), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := CollectBadges(mutatePath, testPath, fslcPath, filepath.Join(root, "payloads"), &out, &errOut); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}
