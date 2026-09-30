package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const configureRulesetScript = "../../skills/govern/bootstrap-project/scripts/configure-github-ruleset.py"

// fakeGH answers the gh calls that configure-github-ruleset.py makes and
// appends each call to FAKE_GH_LOG as one JSON line with its stdin. It keeps
// the ruleset it receives in FAKE_GH_STATE so a later read returns it.
// FAKE_GH_EXISTING=1 reports an existing main ruleset and a deprecated
// issue-branch ruleset.
const fakeGH = `#!/usr/bin/env python3
import json, os, sys

args = sys.argv[1:]
data = "" if sys.stdin.isatty() else sys.stdin.read()
with open(os.environ["FAKE_GH_LOG"], "a") as log:
    log.write(json.dumps({"args": args, "stdin": data}) + "\n")
state = os.environ["FAKE_GH_STATE"]
existing = os.environ.get("FAKE_GH_EXISTING") == "1"
deleted = os.path.exists(state + ".deleted")

if args[:2] == ["auth", "status"]:
    sys.exit(0)
if "PATCH" in args:
    print(json.dumps({"allow_merge_commit": False, "allow_squash_merge": False,
                      "allow_rebase_merge": True, "delete_branch_on_merge": True}))
    sys.exit(0)
if "--paginate" in args:
    page = []
    if existing:
        page.append({"id": 5, "name": "Require pull requests on protected branches",
                     "source_type": "Repository"})
        if not deleted:
            page.append({"id": 9, "name": "Require signed commits on issue branches",
                         "source_type": "Repository"})
    print(json.dumps([page]))
    sys.exit(0)
if "DELETE" in args:
    open(state + ".deleted", "w").close()
    sys.exit(0)
if "POST" in args or "PUT" in args:
    ruleset = json.loads(data)
    ruleset["id"] = 5 if existing else 7
    with open(state, "w") as handle:
        json.dump(ruleset, handle)
    print(json.dumps(ruleset))
    sys.exit(0)
if args[0] == "api" and "/rulesets/" in args[-1]:
    with open(state) as handle:
        print(handle.read())
    sys.exit(0)
sys.stderr.write("unexpected gh call: %r\n" % args)
sys.exit(1)
`

type ghCall struct {
	Args  []string `json:"args"`
	Stdin string   `json:"stdin"`
}

// fakeGHEnv puts fakeGH first on PATH and returns the environment entries
// and the call-log path.
func fakeGHEnv(t *testing.T, existing bool) ([]string, string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(fakeGH), 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "gh.log")
	env := []string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_GH_LOG=" + logPath,
		"FAKE_GH_STATE=" + filepath.Join(dir, "ruleset.json"),
	}
	if existing {
		env = append(env, "FAKE_GH_EXISTING=1")
	}
	return env, logPath
}

func readGHCalls(t *testing.T, logPath string) []ghCall {
	t.Helper()
	content, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var calls []ghCall
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		var call ghCall
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			t.Fatalf("parse gh log line %q: %v", line, err)
		}
		calls = append(calls, call)
	}
	return calls
}

var soloRulesetArgs = []string{
	"--repo", "octo/demo",
	"--branch", "main",
	"--branch", "release",
	"--required-check", "Validate tests",
	"--required-check", "Validate signed pull-request commits",
	"--approvals", "0",
	"--allow-last-push-approval",
}

type rulesetDryRun struct {
	RepositorySettings map[string]bool `json:"repository_settings"`
	Ruleset            struct {
		Name        string `json:"name"`
		Enforcement string `json:"enforcement"`
		Conditions  struct {
			RefName struct {
				Include []string `json:"include"`
			} `json:"ref_name"`
		} `json:"conditions"`
		Rules []struct {
			Type       string         `json:"type"`
			Parameters map[string]any `json:"parameters"`
		} `json:"rules"`
	} `json:"ruleset"`
	DeprecatedIssueRuleset string `json:"deprecated_issue_ruleset"`
}

func TestConfigureRulesetDryRunPrintsPayloadWithoutCallingGH(t *testing.T) {
	env, logPath := fakeGHEnv(t, false)
	stdout, stderr, code := runPythonScript(t, env, configureRulesetScript, soloRulesetArgs...)
	if code != 0 || !strings.Contains(stderr, "Dry run only.") {
		t.Fatalf("dry run: code=%d stderr=%q", code, stderr)
	}
	if calls := readGHCalls(t, logPath); len(calls) != 0 {
		t.Fatalf("dry run called gh: %+v", calls)
	}
	var got rulesetDryRun
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse dry-run payload: %v\n%s", err, stdout)
	}
	wantSettings := map[string]bool{
		"allow_merge_commit": false, "allow_squash_merge": false,
		"allow_rebase_merge": true, "delete_branch_on_merge": true,
	}
	if !reflect.DeepEqual(got.RepositorySettings, wantSettings) {
		t.Errorf("repository settings = %v, want %v", got.RepositorySettings, wantSettings)
	}
	if got.Ruleset.Name != "Require pull requests on protected branches" || got.Ruleset.Enforcement != "active" {
		t.Errorf("ruleset name or enforcement = %q, %q", got.Ruleset.Name, got.Ruleset.Enforcement)
	}
	if want := []string{"refs/heads/main", "refs/heads/release"}; !reflect.DeepEqual(got.Ruleset.Conditions.RefName.Include, want) {
		t.Errorf("branch targets = %v, want %v", got.Ruleset.Conditions.RefName.Include, want)
	}
	var types []string
	rules := map[string]map[string]any{}
	for _, rule := range got.Ruleset.Rules {
		types = append(types, rule.Type)
		rules[rule.Type] = rule.Parameters
	}
	if want := []string{"pull_request", "required_status_checks", "non_fast_forward", "deletion", "required_linear_history"}; !reflect.DeepEqual(types, want) {
		t.Errorf("rule types = %v, want %v", types, want)
	}
	pr := rules["pull_request"]
	if !reflect.DeepEqual(pr["allowed_merge_methods"], []any{"rebase"}) ||
		pr["required_approving_review_count"] != float64(0) ||
		pr["require_last_push_approval"] != false ||
		pr["require_code_owner_review"] != false {
		t.Errorf("pull_request parameters = %v", pr)
	}
	checks := rules["required_status_checks"]
	wantChecks := []any{
		map[string]any{"context": "Validate tests"},
		map[string]any{"context": "Validate signed pull-request commits"},
	}
	if !reflect.DeepEqual(checks["required_status_checks"], wantChecks) || checks["strict_required_status_checks_policy"] != true {
		t.Errorf("required_status_checks parameters = %v", checks)
	}
	if got.DeprecatedIssueRuleset != "Require signed commits on issue branches" {
		t.Errorf("deprecated issue ruleset = %q", got.DeprecatedIssueRuleset)
	}
}

func TestConfigureRulesetRejectsInvalidArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"repository without owner", []string{"--repo", "demo", "--required-check", "Validate signed pull-request commits"}, "--repo must be OWNER/REPO"},
		{"too many approvals", []string{"--repo", "octo/demo", "--approvals", "7", "--required-check", "Validate signed pull-request commits"}, "--approvals must be between 0 and 6"},
		{"solo approval without last-push opt-in", []string{"--repo", "octo/demo", "--approvals", "0", "--required-check", "Validate signed pull-request commits"}, "--approvals 0 requires --allow-last-push-approval"},
		{"no required check", []string{"--repo", "octo/demo"}, "supply at least one --required-check"},
		{"qualified branch", []string{"--repo", "octo/demo", "--branch", "refs/heads/main", "--required-check", "Validate signed pull-request commits"}, "--branch values must be unqualified branch names"},
		{"duplicate branch", []string{"--repo", "octo/demo", "--branch", "main", "--branch", "main", "--required-check", "Validate signed pull-request commits"}, "--branch values must be unique"},
		{"missing signature check", []string{"--repo", "octo/demo", "--required-check", "Validate tests"}, "include --required-check 'Validate signed pull-request commits'"},
	} {
		env, logPath := fakeGHEnv(t, false)
		stdout, stderr, code := runPythonScript(t, env, configureRulesetScript, tc.args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: code=%d stdout=%q stderr=%q, want exit 2 with %q", tc.name, code, stdout, stderr, tc.want)
		}
		if calls := readGHCalls(t, logPath); len(calls) != 0 {
			t.Errorf("%s: rejected arguments called gh: %+v", tc.name, calls)
		}
	}
}

// callShape reduces a gh call to its method and endpoint for comparison.
func callShape(call ghCall) string {
	method := "GET"
	endpoint := ""
	for index, arg := range call.Args {
		if arg == "--method" && index+1 < len(call.Args) {
			method = call.Args[index+1]
		}
		if strings.HasPrefix(arg, "repos/") {
			endpoint = arg
		}
	}
	if len(call.Args) >= 2 && call.Args[0] == "auth" {
		return "auth " + call.Args[1]
	}
	return method + " " + endpoint
}

func TestConfigureRulesetApplyCreatesAndVerifiesRuleset(t *testing.T) {
	env, logPath := fakeGHEnv(t, false)
	dryRun, _, code := runPythonScript(t, env, configureRulesetScript, soloRulesetArgs...)
	if code != 0 {
		t.Fatalf("dry run exit %d", code)
	}
	stdout, stderr, code := runPythonScript(t, env, configureRulesetScript, append(soloRulesetArgs, "--apply")...)
	if code != 0 || strings.TrimSpace(stdout) != "Created and verified ruleset 7 for octo/demo." {
		t.Fatalf("apply: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	calls := readGHCalls(t, logPath)
	var shapes []string
	for _, call := range calls {
		shapes = append(shapes, callShape(call))
	}
	want := []string{
		"auth status",
		"PATCH repos/octo/demo",
		"GET repos/octo/demo/rulesets",
		"POST repos/octo/demo/rulesets",
		"GET repos/octo/demo/rulesets/7",
		"GET repos/octo/demo/rulesets",
	}
	if !reflect.DeepEqual(shapes, want) {
		t.Fatalf("gh calls = %v, want %v", shapes, want)
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(calls[3].Stdin), &sent); err != nil {
		t.Fatalf("parse POST payload: %v", err)
	}
	var planned struct {
		Ruleset map[string]any `json:"ruleset"`
	}
	if err := json.Unmarshal([]byte(dryRun), &planned); err != nil {
		t.Fatal(err)
	}
	if printed := planned.Ruleset; !reflect.DeepEqual(sent, printed) {
		t.Errorf("POST payload differs from the dry-run ruleset:\nsent=%v\nprinted=%v", sent, printed)
	}
}

func TestConfigureRulesetApplyUpdatesRulesetAndRemovesDeprecatedOne(t *testing.T) {
	env, logPath := fakeGHEnv(t, true)
	stdout, stderr, code := runPythonScript(t, env, configureRulesetScript, append(soloRulesetArgs, "--apply")...)
	wantOut := "Updated and verified ruleset 5 for octo/demo.\nRemoved deprecated issue-branch ruleset for octo/demo."
	if code != 0 || strings.TrimSpace(stdout) != wantOut {
		t.Fatalf("apply: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var shapes []string
	for _, call := range readGHCalls(t, logPath) {
		shapes = append(shapes, callShape(call))
	}
	want := []string{
		"auth status",
		"PATCH repos/octo/demo",
		"GET repos/octo/demo/rulesets",
		"PUT repos/octo/demo/rulesets/5",
		"GET repos/octo/demo/rulesets/5",
		"GET repos/octo/demo/rulesets",
		"DELETE repos/octo/demo/rulesets/9",
		"GET repos/octo/demo/rulesets",
	}
	if !reflect.DeepEqual(shapes, want) {
		t.Fatalf("gh calls = %v, want %v", shapes, want)
	}
}
