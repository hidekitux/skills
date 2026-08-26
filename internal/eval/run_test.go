package eval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fakeHost is a deterministic HostRunner for tests: it emits a fixed
// transcript line, writes optional files into the sandbox, and counts runs.
type fakeHost struct {
	name       string
	available  bool
	installErr error
	runErr     error
	line       string
	files      map[string]string
	calls      int
}

func (f *fakeHost) Name() string          { return f.name }
func (f *fakeHost) BinaryAvailable() bool { return f.available }
func (f *fakeHost) CallCount() int        { return f.calls }
func (f *fakeHost) InstallSkills(ctx context.Context, root, sandboxDir string, out, errOut io.Writer) error {
	return f.installErr
}

func (f *fakeHost) Run(ctx context.Context, sandboxDir, prompt string, out io.Writer) error {
	f.calls++
	if f.runErr != nil {
		return f.runErr
	}
	io.WriteString(out, f.line)
	for rel, content := range f.files {
		path := filepath.Join(sandboxDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func passingFake(name string) *fakeHost {
	return &fakeHost{
		name:      name,
		available: true,
		line:      "reproduced the failure and handing the verified fix to write-tests",
		files:     map[string]string{"fix.txt": "fixed"},
	}
}

func runOneForTest(t *testing.T, sc *Scenario, host HostRunner, opts *Options) Record {
	t.Helper()
	record := runOne(context.Background(), sc, host, opts, io.Discard, io.Discard)
	return record
}

func TestRunOnePassesOnMatchingAssertions(t *testing.T) {
	sc := &Scenario{
		ID:     "debug-code-success",
		Skill:  "debug-code",
		Kind:   KindPositive,
		Prompt: "The leap-year check fails for 1900; debug the failure.",
		Expectations: Expectations{
			Handoff:        "write-tests",
			TranscriptMust: []string{"write-tests"},
			FilesMustExist: []string{"fix.txt"},
		},
	}
	host := passingFake("codex")
	record := runOneForTest(t, sc, host, &Options{Commit: "test-commit"})
	if record.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, failures = %v", record.Verdict, record.Failures)
	}
	if record.PromptSHA == "" || record.Commit == "" {
		t.Fatalf("provenance missing: prompt %q commit %q", record.PromptSHA, record.Commit)
	}
}

func TestRunOneSkipsWhenBinaryUnavailable(t *testing.T) {
	sc := &Scenario{ID: "x", Skill: "debug-code", Kind: KindPositive, Prompt: "p"}
	host := &fakeHost{name: "codex", available: false}
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictSkipped || record.SkipReason != "host_cli_unavailable" {
		t.Fatalf("verdict = %s (%s), want skipped host_cli_unavailable", record.Verdict, record.SkipReason)
	}
}

func TestRunOneSkipsGithubSandboxWithoutRepo(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "")
	sc := &Scenario{ID: "x", Skill: "create-issue", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	host := passingFake("codex")
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictSkipped || record.SkipReason != "sandbox_repo_not_configured" {
		t.Fatalf("verdict = %s (%s), want skipped sandbox_repo_not_configured", record.Verdict, record.SkipReason)
	}
	if host.CallCount() != 0 {
		t.Fatalf("host should not run when skipped, ran %d times", host.CallCount())
	}
}

func TestRunOneInfraOnStageRunError(t *testing.T) {
	sc := &Scenario{ID: "x", Skill: "debug-code", Kind: KindPositive, Prompt: "p"}
	host := &fakeHost{name: "codex", available: true, runErr: fmt.Errorf("host crashed")}
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictInfra || !strings.Contains(record.InfraError, "host stage 1") {
		t.Fatalf("verdict = %s (%s), want infrastructure error", record.Verdict, record.InfraError)
	}
}

func TestRunOneInfraOnSkillInstall(t *testing.T) {
	sc := &Scenario{ID: "x", Skill: "debug-code", Kind: KindPositive, Prompt: "p"}
	host := &fakeHost{name: "codex", available: true, installErr: fmt.Errorf("gh unavailable")}
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictInfra || !strings.Contains(record.InfraError, "skill installation") {
		t.Fatalf("verdict = %s (%s), want skill installation error", record.Verdict, record.InfraError)
	}
}

func TestRunOneAppliesCorrections(t *testing.T) {
	sc := &Scenario{
		ID:          "x",
		Skill:       "plan-issue",
		Kind:        KindPositive,
		Prompt:      "Plan the change.",
		Corrections: []string{"That is out of scope; replan."},
		Expectations: Expectations{
			Handoff:        "implement-issue",
			TranscriptMust: []string{"implement-issue"},
		},
	}
	host := &fakeHost{name: "codex", available: true, line: "planning, handoff to implement-issue"}
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, failures = %v", record.Verdict, record.Failures)
	}
	if record.CorrectionsUsed != 1 || host.CallCount() != 2 {
		t.Fatalf("corrections used = %d, host runs = %d, want 1 and 2", record.CorrectionsUsed, host.CallCount())
	}
}

func TestRunAggregateReturnsExpectedExitCodes(t *testing.T) {
	sc := &Scenario{
		ID:     "plan-issue-success",
		Skill:  "plan-issue",
		Kind:   KindPositive,
		Title:  "Plan a ready issue",
		Prompt: "Produce an ordered plan for the ready issue before coding.",
		Expectations: Expectations{
			Handoff:        "implement-issue",
			TranscriptMust: []string{"implement-issue"},
		},
		Rubric: fullRubric(),
	}
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{sc},
		nil,
	)

	t.Run("pass", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"codex"},
			RunnerFor: func(name string) HostRunner {
				return &fakeHost{name: name, available: true, line: "handing to implement-issue"}
			}}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitOK {
			t.Fatalf("exit = %d, want 0\n%s%s", code, out.String(), errOut.String())
		}
	})

	t.Run("assertion failure", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"codex"},
			RunnerFor: func(name string) HostRunner { return &fakeHost{name: name, available: true, line: "done"} }}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitAssertion {
			t.Fatalf("exit = %d, want 1", code)
		}
	})

	t.Run("dry run skips every scenario", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"codex"}, DryRun: true}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitOK {
			t.Fatalf("exit = %d, want 0", code)
		}
		if !strings.Contains(out.String(), "gate=skipped") {
			t.Fatalf("dry run did not report skip gate:\n%s", out.String())
		}
	})

	t.Run("reports written on request", func(t *testing.T) {
		outputDir := t.TempDir()
		opts := &Options{Root: root, Hosts: []string{"codex"}, OutputDir: outputDir,
			RunnerFor: func(name string) HostRunner {
				return &fakeHost{name: name, available: true, line: "handing to implement-issue"}
			}}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitOK {
			t.Fatalf("exit = %d, want 0", code)
		}
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			t.Fatal(err)
		}
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		if len(names) != 2 {
			t.Fatalf("expected JSONL and Markdown reports, got %v", names)
		}
		if !strings.HasSuffix(names[0], ".jsonl") || !strings.HasSuffix(names[1], ".md") {
			t.Fatalf("unexpected report names: %v", names)
		}
	})
}

// TestRunUsesEitherPassPolicyAcrossDrivers verifies the local either-pass
// aggregation: each scenario passes when at least one of the concurrently
// running drivers produces a deterministic pass.
func TestRunUsesEitherPassPolicyAcrossDrivers(t *testing.T) {
	sc := &Scenario{
		ID:     "plan-issue-success",
		Skill:  "plan-issue",
		Kind:   KindPositive,
		Title:  "Plan a ready issue",
		Prompt: "Produce an ordered plan for the ready issue before coding.",
		Expectations: Expectations{
			Handoff:        "implement-issue",
			TranscriptMust: []string{"implement-issue"},
		},
		Rubric: fullRubric(),
	}
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{sc},
		nil,
	)

	t.Run("one pass and one fail still passes the gate", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"a", "b"},
			RunnerFor: func(name string) HostRunner {
				if name == "a" {
					return &fakeHost{name: name, available: true, line: "done without handoff"}
				}
				return &fakeHost{name: name, available: true, line: "handing to implement-issue"}
			}}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitOK {
			t.Fatalf("exit = %d, want 0 (either-pass)\n%s%s", code, out.String(), errOut.String())
		}
		if !strings.Contains(out.String(), "gate=pass") {
			t.Fatalf("missing pass gate:\n%s", out.String())
		}
	})

	t.Run("all drivers fail blocks", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"a", "b"},
			RunnerFor: func(name string) HostRunner {
				return &fakeHost{name: name, available: true, line: "done without handoff"}
			}}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitAssertion {
			t.Fatalf("exit = %d, want 1", code)
		}
	})

	t.Run("all drivers infra reports infrastructure error", func(t *testing.T) {
		opts := &Options{Root: root, Hosts: []string{"a", "b"},
			RunnerFor: func(name string) HostRunner {
				return &fakeHost{name: name, available: true, runErr: fmt.Errorf("crashed")}
			}}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), opts, &out, &errOut); code != ExitInfra {
			t.Fatalf("exit = %d, want 2", code)
		}
	})
}

func TestSelectScenariosBySkills(t *testing.T) {
	plan := baseScenario()
	boundary := &Scenario{ID: "plan-issue-boundary", Skill: "plan-issue", Kind: KindBoundary, Title: "B", Prompt: "p", Expectations: Expectations{Handoff: "ask"}, Rubric: fullRubric()}
	debug := &Scenario{ID: "debug-success", Skill: "debug-code", Kind: KindPositive, Title: "D", Prompt: "p", Expectations: Expectations{Handoff: "write-tests"}, Rubric: fullRubric()}
	e2e := &Scenario{ID: "e2e-debug", Skill: E2ESkill, Kind: KindPositive, Title: "E", Prompt: "",
		Stages:       []Stage{{Skill: "debug-code", Prompt: "p"}},
		Expectations: Expectations{Handoff: "implement-issue"}, Rubric: fullRubric()}
	all := []*Scenario{plan, boundary, debug, e2e}

	selected, err := selectScenarios(all, &Options{Skills: []string{"plan-issue"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].ID != "plan-issue-success" || selected[1].ID != "plan-issue-boundary" {
		t.Fatalf("unexpected selection: %v", ids(selected))
	}

	selected, err = selectScenarios(all, &Options{Skills: []string{"debug-code"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].ID != "debug-success" || selected[1].ID != "e2e-debug" {
		t.Fatalf("e2e crossing the skill should be selected: %v", ids(selected))
	}

	selected, err = selectScenarios(all, &Options{SmokeOnly: true, Skills: []string{"plan-issue"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 0 {
		t.Fatalf("unexpected smoke+skills intersection: %v", ids(selected))
	}
}

func ids(scenarios []*Scenario) []string {
	var names []string
	for _, sc := range scenarios {
		names = append(names, sc.ID)
	}
	return names
}

func TestGateVerdictEitherPass(t *testing.T) {
	record := func(scenario, verdict, host string) Record {
		return Record{Scenario: scenario, Skill: "x", Kind: KindPositive, Host: host, Verdict: verdict}
	}
	cases := []struct {
		name    string
		records []Record
		want    string
	}{
		{"one pass overrides fail", []Record{record("s", VerdictFail, "a"), record("s", VerdictPass, "b")}, VerdictPass},
		{"one pass overrides infra", []Record{record("s", VerdictInfra, "a"), record("s", VerdictPass, "b")}, VerdictPass},
		{"all fail blocks", []Record{record("s", VerdictFail, "a"), record("s", VerdictFail, "b")}, VerdictFail},
		{"fail overrides infra", []Record{record("s", VerdictInfra, "a"), record("s", VerdictFail, "b")}, VerdictFail},
		{"all infra", []Record{record("s", VerdictInfra, "a"), record("s", VerdictInfra, "b")}, VerdictInfra},
		{"all skipped", []Record{record("s", VerdictSkipped, "a"), record("s", VerdictSkipped, "b")}, VerdictSkipped},
	}
	for _, tc := range cases {
		got, _ := gateVerdict(tc.records)
		if got != tc.want {
			t.Fatalf("%s: gateVerdict = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestCommandReviewerParsesScores(t *testing.T) {
	reviewer := &CommandReviewer{Command: `printf '{"trigger_selection":4,"task_completion":5,"evidence_quality":4,"scope_control":5,"safety":5,"user_correction_count":5,"handoff_quality":4}'`}
	sc := &Scenario{ID: "x", Skill: "plan-issue", Kind: KindPositive, Prompt: "p"}
	scores, err := reviewer.Review(context.Background(), sc, "transcript", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(scores) != 7 || scores["task_completion"] != 5 {
		t.Fatalf("unexpected scores: %v", scores)
	}
}

func TestCommandReviewerRejectsOutOfRangeScore(t *testing.T) {
	reviewer := &CommandReviewer{Command: `printf '{"safety":9}'`}
	sc := &Scenario{ID: "x", Skill: "plan-issue", Kind: KindPositive, Prompt: "p"}
	if _, err := reviewer.Review(context.Background(), sc, "t", t.TempDir()); err == nil {
		t.Fatal("expected out-of-range score to be rejected")
	}
}

func TestResolveHosts(t *testing.T) {
	all, err := ResolveHosts("all")
	if err != nil || len(all) != 4 {
		t.Fatalf("ResolveHosts(all) = %v, %v", all, err)
	}
	pair, err := ResolveHosts("opencode,antigravity")
	if err != nil || len(pair) != 2 || pair[0] != HostOpenCode || pair[1] != HostAntigravity {
		t.Fatalf("ResolveHosts(comma list) = %v, %v", pair, err)
	}
	if _, err := ResolveHosts("bogus"); err == nil {
		t.Fatal("expected bogus host to fail")
	}
	if _, err := ResolveHosts(""); err == nil {
		t.Fatal("expected empty host to fail")
	}
}
