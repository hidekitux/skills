package deliberation

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCurrentPolicyIsValid(t *testing.T) {
	policy, err := Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if findings := Validate(policy); len(findings) != 0 {
		t.Fatalf("current policy is invalid: %s", strings.Join(findings, "; "))
	}
}

func TestCurrentPolicyCheckReportsCoverage(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Check(repositoryRoot(t), &out, &errOut); code != 0 {
		t.Fatalf("policy check failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "5 signals, 3 patterns") {
		t.Fatalf("policy coverage missing: %s", out.String())
	}
}

func TestValidateRejectsUnsafePattern(t *testing.T) {
	policy, err := Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	policy.Patterns[0].Authority = "repository_write"
	policy.Patterns[0].Bounds.MaxRetries = -1
	policy.Patterns[0].Judge = "majority"
	findings := Validate(policy)
	for _, want := range []string{"authority must be read_only", "max_retries must not be negative", "judge must require evidence"} {
		found := false
		for _, finding := range findings {
			if strings.Contains(finding, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("finding %q missing from %v", want, findings)
		}
	}
}

func TestLoadUsesStrictFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(Path)), []byte("schema_version: 1\nunknown: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("unknown field was accepted: %v", err)
	}
}
