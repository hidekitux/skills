package fsl

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/diagnostic"
	"github.com/hidekitux/skills/internal/provider"
	"github.com/hidekitux/skills/internal/support"
)

func TestSpecFilesCollectsRepoAndSkillSpecs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/root-a.fsl", "x")
	write(t, root, "specs/nested/b.fsl", "x")
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	write(t, root, "skills/some-skill/README.md", "x")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/nested/b.fsl", "specs/root-a.fsl", "skills/some-skill/specs/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesDedupesSymlinkedSkillSpecs(t *testing.T) {
	root := t.TempDir()
	// A skill-owned source exposed through the required repository-level symlink:
	// both paths resolve to the same physical file, so it must be collected once,
	// using the first-seen exposure path as the stable display path.
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	writeLink(t, root, "specs/some-skill/skill.fsl", "../../skills/some-skill/specs/skill.fsl")
	write(t, root, "specs/repo-owned.fsl", "x")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/repo-owned.fsl", "specs/some-skill/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesCollapsesDuplicateAliases(t *testing.T) {
	root := t.TempDir()
	// Two exposure symlinks aliasing the same physical source must collapse to one.
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	writeLink(t, root, "specs/a/skill.fsl", "../../skills/some-skill/specs/skill.fsl")
	writeLink(t, root, "specs/b/skill.fsl", "../../skills/some-skill/specs/skill.fsl")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/a/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesOrderingIsDeterministic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "skills/z-skill/specs/z.fsl", "x")
	write(t, root, "specs/a.fsl", "x")
	write(t, root, "specs/m.fsl", "x")
	writeLink(t, root, "specs/z-skill/z.fsl", "../../skills/z-skill/specs/z.fsl")

	want, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		got, err := specFiles(root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unstable ordering: got %v, want %v", got, want)
		}
	}
}

func TestSpecFilesRejectsBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	writeLink(t, root, "specs/broken.fsl", "missing-target.fsl")

	if _, err := specFiles(root); err == nil {
		t.Fatal("expected error for a broken symlink")
	}
}

func TestSpecFilesRejectsSymlinkEscapingRepository(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.fsl")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLink(t, root, "specs/escape.fsl", outside)

	if _, err := specFiles(root); err == nil {
		t.Fatal("expected error for a symlink resolving outside the repository")
	}
}

func TestSpecFilesResolvesRootThroughSymlink(t *testing.T) {
	// The repository root reached through a symlink component must not be
	// mistaken for an escape: in-repo specs resolve to a path inside the
	// canonical root and must be accepted.
	real := t.TempDir()
	wrap := filepath.Join(real, "wrap")
	if err := os.MkdirAll(wrap, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wrap, "rootlink")
	if err := os.Symlink(real, root); err != nil {
		t.Fatal(err)
	}
	write(t, root, "specs/a.fsl", "x")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatalf("in-repo spec rejected as outside the repository: %v", err)
	}
	expected := []string{"specs/a.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestVerifyFSLRejectsBrokenSymlinkBeforeRunningFslc(t *testing.T) {
	root := t.TempDir()
	writeLink(t, root, "specs/broken.fsl", "missing-target.fsl")
	var out, errOut bytes.Buffer
	if code := VerifyFSL(root, &out, &errOut); code == 0 {
		t.Fatalf("expected nonzero, got 0: out=%q err=%q", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "broken.fsl") {
		t.Fatalf("expected error to name the broken spec, got err=%q", errOut.String())
	}
}

// TestVerifyFSLResultReportsAnInterruptedVerifierAsInfrastructure keeps a
// killed or timed-out verifier separate from an invalid specification: the
// specification was never judged, so the diagnostic must stay retryable.
func TestVerifyFSLResultReportsAnInterruptedVerifierAsInfrastructure(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/invalid.fsl", "not a valid specification")
	original := fslPort
	t.Cleanup(func() { fslPort = original })
	fslPort = provider.NewFSL(&provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
		return provider.Fail(command, provider.KindInterrupted, 1, "")
	}})

	result := VerifyFSLResult(root, &bytes.Buffer{}, &bytes.Buffer{})
	if !result.Infrastructure {
		t.Fatalf("interrupted verifier result = %#v", result)
	}
	item, err := VerificationDiagnosticForResult(result)
	if err != nil {
		t.Fatal(err)
	}
	if item.Category != diagnostic.InfrastructureError || !item.Retryable || item.Code != VerificationInfrastructureCode {
		t.Fatalf("interrupted verifier diagnostic = %#v", item)
	}
}

func TestVerifyFSLResultDistinguishesInvalidSpecFromUnavailableTool(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/invalid.fsl", "not a valid specification")
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fslc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"check\" ]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FSLC_BIN_DIR", binDir)
	result := VerifyFSLResult(root, &bytes.Buffer{}, &bytes.Buffer{})
	if result.ExitCode != 1 || result.Phase != "check" || result.Infrastructure {
		t.Fatalf("invalid spec result = %#v", result)
	}
	item, err := VerificationDiagnosticForResult(result)
	if err != nil {
		t.Fatal(err)
	}
	if item.Category != diagnostic.ValidationFailure || item.Retryable || item.Code != VerificationValidationDiagnosticCode {
		t.Fatalf("invalid spec diagnostic = %#v", item)
	}

	t.Setenv("FSLC_BIN_DIR", t.TempDir())
	result = VerifyFSLResult(root, &bytes.Buffer{}, &bytes.Buffer{})
	if result.ExitCode != 1 || result.Phase != "check" || !result.Infrastructure {
		t.Fatalf("unavailable tool result = %#v", result)
	}
	item, err = VerificationDiagnosticForResult(result)
	if err != nil {
		t.Fatal(err)
	}
	if item.Category != diagnostic.InfrastructureError || !item.Retryable || item.Code != VerificationInfrastructureCode {
		t.Fatalf("unavailable tool diagnostic = %#v", item)
	}
}

// TestVerifyFSLNamesTheFailureClassInTextOutput keeps the classification in
// the default text format that mise run verify:fsl prints, not only in the
// JSON diagnostic format.
func TestVerifyFSLNamesTheFailureClassInTextOutput(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/invalid.fsl", "not a valid specification")
	infrastructure := "diagnostic: [fsl/fsl.verify.infrastructure] infrastructure_error: FSL verification tool could not run"
	validation := "diagnostic: [fsl/fsl.verify.validation] validation_failure: FSL specification failed verification"
	stub := func(kind provider.Kind) provider.FSL {
		return provider.NewFSL(&provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
			return provider.Fail(command, kind, 1, "")
		}})
	}
	original := fslPort
	t.Cleanup(func() { fslPort = original })
	cases := []struct {
		name string
		port provider.FSL
		want string
	}{
		{"absent verifier", original, infrastructure},
		{"timed out verifier", stub(provider.KindTimeout), infrastructure},
		{"interrupted verifier", stub(provider.KindInterrupted), infrastructure},
		{"rejected specification", stub(provider.KindFailure), validation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FSLC_BIN_DIR", t.TempDir())
			fslPort = tc.port
			var out, errOut bytes.Buffer
			if code := VerifyFSL(root, &out, &errOut); code != 1 {
				t.Fatalf("VerifyFSL() = %d, want 1: out=%q err=%q", code, out.String(), errOut.String())
			}
			if !strings.Contains(errOut.String(), tc.want) {
				t.Fatalf("expected %q in err=%q", tc.want, errOut.String())
			}
			if !strings.Contains(errOut.String(), "evidence=path:specs/invalid.fsl") {
				t.Fatalf("expected the failing spec in err=%q", errOut.String())
			}
		})
	}
}

func TestMutateFSLResultRetainsReportWhenMutationOutputIsInvalid(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/invalid.fsl", "not a valid specification")
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "fslc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FSLC_BIN_DIR", binDir)
	reportPath := filepath.Join(t.TempDir(), "mutation-report.json")
	result := MutateFSLResult(root, &bytes.Buffer{}, &bytes.Buffer{}, MutateOptions{ReportPath: reportPath})
	if result.ExitCode != 1 {
		t.Fatalf("expected mutation failure, got %#v", result)
	}
	if len(result.Report.Specs) != 1 || result.Report.Specs[0].Status != "error" {
		t.Fatalf("expected the current invocation error in the report, got %#v", result.Report)
	}
	diagnostics, err := DiagnosticsForReport(result.Report)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Category != diagnostic.InfrastructureError {
		t.Fatalf("expected one infrastructure diagnostic, got %#v", diagnostics)
	}
}

func TestRunFslcInvokesBinaryAtBinDir(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fslc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf 'fake-fslc-ran\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The binary must be resolved from FSLC_BIN_DIR directly, not via the
	// inherited PATH, so verify-fsl works on CI runners with no fslc on PATH.
	t.Setenv("FSLC_BIN_DIR", dir)
	var out, errOut bytes.Buffer
	if code := runFslc(&out, &errOut, "check", "spec.fsl"); code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "fake-fslc-ran") {
		t.Fatalf("expected the binary at FSLC_BIN_DIR to run, got out=%q err=%q", out.String(), errOut.String())
	}
}

func TestBinPathPrefersFSLCBinDir(t *testing.T) {
	t.Setenv("FSLC_BIN_DIR", "/custom/fslc")
	t.Setenv("SKILLS_ENVIRONMENT_ROOT", "/environment")
	if got := binPath(); got != "/custom/fslc" {
		t.Fatalf("unexpected bin path %q", got)
	}
}

// TestBinPathMatchesEnvironmentState keeps the Go rule and the rule in
// scripts/setup/environment-state.sh in agreement, because the installer
// writes where the shell rule points and the verifier reads where binPath
// points.
func TestBinPathMatchesEnvironmentState(t *testing.T) {
	repository, err := support.ResolveRoot("")
	if err != nil {
		t.Fatal(err)
	}
	setupRoot := t.TempDir()
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"local", map[string]string{"SETUP_ROOT": setupRoot}, setupRoot + "/.mise/fslc"},
		{"local checkout", map[string]string{}, repository + "/.mise/fslc"},
		{"environment root", map[string]string{"SETUP_ROOT": setupRoot, "SKILLS_ENVIRONMENT_ROOT": "/environment"}, "/environment/fslc"},
		{"CI runner", map[string]string{"SETUP_ROOT": setupRoot, "CI": "true", "RUNNER_TEMP": "/runner"}, "/runner/skills-worktree/fslc"},
		{"CI without runner", map[string]string{"SETUP_ROOT": setupRoot, "CI": "true"}, setupRoot + "/.mise/skills-worktree/fslc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"FSLC_BIN_DIR", "SETUP_ROOT", "SKILLS_ENVIRONMENT_ROOT", "CI", "RUNNER_TEMP"} {
				t.Setenv(name, tc.env[name])
			}
			if got := binPath(); got != tc.want {
				t.Fatalf("binPath() = %q, want %q", got, tc.want)
			}
			script := `source "$1/scripts/setup/environment-state.sh" && setup_environment_export && printf '%s' "$FSLC_BIN_DIR"`
			// support.ResolveRoot drops GIT_* variables, so the shell must too, or
			// a Git hook's GIT_DIR makes git rev-parse answer the working directory.
			command := exec.Command("bash", "-c", script, "bash", repository)
			command.Env = support.GitEnv()
			shell, err := command.Output()
			if err != nil {
				t.Fatalf("environment-state.sh: %v", err)
			}
			if string(shell) != tc.want {
				t.Fatalf("environment-state.sh FSLC_BIN_DIR = %q, want %q", shell, tc.want)
			}
		})
	}
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeLink(t *testing.T, root, name, target string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.FromSlash(target), full); err != nil {
		t.Fatal(err)
	}
}
