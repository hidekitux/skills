package govuln

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner returns canned stdout, stderr, and error, and records the
// invocation so tests can assert the govulncheck arguments.
type fakeRunner struct {
	stdout string
	stderr string
	err    error
	dir    string
	name   string
	args   []string
}

func (f *fakeRunner) run(dir, name string, args ...string) ([]byte, []byte, error) {
	f.dir, f.name, f.args = dir, name, args
	return []byte(f.stdout), []byte(f.stderr), f.err
}

const cleanStream = `{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.7.0","db":"https://vuln.go.dev","db_last_modified":"2026-08-26T12:00:00Z","go_version":"go1.26.6","scan_level":"symbol","scan_mode":"source"}}
`

const packageLevelFinding = `{"finding":{"osv":"GO-2024-2687","fixed_version":"v0.3.8","trace":[{"module":"golang.org/x/text","version":"v0.3.0","package":"golang.org/x/text/language"}]}}
`

const symbolLevelFinding = `{"finding":{"osv":"GO-2024-2687","fixed_version":"v0.3.8","trace":[{"module":"golang.org/x/text","version":"v0.3.0","package":"golang.org/x/text/language","function":"Parse","position":{"filename":"language/parse.go","line":123,"column":5}},{"module":"example.com/app","package":"main","function":"main","position":{"filename":"main.go","line":9,"column":2}}]}}
`

func runWith(t *testing.T, stdout string, code int) string {
	t.Helper()
	fake := &fakeRunner{stdout: stdout}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", "", nil); got != code {
		t.Fatalf("Scan() = %d, want %d\nstdout: %s\nstderr: %s", got, code, out.String(), errOut.String())
	}
	return out.String()
}

func TestScanDefaultPatternsInvokesPinnedJSONMode(t *testing.T) {
	fake := &fakeRunner{stdout: cleanStream}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", "", nil); got != codePass {
		t.Fatalf("Scan() = %d, want %d", got, codePass)
	}
	if fake.name != "govulncheck" {
		t.Fatalf("ran %q, want govulncheck", fake.name)
	}
	want := []string{"-json", "-mode", "source", "./..."}
	if len(fake.args) != len(want) {
		t.Fatalf("args = %v, want %v", fake.args, want)
	}
	for i := range want {
		if fake.args[i] != want[i] {
			t.Fatalf("args = %v, want %v", fake.args, want)
		}
	}
	if fake.dir != "." {
		t.Fatalf("dir = %q, want .", fake.dir)
	}
}

func TestScanCustomPatterns(t *testing.T) {
	fake := &fakeRunner{stdout: cleanStream}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", "", []string{"./cmd/..."}); got != codePass {
		t.Fatalf("Scan() = %d, want %d", got, codePass)
	}
	if fake.args[len(fake.args)-1] != "./cmd/..." {
		t.Fatalf("last arg = %q, want ./cmd/...", fake.args[len(fake.args)-1])
	}
}

func TestScanCleanReportsVersions(t *testing.T) {
	out := runWith(t, cleanStream, codePass)
	for _, want := range []string{"PASS", "govulncheck v1.7.0", "https://vuln.go.dev", "2026-08-26"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestScanNonReachablePassesAndReports(t *testing.T) {
	out := runWith(t, cleanStream+packageLevelFinding, codePass)
	for _, want := range []string{"PASS", "non-reachable findings (reported, not blocking): 1", "GO-2024-2687 (fixed in v0.3.8)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestScanReachableFails(t *testing.T) {
	out := runWith(t, cleanStream+symbolLevelFinding, codeFail)
	for _, want := range []string{"FAIL: 1", "GO-2024-2687 (fixed in v0.3.8)", "called from", "language/parse.go:123"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestScanReclassifiesNonReachableToReachable(t *testing.T) {
	// govulncheck emits module/package-level findings before the symbol-level
	// trace for the same OSV; the final classification must be reachable.
	out := runWith(t, cleanStream+packageLevelFinding+symbolLevelFinding, codeFail)
	if strings.Contains(out, "non-reachable findings") {
		t.Fatalf("non-reachable bucket must be empty after reclassification:\n%s", out)
	}
	if !strings.Contains(out, "FAIL: 1") {
		t.Fatalf("expected single reachable finding:\n%s", out)
	}
}

func TestScanInfrastructureErrorFails(t *testing.T) {
	fake := &fakeRunner{stderr: "get \"https://vuln.go.dev\": dial tcp: i/o timeout\n", err: errors.New("exit status 1")}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", "", nil); got != codeErr {
		t.Fatalf("Scan() = %d, want %d", got, codeErr)
	}
	if !strings.Contains(errOut.String(), "govulncheck failed") || !strings.Contains(errOut.String(), "dial tcp") {
		t.Fatalf("stderr missing failure detail:\n%s", errOut.String())
	}
}

func TestScanMalformedJSONFails(t *testing.T) {
	fake := &fakeRunner{stdout: "this is not json", err: nil}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", "", nil); got != codeErr {
		t.Fatalf("Scan() = %d, want %d", got, codeErr)
	}
	if !strings.Contains(errOut.String(), "failed to parse govulncheck output") {
		t.Fatalf("stderr missing parse failure:\n%s", errOut.String())
	}
}

func TestScanMissingConfigFails(t *testing.T) {
	stream := `{"finding":{"osv":"GO-2024-2687","trace":[{"module":"golang.org/x/text","version":"v0.3.0"}]}}
`
	var out, errOut bytes.Buffer
	fake := &fakeRunner{stdout: stream}
	if got := Scan(fake, &out, &errOut, ".", "", nil); got != codeErr {
		t.Fatalf("Scan() = %d, want %d", got, codeErr)
	}
	if !strings.Contains(errOut.String(), "missing config message") {
		t.Fatalf("stderr missing config diagnostic:\n%s", errOut.String())
	}
}

func TestScanRetainsRawOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scan.json")
	fake := &fakeRunner{stdout: cleanStream + symbolLevelFinding}
	var out, errOut bytes.Buffer
	if got := Scan(fake, &out, &errOut, ".", path, nil); got != codeFail {
		t.Fatalf("Scan() = %d, want %d", got, codeFail)
	}
	retained, rerr := os.ReadFile(path)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(retained) != fake.stdout {
		t.Fatalf("retained output does not match the raw JSON stream")
	}
	if !strings.Contains(out.String(), "retained JSON output: "+path) {
		t.Fatalf("output missing retention note:\n%s", out.String())
	}
}

func TestReachableClassification(t *testing.T) {
	cases := []struct {
		name  string
		frame []frame
		want  bool
	}{
		{"module level", []frame{{Module: "golang.org/x/text", Version: "v0.3.0"}}, false},
		{"package level", []frame{{Module: "golang.org/x/text", Version: "v0.3.0", Package: "golang.org/x/text/language"}}, false},
		{"symbol level", []frame{{Module: "golang.org/x/text", Version: "v0.3.0", Package: "golang.org/x/text/language", Function: "Parse"}}, true},
		{"empty trace", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := (finding{OSV: "GO-1", Trace: tc.frame}).reachable(); got != tc.want {
				t.Fatalf("reachable() = %v, want %v", got, tc.want)
			}
		})
	}
}
