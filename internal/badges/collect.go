// Package badges collects FSL mutation and test results into shields.io
// endpoint payloads, ported from scripts/badges/collect-badges.py.
//
// Unlike the Python original, which parsed unittest "Ran N tests" output, the
// test summary is derived from `go test -json ./...` events so the badge data
// workflow can consume Go test output directly.
package badges

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var mutatingLineRE = regexp.MustCompile(`(?m)^Mutating \S+ at depth \d+\r?$`)
var fslcVersionRE = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])fsl_version="([^"]+)"`)

// payloadNames is the fixed set of endpoint payloads written by the collector.
var payloadNames = []string{
	"fsl-killed.json",
	"fsl-kill-rate.json",
	"fsl-survived.json",
	"fslc-version.json",
	"tests-status.json",
	"tests-run.json",
}

// MutationSummary aggregates the mutation-document summaries in a log.
type MutationSummary struct {
	Total    int
	Killed   int
	Survived int
}

// TestSummary reports the count and outcome derived from go test -json events.
type TestSummary struct {
	Count int
	OK    bool
}

// ParseMutateLog aggregates the summary block of every mutation document in
// the log. fslc emits each mutation document as pretty-printed JSON that may
// span multiple lines, so documents are streamed rather than parsed one line at
// a time: non-JSON lines (such as the `Mutating <spec> at depth N` headers) are
// skipped, and each JSON document is accumulated until it forms a complete
// value. Every `Mutating <spec> at depth N` line must be followed by exactly
// one mutation document, so a truncated or missing document fails loudly.
func ParseMutateLog(text string) (MutationSummary, error) {
	summaries := []MutationSummary{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	var document bytes.Buffer
	inDocument := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inDocument {
			if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
				continue
			}
			document.Reset()
			inDocument = true
		}
		document.WriteString(line)
		document.WriteByte('\n')
		if !json.Valid(document.Bytes()) {
			continue
		}

		var doc map[string]any
		if err := json.Unmarshal(document.Bytes(), &doc); err != nil {
			return MutationSummary{}, err
		}
		summary, ok := doc["summary"].(map[string]any)
		if !ok {
			return MutationSummary{}, errors.New("mutate-fsl document is missing its summary block")
		}
		total, err := toInt(summary["total"])
		if err != nil {
			return MutationSummary{}, fmt.Errorf("malformed summary block: %v", err)
		}
		killed, err := toInt(summary["killed"])
		if err != nil {
			return MutationSummary{}, fmt.Errorf("malformed summary block: %v", err)
		}
		survived, err := toInt(summary["survived"])
		if err != nil {
			return MutationSummary{}, fmt.Errorf("malformed summary block: %v", err)
		}
		summaries = append(summaries, MutationSummary{Total: total, Killed: killed, Survived: survived})
		document.Reset()
		inDocument = false
	}
	if err := scanner.Err(); err != nil {
		return MutationSummary{}, err
	}

	mutationRuns := len(mutatingLineRE.FindAllString(text, -1))
	if mutationRuns != len(summaries) {
		return MutationSummary{}, fmt.Errorf(
			"mutate-fsl log parsed %d document(s) for %d mutation run(s); the output is truncated or missing a document",
			len(summaries), mutationRuns,
		)
	}
	if len(summaries) == 0 {
		return MutationSummary{}, errors.New("mutate-fsl log contains no mutation documents")
	}
	aggregate := MutationSummary{}
	for _, s := range summaries {
		aggregate.Total += s.Total
		aggregate.Killed += s.Killed
		aggregate.Survived += s.Survived
	}
	return aggregate, nil
}

func toInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		n, err := v.Int64()
		return int(n), err
	default:
		return 0, fmt.Errorf("expected an integer, got %T", value)
	}
}

// ParseFSLCVersion extracts the pinned fsl_version from install-fslc.sh.
func ParseFSLCVersion(text string) (string, error) {
	match := fslcVersionRE.FindStringSubmatch(text)
	if match == nil {
		return "", errors.New("cannot find the pinned fsl_version in install-fslc.sh")
	}
	return match[1], nil
}

// ParseTestJSON derives a TestSummary from go test -json event lines.
func ParseTestJSON(text string) (TestSummary, error) {
	type event struct {
		Action string `json:"Action"`
		Test   string `json:"Test"`
	}
	fail := false
	count := 0
	anyEvent := false
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		anyEvent = true
		if ev.Action == "fail" {
			fail = true
		}
		if ev.Test != "" && (ev.Action == "pass" || ev.Action == "fail") {
			count++
		}
	}
	if !anyEvent {
		return TestSummary{}, errors.New("test log reports no go test -json events")
	}
	return TestSummary{Count: count, OK: !fail}, nil
}

// ExpectedKillRate returns the (whole, fraction) kill rate to two decimals,
// rounding half to even to match the Python collector.
func ExpectedKillRate(killed, total int) (int, int) {
	hundredths := roundHalfEven(float64(killed) / float64(total) * 10_000)
	return int(hundredths) / 100, int(hundredths) % 100
}

func roundHalfEven(x float64) int64 {
	floor := math.Floor(x)
	frac := x - floor
	if frac < 0.5 {
		return int64(floor)
	}
	if frac > 0.5 {
		return int64(floor + 1)
	}
	if math.Mod(floor, 2) == 0 {
		return int64(floor)
	}
	return int64(floor + 1)
}

type payload struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// RenderPayloads returns the six endpoint payloads for a summary.
func RenderPayloads(summary MutationSummary, fslcVersion string, tests TestSummary) map[string]payload {
	percent, fraction := ExpectedKillRate(summary.Killed, summary.Total)
	testStatus := "passing"
	testColor := "2ea44f"
	if !tests.OK {
		testStatus = "failing"
		testColor = "d73a4a"
	}
	return map[string]payload{
		"fsl-killed.json":    {1, "mutants killed", fmt.Sprintf("%d/%d", summary.Killed, summary.Total), "2ea44f"},
		"fsl-kill-rate.json": {1, "kill rate", fmt.Sprintf("%d.%02d%%", percent, fraction), "2ea44f"},
		"fsl-survived.json":  {1, "surviving mutants", fmt.Sprintf("%d", summary.Survived), "a371f7"},
		"fslc-version.json":  {1, "fslc", "v" + fslcVersion, "0b6bcb"},
		"tests-status.json":  {1, "tests", testStatus, testColor},
		"tests-run.json":     {1, "tests run", fmt.Sprintf("%d", tests.Count), "2ea44f"},
	}
}

// CollectBadges reads the mutation log, the go test -json log, and the pinned
// fslc version, writes the six endpoint payloads to outputDir, and returns the
// process exit code (2 on a malformed input).
func CollectBadges(mutateLog, testLog, fslcScript, outputDir string, out, errOut io.Writer) int {
	mutateText, err := os.ReadFile(mutateLog)
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	testText, err := os.ReadFile(testLog)
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	fslcText, err := os.ReadFile(fslcScript)
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	summary, err := ParseMutateLog(string(mutateText))
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	tests, err := ParseTestJSON(string(testText))
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	fslcVersion, err := ParseFSLCVersion(string(fslcText))
	if err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	if summary.Total <= 0 {
		fmt.Fprintln(errOut, "Badge collection failed: aggregate mutation total must be positive")
		return 2
	}

	percent, fraction := ExpectedKillRate(summary.Killed, summary.Total)
	fmt.Fprintf(out, "FSL mutation badges: %d/%d killed, %d.%02d%% kill rate, %d surviving mutants.\n",
		summary.Killed, summary.Total, percent, fraction, summary.Survived)
	fmt.Fprintf(out, "FSL verifier: fslc %s\n", fslcVersion)
	status := "passing"
	if !tests.OK {
		status = "failing"
	}
	fmt.Fprintf(out, "Tests: %d run, %s.\n", tests.Count, status)

	payloads := RenderPayloads(summary, fslcVersion, tests)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
		return 2
	}
	for _, name := range payloadNames {
		encoded, err := json.MarshalIndent(payloads[name], "", "  ")
		if err != nil {
			fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
			return 2
		}
		if err := os.WriteFile(filepath.Join(outputDir, name), append(encoded, '\n'), 0o644); err != nil {
			fmt.Fprintf(errOut, "Badge collection failed: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(out, "Wrote %d badge payloads to %s.\n", len(payloads), outputDir)
	return 0
}
