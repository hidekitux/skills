package provider

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestClaudeStreamWriterKeepsOnlyAgentOutput(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"system","subtype":"init","skills":["plan-issue","review-pr"]}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Implementing the fix is out of scope."},{"type":"tool_use","name":"Bash","input":{"command":"gh issue create --body '## Acceptance criteria'"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","content":"SKILL.md mentions fix-pr"}]}}`,
		`Error: a line that is not a stream event`,
		`{"type":"result","subtype":"success","result":"Next step is plan-issue."}`,
	}, "\n")
	var out bytes.Buffer
	writer := newClaudeStreamWriter(&out)
	// Split the input mid-line to exercise buffering.
	half := len(stream) / 2
	if _, err := writer.Write([]byte(stream[:half])); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(stream[half:])); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	transcript := out.String()
	for _, want := range []string{
		"Implementing the fix is out of scope.",
		"[tool Bash]",
		"## Acceptance criteria",
		"Error: a line that is not a stream event",
	} {
		if !strings.Contains(transcript, want) {
			t.Errorf("transcript lacks %q:\n%s", want, transcript)
		}
	}
	for _, unwanted := range []string{"review-pr", "fix-pr", "Next step is plan-issue."} {
		if strings.Contains(transcript, unwanted) {
			t.Errorf("transcript contains %q, which the agent did not produce in a kept event:\n%s", unwanted, transcript)
		}
	}
}

func TestClaudeStreamWriterKeepsErrorResultWithSuccessSubtype(t *testing.T) {
	var out bytes.Buffer
	writer := newClaudeStreamWriter(&out)
	line := `{"type":"result","subtype":"success","is_error":true,"result":"Failed to authenticate: OAuth session expired"}` + "\n"
	if _, err := writer.Write([]byte(line)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Failed to authenticate") {
		t.Fatalf("transcript = %q, want the error result text", out.String())
	}
}

func TestClaudeCodeRunConvertsStreamUnderCommandOverride(t *testing.T) {
	for _, override := range []string{"", "wrapper claude -p --output-format stream-json --verbose"} {
		t.Setenv("EVAL_CLAUDE_CMD", override)
		var stdout io.Writer
		stub := &Stub{Handler: func(command Command) (Result, error) {
			stdout = command.Stdout
			return Result{}, nil
		}}
		if err := NewHostCLI(HostClaudeCode, stub).Run(context.Background(), t.TempDir(), "prompt", io.Discard); err != nil {
			t.Fatal(err)
		}
		if _, ok := stdout.(*claudeStreamWriter); !ok {
			t.Fatalf("EVAL_CLAUDE_CMD=%q: stdout = %T, want the stream writer", override, stdout)
		}
	}
}

func TestClaudeStreamWriterKeepsFailedResult(t *testing.T) {
	var out bytes.Buffer
	writer := newClaudeStreamWriter(&out)
	if _, err := writer.Write([]byte(`{"type":"result","subtype":"error_max_turns","result":"stopped early"}`)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "stopped early") {
		t.Fatalf("transcript = %q, want the failed result text", out.String())
	}
}
