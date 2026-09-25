package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gitIsolationSource wraps body in a test file that imports os/exec.
func gitIsolationSource(body string) string {
	return "package demo\n\nimport (\n\t\"context\"\n\t\"os/exec\"\n)\n\nvar _ = context.Background\n\n" + body
}

func TestCheckTestGitIsolation(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		source string
		want   int
		line   string
	}{
		{
			name:   "Env assigned",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\tcmd := exec.Command(\"git\", \"init\")\n\tcmd.Env = nil\n\t_ = cmd.Run()\n}\n"),
			want:   0,
		},
		{
			name:   "Env assigned with var",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\tvar cmd = exec.CommandContext(context.Background(), \"git\", \"init\")\n\tcmd.Env = nil\n\t_ = cmd.Run()\n}\n"),
			want:   0,
		},
		{
			name:   "Env missing",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\tcmd := exec.Command(\"git\", \"init\")\n\t_ = cmd.Run()\n}\n"),
			want:   1,
			line:   "demo_test.go:11:",
		},
		{
			name:   "CommandContext Env missing",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\tcmd := exec.CommandContext(context.Background(), \"git\", \"init\")\n\t_ = cmd.Run()\n}\n"),
			want:   1,
			line:   "demo_test.go:11:",
		},
		{
			name:   "direct call",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\t_, _ = exec.Command(\"git\", \"status\").Output()\n}\n"),
			want:   1,
			line:   "demo_test.go:11:",
		},
		{
			name:   "Env set on another command",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\tother := exec.Command(\"git\", \"init\")\n\tother.Env = nil\n\tcmd := exec.Command(\"git\", \"add\")\n\t_ = other.Run()\n\t_ = cmd.Run()\n}\n"),
			want:   1,
			line:   "demo_test.go:13:",
		},
		{
			name:   "aliased import",
			file:   "demo_test.go",
			source: "package demo\n\nimport run \"os/exec\"\n\nfunc start() {\n\t_ = run.Command(\"git\", \"init\").Run()\n}\n",
			want:   1,
			line:   "demo_test.go:6:",
		},
		{
			name:   "other binary",
			file:   "demo_test.go",
			source: gitIsolationSource("func run() {\n\t_ = exec.Command(\"go\", \"version\").Run()\n}\n"),
			want:   0,
		},
		{
			name:   "non-test file",
			file:   "demo.go",
			source: gitIsolationSource("func run() {\n\t_ = exec.Command(\"git\", \"init\").Run()\n}\n"),
			want:   0,
		},
		{
			name:   "testdata skipped",
			file:   "testdata/demo_test.go",
			source: gitIsolationSource("func run() {\n\t_ = exec.Command(\"git\", \"init\").Run()\n}\n"),
			want:   0,
		},
		{
			name:   "unparsable file",
			file:   "demo_test.go",
			source: "package demo\n\nfunc {\n",
			want:   1,
			line:   "demo_test.go: cannot parse:",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "internal", "demo", filepath.FromSlash(tc.file))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.source), 0o644); err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			if got := CheckTestGitIsolation(root, &out, &errOut); got != tc.want {
				t.Fatalf("CheckTestGitIsolation() = %d, want %d; out=%q err=%q", got, tc.want, out.String(), errOut.String())
			}
			if tc.line != "" && !strings.Contains(errOut.String(), "internal/demo/"+tc.line) {
				t.Fatalf("expected a finding at internal/demo/%s, got %q", tc.line, errOut.String())
			}
		})
	}
}

// TestCheckTestGitIsolationRepository keeps every Go test in this repository
// isolated from an inherited Git environment.
func TestCheckTestGitIsolationRepository(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if got := CheckTestGitIsolation(root, &out, &errOut); got != 0 {
		t.Fatalf("CheckTestGitIsolation(repository) = %d: %s", got, errOut.String())
	}
}
