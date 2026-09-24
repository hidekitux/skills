package check

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCheck(t *testing.T, fn func(root string, out, errOut io.Writer) int, root string) int {
	t.Helper()
	var out, errOut bytes.Buffer
	return fn(root, &out, &errOut)
}

func TestSensitiveContentAcceptsPublicText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("See https://example.com/docs.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runCheck(t, CheckSensitiveContent, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestSensitiveContentRejectsATokenAndPrivateURL(t *testing.T) {
	root := t.TempDir()
	token := "gh" + "p_" + strings.Repeat("a", 20)
	privateURL := "https://" + "192.168.1.1/private"
	content := token + "\n" + privateURL + "\n"
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runCheck(t, CheckSensitiveContent, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

const endpointBadges = `![FSL mutants killed](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-killed.json)
![FSL kill rate](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-kill-rate.json)
![FSL surviving mutants](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-survived.json)
![FSL verifier](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffslc-version.json)
![Tests status](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-status.json)
![Tests run](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-run.json)
`

func writeReadme(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMutationBadgesAcceptsSixEndpointBadges(t *testing.T) {
	root := t.TempDir()
	writeReadme(t, root, endpointBadges)
	if code := runCheck(t, CheckMutationBadges, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestMutationBadgesRejectsAStaticFSLBadge(t *testing.T) {
	root := t.TempDir()
	static := strings.Replace(
		endpointBadges,
		"https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-killed.json",
		"https://img.shields.io/badge/mutants%20killed-164%2F200-2ea44f",
		1,
	)
	writeReadme(t, root, static)
	if code := runCheck(t, CheckMutationBadges, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationBadgesRejectsAForeignEndpointPayload(t *testing.T) {
	root := t.TempDir()
	foreign := strings.Replace(endpointBadges, "%2Fbadge-data%2Ffsl-killed.json", "%2Fimages%2Ffsl-killed.json", 1)
	writeReadme(t, root, foreign)
	if code := runCheck(t, CheckMutationBadges, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationBadgesRejectsAMissingTestBadge(t *testing.T) {
	root := t.TempDir()
	runLine := "![Tests run](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-run.json)\n"
	writeReadme(t, root, strings.Replace(endpointBadges, runLine, "", 1))
	if code := runCheck(t, CheckMutationBadges, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func writeSkill(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\n---\n" + body
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeReadonlyAcceptsAnalyzeSkillWithoutCreationInstructions(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "analyze-project", "Inspect the code and report findings.\n")
	if code := runCheck(t, CheckAnalyzeReadonly, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestAnalyzeReadonlyRejectsIssueCreationInstruction(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "analyze-project", "Create a GitHub issue to track the finding.\n")
	if code := runCheck(t, CheckAnalyzeReadonly, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestAnalyzeReadonlyRejectsPRCreationInstruction(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "analyze-baseline", "Open a pull request with the suggested change.\n")
	if code := runCheck(t, CheckAnalyzeReadonly, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestAnalyzeReadonlyIgnoresNonAnalyzeSkill(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "create-issue", "Create a GitHub issue for the work.\n")
	if code := runCheck(t, CheckAnalyzeReadonly, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestToolLicensesRequiresEveryMiseTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "mise.toml", "[tools]\npython = \"3.11\"\n")
	writeFile(t, root, "TOOL_LICENSES.toml", "[tools]\n")
	if code := runCheck(t, CheckToolLicenses, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestToolLicensesRequiresEveryDirectGoModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "mise.toml", "[tools]\n")
	writeFile(t, root, "go.mod", "module example.com/fixture\n\ngo 1.21\n\nrequire (\n\tgopkg.in/yaml.v3 v3.0.1\n)\n")
	writeFile(t, root, "TOOL_LICENSES.toml", "[tools]\n[go_modules]\n")
	if code := runCheck(t, CheckToolLicenses, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
