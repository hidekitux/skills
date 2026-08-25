// Package hooks holds contract tests for the retained repository Git hooks
// and local-setup shell wrappers, replacing the former tests/test_local_setup.py
// coverage. The hooks under .githooks/ and the setup scripts under
// scripts/setup/ remain shell by design; these tests lock their behavior.
package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot locates the repository root (the directory containing go.mod) from
// this package's file location.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot(), rel))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}
	return string(content)
}

func TestCommitMsgHookUsesPrebuiltValidator(t *testing.T) {
	hook := readRepoFile(t, ".githooks/commit-msg")
	if !strings.Contains(hook, ".mise/bin/validate-commit-message") {
		t.Fatal("commit-msg must use the prebuilt validate-commit-message binary")
	}
	if !strings.Contains(hook, "commitlint") {
		t.Fatal("commit-msg must still invoke commitlint")
	}
	if strings.Contains(hook, "python") {
		t.Fatal("commit-msg must not depend on a Python runtime")
	}
	if strings.Contains(hook, "go run") || strings.Contains(hook, "go build") {
		t.Fatal("commit-msg must not compile code or fetch modules")
	}
}

func TestPreCommitRunsLocalChecks(t *testing.T) {
	hook := readRepoFile(t, ".githooks/pre-commit")
	if !strings.Contains(hook, "mise run check:local") {
		t.Fatalf("pre-commit must run check:local: %q", hook)
	}
}

func TestPrePushRunsFullValidation(t *testing.T) {
	hook := readRepoFile(t, ".githooks/pre-push")
	if !strings.Contains(hook, "mise run validate") {
		t.Fatalf("pre-push must run validate: %q", hook)
	}
}

func TestPostCheckoutOnlyRefreshesSetupOnBranchCheckouts(t *testing.T) {
	hook := readRepoFile(t, ".githooks/post-checkout")
	if !strings.Contains(hook, `[ "$3" = "1" ]`) {
		t.Fatalf("post-checkout must guard on the branch-checkout flag: %q", hook)
	}
	if !strings.Contains(hook, "mise run setup") {
		t.Fatalf("post-checkout must run setup: %q", hook)
	}
}

func TestSetupCommitlintBuildsAndLinksMessageValidator(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-commitlint.sh")
	if !strings.Contains(script, "./cmd/validate-commit-message") {
		t.Fatal("setup-commitlint must build the repository-local message validator")
	}
	if !strings.Contains(script, "validate-commit-message") && !strings.Contains(script, "commitlint") {
		t.Fatal("setup-commitlint must wire the validator beside commitlint")
	}
}

func TestRegisterLocalSkillsIsRevisionKeyed(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/register-local-skills.sh")
	if !strings.Contains(script, "worktree-snapshot") {
		t.Fatal("register-local-skills must stay keyed by the worktree snapshot")
	}
	for _, hostRoot := range []string{".agents/skills", ".claude/skills"} {
		if !strings.Contains(script, hostRoot) {
			t.Fatalf("register-local-skills must register to %s", hostRoot)
		}
	}
}

func TestRegisterLocalSkillsDiscoversNestedSkillsRecursively(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/register-local-skills.sh")
	if !strings.Contains(script, "find \"${source_root}\" -type f -name SKILL.md") {
		t.Fatal("register-local-skills must discover SKILL.md files recursively under skills/")
	}
	if !strings.Contains(script, `target="../../skills/${skill_rel}"`) {
		t.Fatal("register-local-skills must build the symlink target from the canonical repository-relative skill path")
	}
}

func TestSetupLocalSkillsEnablesHooks(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-local-skills.sh")
	if !strings.Contains(script, "core.hooksPath") {
		t.Fatalf("setup-local-skills must enable .githooks via core.hooksPath: %q", script)
	}
}
