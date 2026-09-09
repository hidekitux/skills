package validate

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSkillDirsUnderIncludesFlatAndNamespacedSkills(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "skills/plan-issue/SKILL.md", "---\nname: plan-issue\n---\n")
	writeVFile(t, root, "skills/skills/refactor-code/SKILL.md", "---\nname: refactor-code\n---\n")

	want := []string{
		filepath.Join(root, "skills", "plan-issue"),
		filepath.Join(root, "skills", "skills", "refactor-code"),
	}
	if got := skillDirsUnder(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("skill directories = %v, want %v", got, want)
	}
}

func TestValidateSkillCreatorReportsMissingValidatorWithoutRunningExternalTool(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "skills/plan-issue/SKILL.md", "---\nname: plan-issue\n---\n")

	selectedRoot := filepath.Join(t.TempDir(), "missing")
	fallbackRoot := t.TempDir()
	writeVFile(t, fallbackRoot, "scripts/quick_validate.py", "# fallback fixture\n")
	bin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "uv-called")
	writeExecutable(t, bin, "uv", "#!/bin/sh\nprintf called > \"$UV_TEST_MARKER\"\nexit 99\n")

	t.Setenv("SKILL_CREATOR_ROOT", selectedRoot)
	t.Setenv("CODEX_HOME", fallbackRoot)
	t.Setenv("UV_TEST_MARKER", marker)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out, errOut bytes.Buffer
	if code := ValidateSkillCreator(root, &out, &errOut); code != 2 {
		t.Fatalf("expected missing validator to return 2, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "skill-creator validator not found; set SKILL_CREATOR_ROOT to its skill directory.") {
		t.Fatalf("expected missing-validator diagnostic, got %q", errOut.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("uv must not run before validator preflight, stat error: %v", err)
	}
}
