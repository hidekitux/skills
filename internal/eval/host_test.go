package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/support"
)

func TestDriverConfigsCoverEveryHost(t *testing.T) {
	for _, name := range []string{HostCodex, HostClaudeCode, HostOpenCode, HostAntigravity} {
		if _, ok := driverConfigs[name]; !ok {
			t.Fatalf("driver %q is missing a config", name)
		}
		if config := driverConfigs[name]; config.binary == "" || config.envVar == "" || config.agent == "" {
			t.Fatalf("driver %q config is incomplete: %+v", name, config)
		}
	}
}

func TestClampedAgent(t *testing.T) {
	for _, name := range []string{HostCodex, HostOpenCode, HostAntigravity} {
		if got := clampedAgent(name); got != "codex" {
			t.Fatalf("clampedAgent(%s) = %s, want codex (.agents/skills)", name, got)
		}
	}
	if got := clampedAgent(HostClaudeCode); got != "claude-code" {
		t.Fatalf("clampedAgent(claude-code) = %s", got)
	}
}

func TestAntigravityPrepareRuntimeWritesSettingsWhenKeyModeOptIn(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key-value")
	t.Setenv("EVAL_ANTIGRAVITY_KEY_MODE", "1")
	sandbox := t.TempDir()
	host := newCliHost(HostAntigravity)
	if err := host.prepareRuntime(sandbox); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sandbox, ".gemini", "antigravity-cli", "settings.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings.json not written: %v", err)
	}
	var settings map[string]string
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatal(err)
	}
	if settings["modelProvider"] != "gemini" {
		t.Fatalf("modelProvider = %q, want gemini", settings["modelProvider"])
	}
}

func TestAntigravityPrepareRuntimeDefaultsToAccountLogin(t *testing.T) {
	// Without the key-mode opt-in the driver writes no sandbox configuration
	// and keeps the logged-in Google account from the real HOME.
	t.Setenv("GEMINI_API_KEY", "test-key-value")
	t.Setenv("EVAL_ANTIGRAVITY_KEY_MODE", "")
	sandbox := t.TempDir()
	host := newCliHost(HostAntigravity)
	if err := host.prepareRuntime(sandbox); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sandbox, ".gemini")); !os.IsNotExist(err) {
		t.Fatalf("no sandbox settings expected in account mode, stat err = %v", err)
	}
	env := host.runEnv(sandbox)
	for _, kv := range env {
		if strings.HasPrefix(kv, "HOME=") && strings.TrimPrefix(kv, "HOME=") == sandbox {
			t.Fatalf("HOME must not be isolated in account mode: %v", env)
		}
	}
}

func TestAntigravityRunEnvIsolatesHomeOnlyWithKeyMode(t *testing.T) {
	sandbox := t.TempDir()
	host := newCliHost(HostAntigravity)

	t.Setenv("GEMINI_API_KEY", "test-key-value")
	t.Setenv("EVAL_ANTIGRAVITY_KEY_MODE", "1")
	env := host.runEnv(sandbox)
	if !containsPrefix(env, "HOME="+sandbox) {
		t.Fatalf("expected HOME isolation in key mode, env = %v", env)
	}

	t.Setenv("EVAL_ANTIGRAVITY_KEY_MODE", "")
	env = host.runEnv(sandbox)
	for _, kv := range env {
		if strings.HasPrefix(kv, "HOME=") && strings.TrimPrefix(kv, "HOME=") == sandbox {
			t.Fatalf("HOME must not be isolated in account mode: %v", env)
		}
	}
}

func containsPrefix(kvs []string, prefix string) bool {
	for _, kv := range kvs {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
}

func TestRunEnvFiltersCredentialVariables(t *testing.T) {
	sandbox := t.TempDir()
	host := newCliHost(HostCodex)
	t.Setenv("GEMINI_API_KEY", "gl-test-credential")
	t.Setenv("GH_TOKEN", "ghp_test-credential")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-test-credential")
	t.Setenv("EVAL_GITHUB_REPO", "hidekitux/sandbox")
	env := host.runEnv(sandbox)
	for _, kv := range env {
		if strings.HasPrefix(kv, "GEMINI_API_KEY=") || strings.HasPrefix(kv, "GH_TOKEN=") || strings.HasPrefix(kv, "AWS_SECRET_ACCESS_KEY=") {
			t.Fatalf("credential leaked into evaluated-agent environment: %s", kv)
		}
	}
	if !containsPrefix(env, "GH_REPO=hidekitux/sandbox") {
		t.Fatalf("expected GH_REPO repository context, env = %v", env)
	}
}

func TestRunEnvKeyModeExposesGeminiKeyToAntigravityOnly(t *testing.T) {
	sandbox := t.TempDir()
	t.Setenv("GEMINI_API_KEY", "gl-test-credential")
	t.Setenv("EVAL_ANTIGRAVITY_KEY_MODE", "1")
	env := newCliHost(HostAntigravity).runEnv(sandbox)
	if !containsPrefix(env, "GEMINI_API_KEY=gl-test-credential") {
		t.Fatalf("antigravity key mode must receive GEMINI_API_KEY, env = %v", env)
	}
	if !containsPrefix(env, "HOME="+sandbox) {
		t.Fatalf("antigravity key mode must isolate HOME, env = %v", env)
	}
	env = newCliHost(HostCodex).runEnv(sandbox)
	for _, kv := range env {
		if strings.HasPrefix(kv, "GEMINI_API_KEY=") {
			t.Fatalf("non-antigravity driver inherited GEMINI_API_KEY: %s", kv)
		}
	}
}

func TestWireGithubRepoRegistersConfiguredOrigin(t *testing.T) {
	sandbox := t.TempDir()
	if _, err := support.GitOutputIn(sandbox, "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EVAL_GITHUB_REPO", "")
	if err := wireGithubRepo(sandbox); err != nil {
		t.Fatalf("unconfigured repo must not fail: %v", err)
	}
	t.Setenv("EVAL_GITHUB_REPO", "hidekitux/sandbox")
	if err := wireGithubRepo(sandbox); err != nil {
		t.Fatal(err)
	}
	out, err := support.GitOutputIn(sandbox, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(out) != "https://github.com/hidekitux/sandbox.git" {
		t.Fatalf("origin = %q, err = %v", out, err)
	}
}

func TestModelFlagPinsModelForEveryDriver(t *testing.T) {
	for _, name := range []string{HostCodex, HostClaudeCode, HostOpenCode, HostAntigravity} {
		t.Setenv(modelEnvVars[name], "")
	}
	for _, name := range []string{HostCodex, HostOpenCode} {
		if got := modelFlag(name); len(got) != 2 || got[0] != "-m" {
			t.Fatalf("%s modelFlag = %v", name, got)
		}
	}
	if got := modelFlag(HostCodex); got[1] != defaultCodexModel {
		t.Fatalf("codex default = %s, want %s", got[1], defaultCodexModel)
	}
	if got := modelFlag(HostOpenCode); got[1] != defaultTierModel {
		t.Fatalf("opencode default = %s, want %s", got[1], defaultTierModel)
	}
	for _, name := range []string{HostClaudeCode, HostAntigravity} {
		got := modelFlag(name)
		if len(got) != 2 || got[0] != "--model" {
			t.Fatalf("%s modelFlag = %v", name, got)
		}
	}
	if got := modelFlag(HostAntigravity); got[1] != defaultGeminiModel {
		t.Fatalf("antigravity default = %s, want %s", got[1], defaultGeminiModel)
	}
	if got := modelFlag(HostClaudeCode); got[1] != defaultClaudeModel {
		t.Fatalf("claude default = %s, want %s", got[1], defaultClaudeModel)
	}
}

func TestModelFlagHonorsEnvOverride(t *testing.T) {
	t.Setenv("EVAL_CODEX_MODEL", "gpt-5.1-codex")
	if got := modelFlag(HostCodex); got[1] != "gpt-5.1-codex" {
		t.Fatalf("modelFlag = %v, want env override", got)
	}
}

func TestEffectiveModelMatchesInvocation(t *testing.T) {
	// Non-antigravity drivers follow evaluation-level tier resolution.
	if got := effectiveModel(HostOpenCode, "opencode-go/deepseek-v4-flash"); got != "opencode-go/deepseek-v4-flash" {
		t.Fatalf("effectiveModel = %s", got)
	}
	// Codex keeps its own OpenAI default instead of the tier model.
	if got := effectiveModel(HostCodex, "opencode-go/deepseek-v4-flash"); got != defaultCodexModel {
		t.Fatalf("codex effectiveModel = %s, want %s", got, defaultCodexModel)
	}
	if got := effectiveModel(HostOpenCode, "unset"); got != defaultTierModel {
		t.Fatalf("effectiveModel(unset) = %s", got)
	}
	// Antigravity and claude-code pin their own defaults and ignore the tier.
	if got := effectiveModel(HostAntigravity, "opencode-go/deepseek-v4-flash"); got != defaultGeminiModel {
		t.Fatalf("antigravity effectiveModel = %s, want %s", got, defaultGeminiModel)
	}
	if got := effectiveModel(HostClaudeCode, "opencode-go/deepseek-v4-flash"); got != defaultClaudeModel {
		t.Fatalf("claude effectiveModel = %s, want %s", got, defaultClaudeModel)
	}
	t.Setenv("EVAL_ANTIGRAVITY_MODEL", "gemini-3-pro-image-preview")
	if got := effectiveModel(HostAntigravity, "opencode-go/deepseek-v4-flash"); got != "gemini-3-pro-image-preview" {
		t.Fatalf("antigravity override = %s", got)
	}
}

// TestRunnerForReturnsRealDrivers guards the public runner factory used by
// cmd/evaluate.
func TestRunnerForReturnsRealDrivers(t *testing.T) {
	host := runnerFor(HostAntigravity)
	if host.Name() != HostAntigravity {
		t.Fatalf("runner name = %s", host.Name())
	}
}
