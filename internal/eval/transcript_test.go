package eval

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/provider"
)

// partialHost writes part of a transcript and then fails its host stage.
type partialHost struct{ *fakeHost }

func (h *partialHost) Run(ctx context.Context, sandboxDir, prompt string, out io.Writer) error {
	io.WriteString(out, "partial output before the failure")
	return errors.New("host stage failed")
}

func transcriptScenario() *Scenario {
	return &Scenario{
		ID: "plan-issue-success", Skill: "plan-issue", Kind: KindPositive,
		Title: "Plan a ready issue", Prompt: "Produce an ordered plan for the ready issue before coding.",
		Expectations: Expectations{Handoff: "implement-issue", TranscriptMust: []string{"implement-issue"}},
		Rubric:       fullRubric(),
	}
}

func readTranscript(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("transcript %s was not written: %v", path, err)
	}
	return string(content)
}

func TestRunWritesTranscriptsNextToTheReport(t *testing.T) {
	for _, tc := range []struct {
		name, line, want string
	}{
		{"passing scenario", "handing to implement-issue", "handing to implement-issue"},
		{"failing scenario", "done", "done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc := transcriptScenario()
			root := scaffoldEval(t, []map[string]string{skillEntry("plan-issue", "experimental")}, []*Scenario{sc}, nil)
			outputDir := t.TempDir()
			opts := &Options{Root: root, Hosts: []string{"codex"}, OutputDir: outputDir, RunID: "run-x",
				RunnerFor: func(name string) provider.HostCLI {
					return &fakeHost{name: name, available: true, line: tc.line}
				}}
			var out, errOut bytes.Buffer
			Run(context.Background(), opts, &out, &errOut)
			got := readTranscript(t, filepath.Join(outputDir, "run-x", "transcripts", "codex", "plan-issue-success.txt"))
			if !strings.Contains(got, tc.want) {
				t.Fatalf("transcript = %q, want it to contain %q", got, tc.want)
			}
			if _, err := os.Stat(filepath.Join(root, "evaluations", "evidence")); !os.IsNotExist(err) {
				t.Fatalf("evaluations/evidence exists after a run without --retain: %v", err)
			}
		})
	}
}

func TestRunOneWritesPartialTranscriptOnFailedHostStage(t *testing.T) {
	dir := t.TempDir()
	host := &partialHost{fakeHost: &fakeHost{name: "codex", available: true}}
	record := runOneForTest(t, transcriptScenario(), host, &Options{TranscriptDir: dir})
	if record.Verdict != VerdictInfra {
		t.Fatalf("verdict = %s, want %s", record.Verdict, VerdictInfra)
	}
	got := readTranscript(t, filepath.Join(dir, "codex", "plan-issue-success.txt"))
	if !strings.Contains(got, "partial output before the failure") {
		t.Fatalf("transcript = %q, want the partial output", got)
	}
}

// cancelingHost writes part of a transcript, then interrupts the run.
type cancelingHost struct {
	*fakeHost
	cancel context.CancelFunc
}

func (h *cancelingHost) Run(ctx context.Context, sandboxDir, prompt string, out io.Writer) error {
	io.WriteString(out, "partial output before the interruption")
	h.cancel()
	return ctx.Err()
}

func TestRunOneWritesPartialTranscriptOnInterruption(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	host := &cancelingHost{fakeHost: &fakeHost{name: "codex", available: true}, cancel: cancel}
	record := runOne(ctx, transcriptScenario(), host, &Options{TranscriptDir: dir}, io.Discard, io.Discard)
	if record.Verdict != VerdictInterrupted {
		t.Fatalf("verdict = %s, want %s", record.Verdict, VerdictInterrupted)
	}
	got := readTranscript(t, filepath.Join(dir, "codex", "plan-issue-success.txt"))
	if !strings.Contains(got, "partial output before the interruption") {
		t.Fatalf("transcript = %q, want the partial output", got)
	}
}

func TestRunOneWritesNoTranscriptWithoutDirectory(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	runOneForTest(t, transcriptScenario(), &fakeHost{name: "codex", available: true, line: "done"}, &Options{})
	entries, err := os.ReadDir(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("files written without a transcript directory: %v", entries)
	}
}

func TestDeliberationWritesOneTranscriptPerRole(t *testing.T) {
	dir := t.TempDir()
	scenario := &Scenario{
		ID: "deliberation", Skill: "debug-code", Kind: KindPositive,
		Prompt:       "Investigate the reported failure and hand off the verified result.",
		Deliberation: deliberationForTest(),
		Expectations: Expectations{Handoff: "write-tests", TranscriptMust: []string{"write-tests"}},
	}
	runOneForTest(t, scenario, &recordingHost{name: "codex", available: true}, &Options{Commit: "test-commit", TranscriptDir: dir})
	for _, name := range []string{"deliberation.baseline.txt", "deliberation.candidate-1.txt", "deliberation.candidate-2.txt"} {
		readTranscript(t, filepath.Join(dir, "codex", name))
	}
}

func TestWriteTranscriptReportsWriteFailure(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	writeTranscript(filepath.Join(blocker, "transcripts"), "s", "codex", "", "text", &errOut)
	if !strings.Contains(errOut.String(), "cannot write transcript") {
		t.Fatalf("errOut = %q, want the write failure", errOut.String())
	}
}
