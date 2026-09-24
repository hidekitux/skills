// Package govuln implements the pinned Go vulnerability scan behind the
// check:go-vuln mise task (cmd/scan-go-vuln). It runs the pinned govulncheck
// binary in streaming JSON mode, reports the scanner and vulnerability
// database versions from the JSON Config message, and applies the Issue #178
// failure policy: reachable findings fail, non-reachable findings are reported
// but do not fail, and infrastructure errors fail.
package govuln

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Scan exit codes: 0 clean (no reachable findings), 1 reachable findings,
// 2 infrastructure or scan error.
const (
	codePass = 0
	codeFail = 1
	codeErr  = 2
)

// runner abstracts subprocess execution so the failure policy is testable
// without invoking the real govulncheck binary.
type runner interface {
	run(dir, name string, args ...string) (stdout, stderr []byte, err error)
}

// execRunner runs the pinned govulncheck on the real toolchain.
type execRunner struct{}

func (execRunner) run(dir, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// config mirrors the govulncheck streaming JSON Config message, which must be
// the first message of a stream (schema in golang.org/x/vuln/internal/govulncheck).
type config struct {
	ProtocolVersion string     `json:"protocol_version"`
	ScannerName     string     `json:"scanner_name,omitempty"`
	ScannerVersion  string     `json:"scanner_version,omitempty"`
	DB              string     `json:"db,omitempty"`
	DBLastModified  *time.Time `json:"db_last_modified,omitempty"`
	GoVersion       string     `json:"go_version,omitempty"`
	ScanLevel       string     `json:"scan_level,omitempty"`
	ScanMode        string     `json:"scan_mode,omitempty"`
}

// position is one source position in a finding trace.
type position struct {
	Filename string `json:"filename,omitempty"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// frame is one element of a finding trace, sorted from the imported
// vulnerable symbol to the entry point.
type frame struct {
	Module   string    `json:"module"`
	Version  string    `json:"version,omitempty"`
	Package  string    `json:"package,omitempty"`
	Function string    `json:"function,omitempty"`
	Receiver string    `json:"receiver,omitempty"`
	Position *position `json:"position,omitempty"`
}

// finding mirrors the govulncheck streaming JSON Finding message. A finding is
// reachable when it was called from user code; the trace then carries frames
// with a function. Module- and package-level findings carry a single frame
// without a function and mean the vulnerable code was required or imported but
// not called.
type finding struct {
	OSV          string  `json:"osv,omitempty"`
	FixedVersion string  `json:"fixed_version,omitempty"`
	Trace        []frame `json:"trace,omitempty"`
}

// reachable reports whether any trace frame names a called function.
func (f finding) reachable() bool {
	for _, fr := range f.Trace {
		if fr.Function != "" {
			return true
		}
	}
	return false
}

// message is one element of the govulncheck JSON stream; govulncheck emits it
// for configuration, progress, the SBOM, OSV entries, and findings. Only the
// fields the failure policy needs are decoded; unknown fields are ignored.
type message struct {
	Config  *config  `json:"config,omitempty"`
	Finding *finding `json:"finding,omitempty"`
}

// result classifies the findings of one scan by reachability, one entry per
// distinct OSV identifier.
type result struct {
	cfg          *config
	reachable    []finding
	nonReachable []finding
}

// note records a finding once per OSV. govulncheck emits module-, package-,
// and symbol-level findings for the same vulnerability as analysis proceeds,
// so a later reachable finding for an OSV first seen as non-reachable
// reclassifies it.
func (r *result) note(f finding) {
	if f.reachable() {
		r.nonReachable = removeOSV(r.nonReachable, f.OSV)
		if containsOSV(r.reachable, f.OSV) {
			return
		}
		r.reachable = append(r.reachable, f)
		return
	}
	if containsOSV(r.nonReachable, f.OSV) {
		return
	}
	r.nonReachable = append(r.nonReachable, f)
}

func containsOSV(findings []finding, osv string) bool {
	for _, f := range findings {
		if f.OSV == osv {
			return true
		}
	}
	return false
}

func removeOSV(findings []finding, osv string) []finding {
	kept := findings[:0]
	for _, f := range findings {
		if f.OSV != osv {
			kept = append(kept, f)
		}
	}
	return kept
}

// decodeStream consumes the concatenated JSON messages of one govulncheck -json
// run and classifies its findings.
func decodeStream(r io.Reader) (*result, error) {
	res := &result{}
	dec := json.NewDecoder(r)
	for {
		var m message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode message: %w", err)
		}
		switch {
		case m.Config != nil:
			if res.cfg != nil {
				return nil, fmt.Errorf("duplicate config message")
			}
			res.cfg = m.Config
		case m.Finding != nil:
			res.note(*m.Finding)
		}
	}
	if res.cfg == nil {
		return nil, fmt.Errorf("missing config message")
	}
	return res, nil
}

// Run scans the module at dir with the pinned govulncheck binary and applies
// the failure policy; it is the entry point of cmd/scan-go-vuln. outPath, when
// non-empty, retains the raw JSON stream as machine-readable evidence. Package
// patterns default to ./... when none are given.
func Run(out, errOut io.Writer, dir, outPath string, patterns []string) int {
	return Scan(execRunner{}, out, errOut, dir, outPath, patterns)
}

// Scan is the testable core of Run; run must be the execRunner in production.
func Scan(run runner, out, errOut io.Writer, dir, outPath string, patterns []string) int {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	args := append([]string{"-json", "-mode", "source"}, patterns...)
	stdout, stderr, err := run.run(dir, "govulncheck", args...)
	if outPath != "" {
		if werr := os.WriteFile(outPath, stdout, 0o644); werr != nil {
			fmt.Fprintf(errOut, "failed to retain scan output: %v\n", werr)
			return codeErr
		}
		fmt.Fprintf(out, "retained JSON output: %s\n", outPath)
	}
	if err != nil {
		fmt.Fprintf(errOut, "govulncheck failed: %v\n", err)
		io.WriteString(errOut, string(stderr))
		if len(stderr) > 0 && !strings.HasSuffix(string(stderr), "\n") {
			fmt.Fprintln(errOut)
		}
		return codeErr
	}
	res, perr := decodeStream(bytes.NewReader(stdout))
	if perr != nil {
		fmt.Fprintf(errOut, "failed to parse govulncheck output: %v\n", perr)
		if len(stderr) > 0 {
			io.WriteString(errOut, string(stderr))
		}
		return codeErr
	}
	writeReport(out, res, dir, patterns)
	if len(res.reachable) > 0 {
		return codeFail
	}
	return codePass
}

// writeReport prints the scanner and database versions from the JSON Config
// header, the scan scope, and every classified finding.
func writeReport(w io.Writer, res *result, dir string, patterns []string) {
	cfg := res.cfg
	if cfg.ScannerName != "" {
		fmt.Fprintf(w, "scanner: %s %s\n", cfg.ScannerName, cfg.ScannerVersion)
	}
	if cfg.DB != "" {
		fmt.Fprintf(w, "database: %s", cfg.DB)
		if cfg.DBLastModified != nil {
			fmt.Fprintf(w, " (last modified %s)", cfg.DBLastModified.Format(time.RFC3339))
		}
		fmt.Fprintln(w)
	}
	if cfg.GoVersion != "" {
		fmt.Fprintf(w, "go version: %s\n", cfg.GoVersion)
	}
	fmt.Fprintf(w, "scanning: %s in %s\n", strings.Join(patterns, " "), dir)
	if len(res.reachable) == 0 {
		fmt.Fprintln(w, "PASS: no reachable vulnerabilities.")
	} else {
		fmt.Fprintf(w, "FAIL: %d reachable vulnerability/vulnerabilities found.\n", len(res.reachable))
		fmt.Fprintln(w, "Reachable findings fail the designated validation tier; fix the dependency or record a reviewed exception (docs/security.md).")
		for _, f := range res.reachable {
			fmt.Fprintf(w, "  - %s", f.OSV)
			if f.FixedVersion != "" {
				fmt.Fprintf(w, " (fixed in %s)", f.FixedVersion)
			}
			fmt.Fprintf(w, "\n    called from: %s\n", formatTrace(f.Trace))
		}
	}
	if len(res.nonReachable) > 0 {
		fmt.Fprintf(w, "non-reachable findings (reported, not blocking): %d\n", len(res.nonReachable))
		for _, f := range res.nonReachable {
			fmt.Fprintf(w, "  - %s", f.OSV)
			if f.FixedVersion != "" {
				fmt.Fprintf(w, " (fixed in %s)", f.FixedVersion)
			}
			fmt.Fprintln(w, ": imported but not called")
		}
	}
}

// formatTrace joins the frames of a reachable finding from the vulnerable
// symbol to the entry point.
func formatTrace(trace []frame) string {
	parts := make([]string, 0, len(trace))
	for _, fr := range trace {
		var part string
		if fr.Package != "" {
			part = fr.Package
		}
		if fr.Function != "" {
			if part != "" {
				part += "."
			}
			part += fr.Function
		}
		if fr.Position != nil && fr.Position.Line > 0 {
			part += fmt.Sprintf(" (%s:%d)", fr.Position.Filename, fr.Position.Line)
		}
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "unreachable trace"
	}
	return strings.Join(parts, " -> ")
}
