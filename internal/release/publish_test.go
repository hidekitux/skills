package release

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestPublishReleaseUsesCanonicalMiseTasks(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test source")
	}
	content, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "publish.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, task := range []string{"validate:all", "validate:skill-creator", "verify:release"} {
		if !strings.Contains(source, `"`+task+`"`) {
			t.Errorf("publish.go must invoke canonical task %q", task)
		}
	}
	for _, retired := range []string{"validate", "validate-skill-creator", "verify-release"} {
		if strings.Contains(source, `"`+retired+`"`) {
			t.Errorf("publish.go must not invoke retired task %q", retired)
		}
	}
}

func writeSkillCreatorFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "quick_validate.py"), []byte("# fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPublishReleaseStopsBeforePublicationWhenVerificationFails(t *testing.T) {
	runner := newFakeReleaseRunner()
	runner.streamResults[commandKey("mise", "run", "validate:all")] = 0
	runner.streamResults[commandKey("mise", "run", "validate:skill-creator")] = 0
	runner.streamResults[commandKey("mise", "run", "verify:release", "--", "v1.2.3")] = 7
	t.Setenv("SKILL_CREATOR_ROOT", writeSkillCreatorFixture(t))

	var out, errOut bytes.Buffer
	if code := publishRelease([]string{"v1.2.3"}, &out, &errOut, runner); code != 7 {
		t.Fatalf("expected verification exit code 7, got %d", code)
	}
	want := []fakeCommandCall{
		{name: "mise", args: []string{"run", "validate:all"}},
		{name: "mise", args: []string{"run", "validate:skill-creator"}},
		{name: "mise", args: []string{"run", "verify:release", "--", "v1.2.3"}},
	}
	if !reflect.DeepEqual(runner.streamCalls, want) {
		t.Fatalf("stream calls = %+v, want %+v", runner.streamCalls, want)
	}
}

func TestPublishReleaseRunsCommandsInOrderOnSuccess(t *testing.T) {
	runner := newFakeReleaseRunner()
	runner.streamResults[commandKey("mise", "run", "validate:all")] = 0
	runner.streamResults[commandKey("mise", "run", "validate:skill-creator")] = 0
	runner.streamResults[commandKey("mise", "run", "verify:release", "--", "v1.2.3")] = 0
	runner.streamResults[commandKey("gh", "skill", "publish", "--tag", "v1.2.3")] = 0
	t.Setenv("SKILL_CREATOR_ROOT", writeSkillCreatorFixture(t))

	var out, errOut bytes.Buffer
	if code := publishRelease([]string{"v1.2.3"}, &out, &errOut, runner); code != 0 {
		t.Fatalf("expected publication sequence to pass, got %d: %s", code, errOut.String())
	}
	want := []fakeCommandCall{
		{name: "mise", args: []string{"run", "validate:all"}},
		{name: "mise", args: []string{"run", "validate:skill-creator"}},
		{name: "mise", args: []string{"run", "verify:release", "--", "v1.2.3"}},
		{name: "gh", args: []string{"skill", "publish", "--tag", "v1.2.3"}},
	}
	if !reflect.DeepEqual(runner.streamCalls, want) {
		t.Fatalf("stream calls = %+v, want %+v", runner.streamCalls, want)
	}
}

func TestPublishReleaseSkipsUnavailableSkillCreator(t *testing.T) {
	runner := newFakeReleaseRunner()
	runner.streamResults[commandKey("mise", "run", "validate:all")] = 0
	runner.streamResults[commandKey("mise", "run", "verify:release", "--", "v1.2.3")] = 0
	runner.streamResults[commandKey("gh", "skill", "publish", "--tag", "v1.2.3")] = 0
	t.Setenv("SKILL_CREATOR_ROOT", filepath.Join(t.TempDir(), "missing"))

	var out, errOut bytes.Buffer
	if code := publishRelease([]string{"v1.2.3"}, &out, &errOut, runner); code != 0 {
		t.Fatalf("expected unavailable skill-creator path to pass, got %d: %s", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "gh skill publish") {
		t.Fatal("publication command must not appear in diagnostic output")
	}
	want := []fakeCommandCall{
		{name: "mise", args: []string{"run", "validate:all"}},
		{name: "mise", args: []string{"run", "verify:release", "--", "v1.2.3"}},
		{name: "gh", args: []string{"skill", "publish", "--tag", "v1.2.3"}},
	}
	if !reflect.DeepEqual(runner.streamCalls, want) {
		t.Fatalf("stream calls = %+v, want %+v", runner.streamCalls, want)
	}
}

func TestPublishReleaseUsesCommandAdapter(t *testing.T) {
	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands.log")
	writeExecutable(t, bin, "mise", "#!/bin/sh\nprintf 'mise %s\\n' \"$*\" >> \"$RELEASE_TEST_LOG\"\nexit 0\n")
	writeExecutable(t, bin, "gh", "#!/bin/sh\nprintf 'gh %s\\n' \"$*\" >> \"$RELEASE_TEST_LOG\"\nexit 0\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RELEASE_TEST_LOG", logPath)
	t.Setenv("SKILL_CREATOR_ROOT", filepath.Join(t.TempDir(), "missing"))

	var out, errOut bytes.Buffer
	if code := PublishRelease([]string{"v1.2.3"}, &out, &errOut); code != 0 {
		t.Fatalf("expected command adapter to pass, got %d: %s", code, errOut.String())
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "mise run validate:all\nmise run verify:release -- v1.2.3\ngh skill publish --tag v1.2.3\n"
	if string(content) != want {
		t.Fatalf("command log = %q, want %q", content, want)
	}
}
