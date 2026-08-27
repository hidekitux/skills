package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDependabotYAML = `version: 2
updates:
  - package-ecosystem: github-actions
    directory: "/"
    schedule:
      interval: weekly
    commit-message:
      prefix: "ci"
    open-pull-requests-limit: 5
  - package-ecosystem: gomod
    directory: "/"
    schedule:
      interval: weekly
    commit-message:
      prefix: "ci"
    open-pull-requests-limit: 5
`

func writeDependabotConfig(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".github")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dependabot.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runDependabotCheck(t *testing.T, content string) (int, string) {
	t.Helper()
	root := writeDependabotConfig(t, content)
	var out, errOut bytes.Buffer
	code := CheckDependabotConfig(root, &out, &errOut)
	return code, out.String() + errOut.String()
}

func TestDependabotConfigPassesWhenCoveredAndBounded(t *testing.T) {
	code, output := runDependabotCheck(t, validDependabotYAML)
	if code != 0 {
		t.Fatalf("expected 0, got %d:\n%s", code, output)
	}
	for _, want := range []string{"Dependabot-config check passed", "2 ecosystem(s)"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestDependabotConfigRejectsMissingEcosystem(t *testing.T) {
	code, output := runDependabotCheck(t, validDependabotYAML)
	if code != 0 {
		t.Fatalf("expected 0, got %d:\n%s", code, output)
	}
	gomodIdx := strings.Index(validDependabotYAML, "  - package-ecosystem: gomod")
	if gomodIdx < 0 {
		t.Fatal("fixture missing gomod entry")
	}
	code, output = runDependabotCheck(t, validDependabotYAML[:gomodIdx])
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "gomod: missing Dependabot update entry") {
		t.Fatalf("output missing gomod diagnostic:\n%s", output)
	}
}

func TestDependabotConfigRejectsUnboundedFrequency(t *testing.T) {
	broken := strings.Replace(validDependabotYAML, "interval: weekly", "interval: daily", 1)
	code, output := runDependabotCheck(t, broken)
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "schedule.interval must be weekly") {
		t.Fatalf("output missing interval diagnostic:\n%s", output)
	}
}

func TestDependabotConfigRejectsUnboundedPullRequestLimit(t *testing.T) {
	broken := strings.Replace(validDependabotYAML, "open-pull-requests-limit: 5", "open-pull-requests-limit: 20", 1)
	code, output := runDependabotCheck(t, broken)
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "open-pull-requests-limit must be between 1 and 5") {
		t.Fatalf("output missing limit diagnostic:\n%s", output)
	}
}

func TestDependabotConfigRejectsMissingCommitPrefix(t *testing.T) {
	broken := strings.Replace(validDependabotYAML, "      prefix: \"ci\"\n", "", 1)
	code, output := runDependabotCheck(t, broken)
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "commit-message.prefix must be ci") {
		t.Fatalf("output missing prefix diagnostic:\n%s", output)
	}
}

func TestDependabotConfigFailsOnMalformedYAML(t *testing.T) {
	code, output := runDependabotCheck(t, "version: 2\nupdates: [\n")
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "cannot parse dependabot.yml") {
		t.Fatalf("output missing parse diagnostic:\n%s", output)
	}
}

func TestDependabotConfigFailsWhenFileMissing(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	if code := CheckDependabotConfig(root, &out, &errOut); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}
