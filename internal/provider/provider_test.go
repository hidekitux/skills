package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRedactRemovesCredentialsAndPrivateURLs(t *testing.T) {
	cleaned := Redact("fatal: ghp_0123456789abcdef rejected by https://build.local/status")
	if strings.Contains(cleaned, "ghp_0123456789abcdef") {
		t.Fatalf("credential survived redaction: %q", cleaned)
	}
	if strings.Contains(cleaned, "build.local") {
		t.Fatalf("private URL survived redaction: %q", cleaned)
	}
	public := Redact("see https://github.com/hidekitux/skills for the tag")
	if !strings.Contains(public, "https://github.com/hidekitux/skills") {
		t.Fatalf("public URL must stay readable: %q", public)
	}
}

func TestRedactKeepsTheFailingTailWithinTheLimit(t *testing.T) {
	cleaned := Redact(strings.Repeat("a", detailLimit+50) + "END")
	if len([]rune(cleaned)) > detailLimit+3 {
		t.Fatalf("detail length = %d, want at most %d", len([]rune(cleaned)), detailLimit+3)
	}
	if !strings.HasSuffix(cleaned, "END") {
		t.Fatalf("detail must keep the tail: %q", cleaned)
	}
}

func TestOSRunnerClassifiesEveryFailureKind(t *testing.T) {
	runner := OSRunner{}

	if _, err := runner.Run(context.Background(), Command{
		Port: PortShell, Operation: "script", Name: "sh", Args: []string{"-c", "exit 3"},
	}); err == nil {
		t.Fatal("expected a failure")
	} else {
		var providerErr *Error
		if !errors.As(err, &providerErr) || providerErr.Kind != KindFailure || providerErr.ExitCode != 3 {
			t.Fatalf("failure classification = %+v", err)
		}
	}

	if _, err := runner.Run(context.Background(), Command{
		Port: PortShell, Operation: "script", Name: "sh", Args: []string{"-c", "sleep 5"},
		Timeout: 50 * time.Millisecond,
	}); err == nil {
		t.Fatal("expected a timeout")
	} else if kind, _ := KindOf(err); kind != KindTimeout {
		t.Fatalf("timeout classification = %v", kind)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if _, err := runner.Run(cancelled, Command{
		Port: PortShell, Operation: "script", Name: "sh", Args: []string{"-c", "sleep 5"},
	}); err == nil {
		t.Fatal("expected an interruption")
	} else if kind, _ := KindOf(err); kind != KindInterrupted {
		t.Fatalf("interruption classification = %v", kind)
	}

	if _, err := runner.Run(context.Background(), Command{
		Port: PortTool, Operation: "run", Name: "skills-provider-absent-binary",
	}); !IsUnavailable(err) {
		t.Fatalf("unavailable classification = %v", err)
	}
}

func TestOSRunnerSeparatesAndStreamsOutput(t *testing.T) {
	// One writer receives both streams, which is what internal/eval passes for
	// a host stage transcript. Run it repeatedly so the race detector observes
	// the two copy goroutines.
	for attempt := 0; attempt < 20; attempt++ {
		var streamed strings.Builder
		result, err := OSRunner{}.Run(context.Background(), Command{
			Port: PortShell, Operation: "script", Name: "sh",
			Args:   []string{"-c", "printf out; printf err 1>&2"},
			Stdout: &streamed, Stderr: &streamed,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Stdout != "out" || result.Stderr != "err" {
			t.Fatalf("result = %+v", result)
		}
		if len(result.Combined) != 6 || streamed.Len() != 6 {
			t.Fatalf("combined = %q, streamed = %q", result.Combined, streamed.String())
		}
	}
}

func TestOSRunnerPassesStdinAndDirectory(t *testing.T) {
	dir := t.TempDir()
	result, err := OSRunner{}.Run(context.Background(), Command{
		Port: PortShell, Operation: "script", Name: "sh", Args: []string{"-c", "cat; pwd"},
		Dir: dir, Stdin: "payload\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Stdout, "payload\n") {
		t.Fatalf("stdin was not passed: %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, dir) {
		t.Fatalf("working directory was not applied: %q", result.Stdout)
	}
}

func TestProviderErrorStaysDistinguishableFromAValidationFailure(t *testing.T) {
	if _, ok := KindOf(errors.New("SKILL.md is missing a description")); ok {
		t.Fatal("a product validation failure must not classify as a provider failure")
	}
	exhausted := RetryExhausted(PortGitHub, "api", 3, errors.New("rate limited"))
	if kind, ok := KindOf(exhausted); !ok || kind != KindRetryExhausted {
		t.Fatalf("retry exhaustion = %v", kind)
	}
	if !strings.Contains(exhausted.Error(), "3 attempts") {
		t.Fatalf("retry exhaustion detail = %q", exhausted.Error())
	}
}

func TestErrorMessageNeverCarriesACredential(t *testing.T) {
	_, err := OSRunner{}.Run(context.Background(), Command{
		Port: PortShell, Operation: "script", Name: "sh",
		Args: []string{"-c", "echo token=ghp_0123456789abcdef 1>&2; exit 1"},
	})
	if err == nil {
		t.Fatal("expected a failure")
	}
	if strings.Contains(err.Error(), "ghp_0123456789abcdef") {
		t.Fatalf("credential reached the diagnostic: %v", err)
	}
}

func TestStubSubstitutesEveryProcessBackedPort(t *testing.T) {
	stub := &Stub{
		Handler: func(command Command) (Result, error) {
			return Result{Stdout: command.Name + " " + strings.Join(command.Args, " ")}, nil
		},
	}
	ctx := context.Background()
	if _, err := NewGit(stub).Output(ctx, "/tmp", "rev-parse", "HEAD"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewGitHub(stub).Combined(ctx, "/tmp", "project", "list"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMise(stub).Task(ctx, "/tmp", io.Discard, io.Discard, "run", "validate:all"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFSL(stub).Verify(ctx, "/bin/fslc", io.Discard, io.Discard, "check"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTool(stub).Invoke(ctx, Command{Name: "govulncheck", Args: []string{"-json", "./..."}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewShell(stub).Script(ctx, "/tmp", "true", "", nil, time.Second); err != nil {
		t.Fatal(err)
	}
	if len(stub.Calls) != 6 {
		t.Fatalf("recorded %d calls, want 6", len(stub.Calls))
	}
	ports := map[string]bool{}
	for _, call := range stub.Calls {
		ports[call.Port] = true
	}
	for _, port := range []string{PortGit, PortGitHub, PortMise, PortFSL, PortTool, PortShell} {
		if !ports[port] {
			t.Fatalf("port %q was not recorded: %+v", port, stub.Calls)
		}
	}
}

func TestStubReportsAnAbsentBinaryAsUnavailable(t *testing.T) {
	stub := &Stub{Absent: map[string]bool{"gh": true}}
	if NewGitHub(stub).Available() {
		t.Fatal("gh must report as unavailable")
	}
	_, err := NewGitHub(stub).Combined(context.Background(), "", "project", "list")
	if !IsUnavailable(err) {
		t.Fatalf("classification = %v", err)
	}
}

func TestHostCLIRunsThroughTheSubstitutedRunner(t *testing.T) {
	stub := &Stub{}
	host := NewHostCLI(HostCodex, stub)
	if err := host.Run(context.Background(), "/sandbox", "do the task", io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(stub.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1", len(stub.Calls))
	}
	call := stub.Calls[0]
	if call.Port != PortHost || call.Operation != HostCodex {
		t.Fatalf("call = %+v", call)
	}
	if call.Args[len(call.Args)-1] != "do the task" {
		t.Fatalf("prompt must be the last argument: %v", call.Args)
	}
	if call.Timeout != StageTimeout {
		t.Fatalf("timeout = %v, want %v", call.Timeout, StageTimeout)
	}
}

func TestHostCLIInstallsSkillsThroughTheGitAndGitHubPorts(t *testing.T) {
	stub := &Stub{}
	t.Setenv("EVAL_GITHUB_REPO", "")
	if err := NewHostCLI(HostClaudeCode, stub).InstallSkills(context.Background(), "/repo", "/sandbox", io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(stub.Calls) != 2 {
		t.Fatalf("recorded %d calls, want 2", len(stub.Calls))
	}
	if stub.Calls[0].Port != PortGit || stub.Calls[1].Port != PortGitHub {
		t.Fatalf("calls = %+v", stub.Calls)
	}
	if got := strings.Join(stub.Calls[1].Args, " "); !strings.Contains(got, "--agent claude-code") {
		t.Fatalf("gh skill install arguments = %q", got)
	}
}

func TestHostCLIReportsAnAbsentDriverBinary(t *testing.T) {
	stub := &Stub{Absent: map[string]bool{"codex": true}}
	if NewHostCLI(HostCodex, stub).BinaryAvailable() {
		t.Fatal("an absent driver binary must report as unavailable")
	}
}

func TestGitHubAPIClassifiesTransportAndStatusFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.Write([]byte(`[{"sha":"a"}]`))
	}))
	defer server.Close()

	api := NewGitHubAPI(server.URL)
	body, err := api.Get(context.Background(), "/repos/o/r/pulls/1/commits", "test-token")
	if err != nil || string(body) != `[{"sha":"a"}]` {
		t.Fatalf("body = %q, err = %v", body, err)
	}

	if _, err := api.Get(context.Background(), "/missing", "test-token"); err == nil {
		t.Fatal("expected a failure")
	} else {
		var providerErr *Error
		if !errors.As(err, &providerErr) || providerErr.Kind != KindFailure || providerErr.ExitCode != http.StatusNotFound {
			t.Fatalf("status classification = %+v", err)
		}
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(cancelled, "/repos/o/r/pulls/1/commits", "test-token"); err == nil {
		t.Fatal("expected an interruption")
	} else if kind, _ := KindOf(err); kind != KindInterrupted {
		t.Fatalf("interruption classification = %v", kind)
	}
}
