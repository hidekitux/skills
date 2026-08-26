package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hidekitux/skills/internal/support"
)

// Exit codes for cmd/evaluate. Usage errors use 3 so classification stays
// distinct from the other repository commands' flag error code of 2.
const (
	ExitOK        = 0
	ExitAssertion = 1
	ExitInfra     = 2
	ExitUsage     = 3
)

// Options configures one evaluation run.
type Options struct {
	Root       string
	Hosts      []string
	SmokeOnly  bool
	ScenarioID string
	Skills     []string
	OutputDir  string
	DryRun     bool
	Model      string
	Commit     string
	// RunnerFor substitutes host runners (tests). When nil, runnerFor(name)
	// provides the real drivers.
	RunnerFor func(name string) HostRunner
	Reviewer  RubricReviewer
}

// selectScenarios applies the scenario, skills, and smoke filters in order.
// The skills filter keeps scenarios owned by a listed skill and e2e flows
// whose stages cross a listed skill.
func selectScenarios(all []*Scenario, opts *Options) ([]*Scenario, error) {
	var selected []*Scenario
	for _, sc := range all {
		if opts.ScenarioID != "" && sc.ID != opts.ScenarioID {
			continue
		}
		if len(opts.Skills) > 0 && !scenarioTouchesSkills(sc, opts.Skills) {
			continue
		}
		if opts.SmokeOnly && !sc.Smoke {
			continue
		}
		selected = append(selected, sc)
	}
	return selected, nil
}

// scenarioTouchesSkills reports whether a scenario belongs to one of the
// skills or is an e2e flow whose stages cross one of them.
func scenarioTouchesSkills(sc *Scenario, skills []string) bool {
	for _, name := range skills {
		if sc.Skill == name {
			return true
		}
		if sc.Skill == E2ESkill {
			for _, stage := range sc.Stages {
				if stage.Skill == name {
					return true
				}
			}
		}
	}
	return false
}

// RubricReviewer produces 1-5 scores for the seven rubric dimensions from a
// scenario's guidance plus the sandbox state and transcript. A nil reviewer
// records scores as pending. Review is always read-only and bounded.
type RubricReviewer interface {
	Review(ctx context.Context, sc *Scenario, transcript, sandboxDir string) (map[string]int, error)
}

// stageFixture copies evaluations/fixtures/<key> into the sandbox.
func stageFixture(root, key, sandboxDir string) error {
	source := filepath.Join(root, fixtureBase, key)
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("fixture %q is not a directory", key)
	}
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(sandboxDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
}

// promptSHA returns the SHA-256 of every stage prompt joined by null bytes.
func promptSHA(sc *Scenario) string {
	hash := sha256.New()
	for _, prompt := range sc.prompts() {
		hash.Write([]byte(prompt))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// repoCommit returns the HEAD commit of the repository, used as provenance.
func repoCommit(root string) string {
	out, err := support.GitOutputIn(root, "rev-parse", "HEAD")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out)
}

// shouldSkip records a skipped verdict before any sandbox is created.
func shouldSkip(sc *Scenario, opts *Options) (string, bool) {
	if opts.DryRun {
		return "dry_run_no_host_execution", true
	}
	if sc.GithubSandbox && os.Getenv("EVAL_GITHUB_REPO") == "" {
		return "sandbox_repo_not_configured", true
	}
	return "", false
}

// runOne evaluates one scenario on one driver and returns its record.
func runOne(ctx context.Context, sc *Scenario, host HostRunner, opts *Options, out, errOut io.Writer) Record {
	record := Record{
		RunID:        time.Now().UTC().Format("20060102T150405Z"),
		Scenario:     sc.ID,
		Skill:        sc.Skill,
		Kind:         sc.Kind,
		Host:         host.Name(),
		Model:        opts.Model,
		Commit:       opts.Commit,
		PromptSHA:    promptSHA(sc),
		RubricReview: RubricNA,
	}
	if sc.Fixture != "" {
		record.Fixtures = []string{sc.Fixture}
	}

	if reason, skip := shouldSkip(sc, opts); skip {
		record.Verdict = VerdictSkipped
		record.SkipReason = reason
		return record
	}
	if !host.BinaryAvailable() {
		record.Verdict = VerdictSkipped
		record.SkipReason = "host_cli_unavailable"
		return record
	}

	sandboxDir, err := os.MkdirTemp("", "skills-eval-")
	if err != nil {
		record.Verdict = VerdictInfra
		record.InfraError = "cannot create sandbox: " + err.Error()
		return record
	}
	defer os.RemoveAll(sandboxDir)

	if sc.Fixture != "" {
		if err := stageFixture(opts.Root, sc.Fixture, sandboxDir); err != nil {
			record.Verdict = VerdictInfra
			record.InfraError = "fixture staging: " + err.Error()
			return record
		}
	}
	before := snapshotHashes(sandboxDir, sc.Expectations.UnchangedFiles)

	installOut := &strings.Builder{}
	if err := host.InstallSkills(ctx, opts.Root, sandboxDir, installOut, errOut); err != nil {
		record.Verdict = VerdictInfra
		record.InfraError = "skill installation: " + err.Error()
		return record
	}

	var transcript strings.Builder
	prompts := sc.prompts()
	for index, prompt := range prompts {
		if err := host.Run(ctx, sandboxDir, prompt, &transcript); err != nil {
			record.Verdict = VerdictInfra
			record.InfraError = fmt.Sprintf("host stage %d: %v", index+1, err)
			return record
		}
	}
	correctionsUsed := runCorrections(ctx, sc, host, sandboxDir, &transcript)
	record.CorrectionsUsed = correctionsUsed

	failures := evaluateAssertions(ctx, sc, transcript.String(), sandboxDir, before)
	if len(failures) > 0 {
		record.Verdict = VerdictFail
		record.Failures = failures
		record.RubricReview = RubricNA
		return record
	}
	record.Verdict = VerdictPass
	if opts.Reviewer != nil {
		scores, err := opts.Reviewer.Review(ctx, sc, transcript.String(), sandboxDir)
		if err != nil {
			record.InfraError = "rubric review: " + err.Error()
			record.RubricReview = RubricPending
			return record
		}
		record.RubricScores = scores
		record.RubricReview = RubricComplete
	}
	return record
}

// runCorrections feeds the scenario's scripted user correction turns into
// single-stage scenarios, re-running the stage after each turn and counting
// how many were needed. Multi-stage flows record 0 and leave corrections to
// the reviewer so the flow's handoffs stay contiguous.
func runCorrections(ctx context.Context, sc *Scenario, host HostRunner, sandboxDir string, transcript *strings.Builder) int {
	if len(sc.Corrections) == 0 || len(sc.prompts()) > 1 {
		return 0
	}
	used := 0
	for _, correction := range sc.Corrections {
		used++
		fmt.Fprintf(transcript, "\n--- user correction %d ---\n%s\n", used, correction)
		if err := host.Run(ctx, sandboxDir, correction, transcript); err != nil {
			break
		}
	}
	return used
}

// gateVerdict aggregates the per-driver records of one scenario under the
// local either-pass policy (Issue 173 decision record): the scenario passes
// when at least one driver produced a deterministic pass; deterministic
// failures block only when no driver passed; infrastructure errors surface
// when neither driver passed nor failed. The per-driver records keep every
// finding visible.
func gateVerdict(records []Record) (string, string) {
	var pass, fail, infra bool
	for _, record := range records {
		switch record.Verdict {
		case VerdictPass:
			pass = true
		case VerdictFail:
			fail = true
		case VerdictInfra:
			infra = true
		}
	}
	if pass {
		return VerdictPass, ""
	}
	if fail {
		return VerdictFail, "deterministic failure on every driver that ran"
	}
	if infra {
		return VerdictInfra, "no driver produced a usable verdict (infrastructure errors everywhere)"
	}
	return VerdictSkipped, ""
}

// Run evaluates the selected scenarios for the configured drivers and writes
// machine-readable JSONL and human-readable Markdown reports into the output
// directory. Drivers run concurrently per scenario; the aggregate gate uses
// the either-pass policy. It returns the aggregate exit code.
func Run(ctx context.Context, opts *Options, out, errOut io.Writer) int {
	scenarios, err := LoadAllScenarios(opts.Root)
	if err != nil {
		fmt.Fprintf(errOut, "evaluate: %v\n", err)
		return ExitUsage
	}
	if opts.ScenarioID != "" {
		selected, err := ScenarioByID(scenarios, opts.ScenarioID)
		if err != nil {
			fmt.Fprintf(errOut, "evaluate: %v\n", err)
			return ExitUsage
		}
		scenarios = []*Scenario{selected}
	} else {
		scenarios, err = selectScenarios(scenarios, opts)
		if err != nil {
			fmt.Fprintf(errOut, "evaluate: %v\n", err)
			return ExitUsage
		}
	}
	if len(scenarios) == 0 {
		fmt.Fprintln(errOut, "evaluate: no scenarios selected")
		return ExitUsage
	}

	opts.Commit = repoCommit(opts.Root)
	if opts.Model == "" {
		opts.Model = resolveModel(opts.Root)
	}
	// Pin the evaluation-level tier model for the tier-driven opencode driver
	// so the invoked CLI never falls back to an unspecified default. The
	// codex, claude-code, and antigravity drivers keep their own fixed
	// defaults; per-driver EVAL_*_MODEL overrides win everywhere.
	for _, name := range opts.Hosts {
		if name != HostOpenCode {
			continue
		}
		envVar := modelEnvVars[name]
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, effectiveModel(name, opts.Model))
		}
	}

	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
			fmt.Fprintf(errOut, "evaluate: cannot create output directory: %v\n", err)
			return ExitInfra
		}
	}

	runID := time.Now().UTC().Format("20060102T150405Z")
	var records []Record
	gates := map[string]string{}
	for _, sc := range scenarios {
		results := make([]Record, len(opts.Hosts))
		var wg sync.WaitGroup
		for index, hostName := range opts.Hosts {
			host := runnerFor(hostName)
			if opts.RunnerFor != nil {
				host = opts.RunnerFor(hostName)
			}
			// Record the driver's effective model so provenance matches the
			// model actually invoked.
			optsCopy := *opts
			optsCopy.Model = effectiveModel(hostName, opts.Model)
			wg.Add(1)
			go func(host HostRunner) {
				defer wg.Done()
				results[index] = runOne(ctx, sc, host, &optsCopy, out, errOut)
			}(host)
		}
		wg.Wait()
		for _, record := range results {
			record.RunID = runID
			records = append(records, record)
		}
		verdict, detail := gateVerdict(results)
		gates[sc.ID] = verdict
		fmt.Fprintf(out, "scenario %s: gate=%s %s\n", sc.ID, verdict, detail)
	}

	if opts.OutputDir != "" {
		jsonlPath := filepath.Join(opts.OutputDir, runID+".jsonl")
		if err := writeJSONL(jsonlPath, records); err != nil {
			fmt.Fprintf(errOut, "evaluate: cannot write JSONL report: %v\n", err)
			return ExitInfra
		}
		markdownPath := filepath.Join(opts.OutputDir, runID+".md")
		f, err := os.Create(markdownPath)
		if err != nil {
			fmt.Fprintf(errOut, "evaluate: cannot write Markdown report: %v\n", err)
			return ExitInfra
		}
		markdownSummary(f, records, gates, opts.Model, opts.Commit)
		if err := f.Close(); err != nil {
			fmt.Fprintf(errOut, "evaluate: cannot finalize Markdown report: %v\n", err)
			return ExitInfra
		}
		fmt.Fprintf(out, "reports written to %s\n", opts.OutputDir)
	}

	hadFail := false
	hadInfra := false
	for _, verdict := range gates {
		switch verdict {
		case VerdictFail:
			hadFail = true
		case VerdictInfra:
			hadInfra = true
		}
	}
	switch {
	case hadFail:
		return ExitAssertion
	case hadInfra:
		return ExitInfra
	default:
		return ExitOK
	}
}
