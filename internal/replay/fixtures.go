package replay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fixtureManifest struct {
	Fixtures []fixtureExpectation `json:"fixtures"`
}

type fixtureExpectation struct {
	Path          string  `json:"path"`
	Valid         bool    `json:"valid"`
	Outcome       Outcome `json:"outcome"`
	Invariant     string  `json:"invariant,omitempty"`
	EventSequence int     `json:"event_sequence,omitempty"`
}

// CheckFixtures replays the committed cross-skill fixtures and checks the
// expected outcome, invariant, and source event boundary without reading raw
// event payloads into its report.
func CheckFixtures(root string, out, errOut io.Writer) int {
	base := filepath.Join(root, "workflow", "replay-fixtures")
	manifestPath := filepath.Join(base, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintf(errOut, "replay fixture check failed: %v\n", err)
		return 1
	}
	var manifest fixtureManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil || len(manifest.Fixtures) == 0 {
		fmt.Fprintln(errOut, "replay fixture check failed: manifest is invalid")
		return 1
	}
	for _, expected := range manifest.Fixtures {
		if expected.Path == "" || filepath.IsAbs(expected.Path) || expected.Path == "." || filepath.Clean(expected.Path) != expected.Path || filepath.HasPrefix(expected.Path, ".."+string(filepath.Separator)) {
			fmt.Fprintf(errOut, "replay fixture check failed: unsafe fixture path %q\n", expected.Path)
			return 1
		}
		set, err := ReadJSONL(filepath.Join(base, expected.Path))
		if err != nil {
			fmt.Fprintf(errOut, "replay fixture %s could not be read: %v\n", expected.Path, err)
			return 1
		}
		report := Replay(root, set)
		if report.Valid != expected.Valid || report.Outcome != expected.Outcome {
			fmt.Fprintf(errOut, "replay fixture %s returned valid=%t outcome=%s; want valid=%t outcome=%s\n", expected.Path, report.Valid, report.Outcome, expected.Valid, expected.Outcome)
			return 1
		}
		if expected.Invariant != "" {
			if len(report.Findings) == 0 || report.Findings[0].Invariant != expected.Invariant || (expected.EventSequence > 0 && report.Findings[0].EventSequence != expected.EventSequence) {
				fmt.Fprintf(errOut, "replay fixture %s finding = %#v; want invariant %s at event %d\n", expected.Path, report.Findings, expected.Invariant, expected.EventSequence)
				return 1
			}
		}
		fmt.Fprintf(out, "replay fixture passed: %s (%s)\n", expected.Path, report.Outcome)
	}
	fmt.Fprintf(out, "replay fixture check passed: %d file(s).\n", len(manifest.Fixtures))
	return 0
}
