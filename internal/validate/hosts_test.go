package validate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func hostInstallFixture(t *testing.T, mode string) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeVFile(t, root, "skills/plan-issue/SKILL.md", "---\nname: plan-issue\n---\n")
	writeVFile(t, root, "skills/skills/refactor-code/SKILL.md", "---\nname: refactor-code\n---\n")

	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "gh.log")
	writeExecutable(t, bin, "gh", `#!/bin/sh
set -eu

root="$3"
agent=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --agent)
      agent="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
case "$agent" in
  codex) install_root="$PWD/.agents/skills" ;;
  claude-code) install_root="$PWD/.claude/skills" ;;
  *) exit 91 ;;
esac
mkdir -p "$install_root"
find "$root/skills" -type f -name SKILL.md | sort | while IFS= read -r manifest; do
  skill_dir=$(dirname "$manifest")
  skill_name=$(basename "$skill_dir")
  relative_dir=${skill_dir#"$root/skills/"}
  case "$HOST_TEST_MODE:$skill_name" in
    missing:refactor-code) continue ;;
    broken:refactor-code) target="$root/skills/does-not-exist" ;;
    *) target="$root/skills/$relative_dir" ;;
  esac
  ln -s "$target" "$install_root/$skill_name"
done
printf '%s|%s|%s\n' "$agent" "$root" "$PWD" >> "$HOST_TEST_LOG"
`)
	t.Setenv("HOST_TEST_MODE", mode)
	t.Setenv("HOST_TEST_LOG", logPath)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMPDIR", t.TempDir())
	return root, logPath
}

func runHosts(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckHosts(root, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCheckHostsAcceptsCompleteFlatAndNamespacedInstallation(t *testing.T) {
	root, logPath := hostInstallFixture(t, "complete")

	code, out, errOut := runHosts(t, root)
	if code != 0 {
		t.Fatalf("expected complete installation to pass, got %d: %s", code, errOut)
	}
	for _, want := range []string{
		"codex installation validated: 2 skill(s) in .agents/skills.",
		"claude-code installation validated: 2 skill(s) in .claude/skills.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q: %s", want, out)
		}
	}
	if errOut != "" {
		t.Fatalf("expected no error output, got %q", errOut)
	}

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(lines) != 2 || !strings.Contains(string(log), "codex|") || !strings.Contains(string(log), "claude-code|") {
		t.Fatalf("unexpected gh calls: %q", log)
	}
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) != 3 {
			t.Fatalf("unexpected gh call record: %q", line)
		}
		tmpRoot, err := filepath.EvalSymlinks(strings.TrimRight(os.Getenv("TMPDIR"), string(os.PathSeparator)))
		if err != nil {
			t.Fatal(err)
		}
		tmpRoot += string(os.PathSeparator)
		if !strings.HasPrefix(fields[2], tmpRoot) {
			t.Fatalf("gh ran outside the temporary directory: %q (TMPDIR=%q)", fields[2], tmpRoot)
		}
	}
}

func TestCheckHostsRejectsMissingRegistration(t *testing.T) {
	root, _ := hostInstallFixture(t, "missing")

	code, _, errOut := runHosts(t, root)
	if code != 1 {
		t.Fatalf("expected missing registration to fail with 1, got %d: %s", code, errOut)
	}
	if !strings.Contains(errOut, "codex did not install refactor-code at .agents/skills.") {
		t.Fatalf("expected missing-registration diagnostic, got %q", errOut)
	}
}

func TestCheckHostsRejectsUnresolvedRegistration(t *testing.T) {
	root, _ := hostInstallFixture(t, "broken")

	code, _, errOut := runHosts(t, root)
	if code != 1 {
		t.Fatalf("expected unresolved registration to fail with 1, got %d: %s", code, errOut)
	}
	if !strings.Contains(errOut, "codex did not install refactor-code at .agents/skills.") {
		t.Fatalf("expected unresolved-registration diagnostic, got %q", errOut)
	}
}
