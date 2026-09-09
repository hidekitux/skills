package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixtureRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestRepresentativeTraceFixtureCoversTerminalAndHandoffStates(t *testing.T) {
	root := fixtureRepositoryRoot(t)
	path := filepath.Join(root, "workflow", "trace-fixtures", "representative.jsonl")
	traces, err := ReadJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 5 {
		t.Fatalf("trace count = %d, want 5", len(traces))
	}
	seen := map[string]bool{}
	handoffs := 0
	retries := 0
	for _, item := range traces {
		seen[item.Terminal.Status] = true
		for _, event := range item.Events {
			if event.Kind == KindHandoff {
				handoffs++
			}
			if event.Kind == KindRetry {
				retries++
			}
		}
	}
	for _, status := range []string{"success", "failed", "interrupted", "skipped"} {
		if !seen[status] {
			t.Fatalf("terminal status %q is missing: %v", status, seen)
		}
	}
	if handoffs != 2 || retries != 1 {
		t.Fatalf("handoffs=%d retries=%d", handoffs, retries)
	}
}

func TestInvalidTraceFixtureRejectsUnknownPromptField(t *testing.T) {
	root := fixtureRepositoryRoot(t)
	path := filepath.Join(root, "workflow", "trace-fixtures", "invalid", "unknown-field.jsonl")
	report := ValidateJSONL(path, root)
	if report.Valid || !contains(report.Findings, "unknown field") || !contains(report.Findings, "prompt") {
		t.Fatalf("invalid fixture was accepted: %#v", report)
	}
}

func TestSensitiveFixtureValuesNeverReachPersistedJSON(t *testing.T) {
	item := validTrace()
	item.Model = "gpt-5 password=fixture-secret"
	item.HostVersion = "prompt: private source content"
	item.Events[2].Evidence.Ref = "https://private.example/path?token=fixture-secret"
	clean, err := Sanitize(item)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, secret := range []string{"fixture-secret", "private.example", "private source content"} {
		if strings.Contains(text, secret) {
			t.Fatalf("sensitive value %q survived: %s", secret, text)
		}
	}
	if strings.Contains(text, "prompt") {
		t.Fatalf("prompt marker survived: %s", text)
	}
}

func TestCheckFixturesPassesRepositoryFixtures(t *testing.T) {
	root := fixtureRepositoryRoot(t)
	var out, errOut bytes.Buffer
	if code := CheckFixtures(root, &out, &errOut); code != 0 {
		t.Fatalf("fixture check failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "representative.jsonl") {
		t.Fatalf("fixture result missing: %s", out.String())
	}
}

func TestInvalidFixtureRemainsUnpersistable(t *testing.T) {
	root := fixtureRepositoryRoot(t)
	path := filepath.Join(root, "workflow", "trace-fixtures", "invalid", "unknown-field.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadJSONL(path); err == nil {
		t.Fatal("invalid fixture was readable")
	}
}

func TestValidateJSONLRejectsTrailingData(t *testing.T) {
	root := fixtureRepositoryRoot(t)
	path := filepath.Join(t.TempDir(), "trailing.jsonl")
	content, err := os.ReadFile(filepath.Join(root, "workflow", "trace-fixtures", "representative.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	line := strings.SplitN(string(content), "\n", 2)[0] + " trailing-json\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	report := ValidateJSONL(path, root)
	if report.Valid || !contains(report.Findings, "trailing JSON") {
		t.Fatalf("trailing data was accepted: %#v", report)
	}
}
