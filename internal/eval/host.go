package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/support"
)

// Supported evaluation drivers. The evaluation runs locally (Issue 173
// decision record): each driver invokes a host CLI headlessly in an isolated
// sandbox. Driver names are stable identifiers used by --host and recorded as
// provenance.
const (
	HostCodex       = "codex"
	HostClaudeCode  = "claude-code"
	HostOpenCode    = "opencode"
	HostAntigravity = "antigravity"
)

// stageTimeout bounds one host stage run so a stuck agent cannot stall the
// whole evaluation run.
const stageTimeout = 5 * time.Minute

// defaultTierModel is the fallback model provenance and explicit -m value
// for the opencode driver when the repository's opencode.json role tiers
// cannot be read: the contracted OpenCode Go low tier.
const defaultTierModel = "opencode-go/deepseek-v4-flash"

// defaultCodexModel is the fixed default model for the codex driver. Codex is
// an OpenAI host, so it uses an OpenAI ChatGPT-tier model, not the OpenCode Go
// tier model; GPT-5.6 Luna is the selected default (override with
// EVAL_CODEX_MODEL).
const defaultCodexModel = "gpt-5.6-luna"

// defaultGeminiModel is the fixed default model for the antigravity driver,
// which does not consume OpenCode Go tier models. It must be one of the
// model identifiers the pinned antigravity CLI advertises; Gemini 3.7 Flash
// low is the cheapest 3.7 Flash variant (override with
// EVAL_ANTIGRAVITY_MODEL).
const defaultGeminiModel = "gemini-3.7-flash-low"

// defaultClaudeModel is the fixed default for the claude-code driver, which
// does not consume OpenCode Go tier models (override with EVAL_CLAUDE_MODEL).
const defaultClaudeModel = "claude-sonnet-5"

// HostRunner executes one headless stage in a sandbox for a driver. The
// runner is an interface so deterministic tests can substitute a fake host.
type HostRunner interface {
	// Name returns the driver identifier (codex, claude-code, opencode,
	// or antigravity).
	Name() string
	// BinaryAvailable reports whether the driver CLI binary exists.
	BinaryAvailable() bool
	// InstallSkills stages the repository skills into the sandbox for the
	// driver, mirroring `gh skill install --from-local`, and prepares any
	// driver-specific runtime configuration.
	InstallSkills(ctx context.Context, root, sandboxDir string, out, errOut io.Writer) error
	// Run executes one headless prompt in the sandbox and appends the
	// combined output to the transcript.
	Run(ctx context.Context, sandboxDir, prompt string, out io.Writer) error
}

// clampedAgent maps a driver to the gh skill agent value used for project
// skill staging. codex, opencode, and antigravity all read `.agents/skills`;
// claude-code reads `.claude/skills`.
func clampedAgent(driver string) string {
	if driver == HostClaudeCode {
		return "claude-code"
	}
	return "codex"
}

// Driver descriptors: binary name, environment override variable, the gh
// skill agent for staging, and the default fixed arguments.
type driverConfig struct {
	binary string
	envVar string
	agent  string
	// fixedArgs is the argument prefix; the prompt is appended last. Drivers
	// that need the sandbox path (antigravity --add-dir) use dirArg.
	dirArg    string // "--add-dir" when set; the sandbox path follows it
	fixedArgs []string
}

var driverConfigs = map[string]driverConfig{
	HostCodex: {
		binary:    "codex",
		envVar:    "EVAL_CODEX_CMD",
		agent:     "codex",
		fixedArgs: []string{"exec"},
	},
	HostClaudeCode: {
		binary:    "claude",
		envVar:    "EVAL_CLAUDE_CMD",
		agent:     "claude-code",
		fixedArgs: []string{"-p"},
	},
	HostOpenCode: {
		binary:    "opencode",
		envVar:    "EVAL_OPENCODE_CMD",
		agent:     "codex",
		fixedArgs: []string{"run", "--auto"},
	},
	HostAntigravity: {
		binary:    "antigravity",
		envVar:    "EVAL_ANTIGRAVITY_CMD",
		agent:     "codex",
		dirArg:    "--add-dir",
		fixedArgs: []string{"--print-timeout", "5m", "--output-format", "text", "--dangerously-skip-permissions"},
	},
}

// modelEnvVars maps every driver to its model override variable. Model
// selection is always explicit (Issue 173 decision record): a driver never
// falls back to an undefined CLI default, so evaluations stay reproducible
// and provenance matches the model actually invoked.
var modelEnvVars = map[string]string{
	HostCodex:       "EVAL_CODEX_MODEL",
	HostClaudeCode:  "EVAL_CLAUDE_MODEL",
	HostOpenCode:    "EVAL_OPENCODE_MODEL",
	HostAntigravity: "EVAL_ANTIGRAVITY_MODEL",
}

// driverConfigFor returns the descriptor for a driver name.
func driverConfigFor(name string) (driverConfig, bool) {
	config, ok := driverConfigs[name]
	return config, ok
}

// cliHost implements HostRunner for one local host CLI driver.
type cliHost struct {
	name   string
	config driverConfig
}

func newCliHost(name string) *cliHost {
	return &cliHost{name: name, config: driverConfigs[name]}
}

func (h *cliHost) Name() string { return h.name }

// commandLine resolves the full command from the environment override or the
// documented default, for binary availability probing.
func (h *cliHost) commandLine() (string, []string) {
	if override := os.Getenv(h.config.envVar); override != "" {
		fields := strings.Fields(override)
		if len(fields) > 0 {
			return fields[0], fields[1:]
		}
	}
	return h.config.binary, h.config.fixedArgs
}

func (h *cliHost) BinaryAvailable() bool {
	binary, _ := h.commandLine()
	_, err := exec.LookPath(binary)
	return err == nil
}

// modelFlag returns the explicit model flag and value for the driver. Every
// driver pins a model (Issue 173 decision record): the per-driver override
// environment variable wins, otherwise the driver default — the contracted
// tier model for codex/claude-code/opencode and the fixed Gemini model for
// antigravity. Run evaluation-level model resolution through
// effectiveModel so the recorded provenance matches the invoked model.
func modelFlag(name string) []string {
	model := os.Getenv(modelEnvVars[name])
	if model == "" {
		switch name {
		case HostAntigravity:
			model = defaultGeminiModel
		case HostClaudeCode:
			model = defaultClaudeModel
		case HostCodex:
			model = defaultCodexModel
		default:
			model = defaultTierModel
		}
	}
	if name == HostCodex || name == HostOpenCode {
		return []string{"-m", model}
	}
	return []string{"--model", model}
}

// effectiveModel resolves the model a driver will actually invoke, matching
// modelFlag: environment override, the per-driver default, or the
// evaluation-level resolved tier model for the tier-driven drivers.
func effectiveModel(name, tier string) string {
	if model := os.Getenv(modelEnvVars[name]); model != "" {
		return model
	}
	switch name {
	case HostAntigravity:
		return defaultGeminiModel
	case HostClaudeCode:
		return defaultClaudeModel
	case HostCodex:
		return defaultCodexModel
	}
	if tier == "" || tier == "unset" {
		return defaultTierModel
	}
	return tier
}

// InstallSkills initializes a Git repository in the sandbox, stages the
// published repository skills the same way check:hosts validates
// installation, and prepares driver-specific runtime configuration.
func (h *cliHost) InstallSkills(ctx context.Context, root, sandboxDir string, out, errOut io.Writer) error {
	if _, err := support.GitOutputIn(sandboxDir, "init", "--quiet"); err != nil {
		return fmt.Errorf("cannot initialize sandbox repository: %w", err)
	}
	cmd := exec.CommandContext(ctx, "gh", "skill", "install", root, "--from-local", "--all",
		"--agent", clampedAgent(h.name), "--scope", "project")
	cmd.Dir = sandboxDir
	cmd.Env = support.GitEnv()
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return err
	}
	return h.prepareRuntime(sandboxDir)
}

// keyModeEnv marks the opt-in for antigravity Gemini API key authentication.
// Local evaluation defaults to the logged-in Google account (keyring); the
// key mode exists for headless/CI runs where no account session exists.
const keyModeEnv = "EVAL_ANTIGRAVITY_KEY_MODE"

// prepareRuntime writes driver-specific configuration into the sandbox. The
// antigravity driver switches to Gemini API key authentication only when the
// caller opts in (EVAL_ANTIGRAVITY_KEY_MODE=1 with GEMINI_API_KEY set), per
// the official install guide (antigravity.google/docs/cli/install):
// settings.json declares modelProvider gemini and the sandbox HOME keeps the
// user's own global configuration untouched. Without the opt-in, the driver
// uses the logged-in Google account from the real HOME and writes nothing.
func (h *cliHost) prepareRuntime(sandboxDir string) error {
	if h.name != HostAntigravity || os.Getenv(keyModeEnv) != "1" || os.Getenv("GEMINI_API_KEY") == "" {
		return nil
	}
	settingsDir := filepath.Join(sandboxDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return err
	}
	content, err := json.Marshal(map[string]string{"modelProvider": "gemini"})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(settingsDir, "settings.json"), content, 0o644)
}

// runEnv returns the child environment: the repository's GitEnv plus, for
// the antigravity driver in opt-in API key mode, an isolated HOME so the CLI
// reads settings.json from the sandbox and never the user's real
// configuration. Without the opt-in the driver keeps the real HOME and uses
// the logged-in Google account.
func (h *cliHost) runEnv(sandboxDir string) []string {
	env := support.GitEnv()
	if h.name == HostAntigravity && os.Getenv(keyModeEnv) == "1" && os.Getenv("GEMINI_API_KEY") != "" {
		env = append(env, "HOME="+sandboxDir)
	}
	return env
}

// Run executes one headless stage prompt in the sandbox. The command
// arguments come from the resolved command line. The invocation is
// deliberately simple and documented so a first live run against the pinned
// CLI can confirm or refine it via the EVAL_*_CMD environment variables.
// The stage output is streamed to out; on failure the error carries a snippet
// of the CLI output so infrastructure errors stay attributable.
func (h *cliHost) Run(ctx context.Context, sandboxDir, prompt string, out io.Writer) error {
	binary, args := h.commandLine()
	if h.config.dirArg != "" {
		args = append(args, h.config.dirArg, sandboxDir)
	}
	args = append(args, modelFlag(h.name)...)
	// antigravity's --print flag consumes the next argument as its value, so
	// the prompt must directly follow it; other flags stay before it.
	if h.name == HostAntigravity {
		args = append(args, "--print", prompt)
	} else {
		args = append(args, prompt)
	}
	runCtx, cancel := context.WithTimeout(ctx, stageTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, binary, args...)
	cmd.Dir = sandboxDir
	cmd.Env = h.runEnv(sandboxDir)
	var captured bytes.Buffer
	cmd.Stdout = io.MultiWriter(out, &captured)
	cmd.Stderr = io.MultiWriter(out, &captured)
	if err := cmd.Run(); err != nil {
		if message := strings.TrimSpace(captured.String()); message != "" {
			if len(message) > 300 {
				message = "..." + message[len(message)-300:]
			}
			return fmt.Errorf("exit %d: %s", support.ExitError(err), message)
		}
		return err
	}
	return nil
}

// ResolveHosts resolves the --host flag value into a deterministic driver
// list. The value is a comma-separated driver list, or "all" for every
// driver.
func ResolveHosts(hostFlag string) ([]string, error) {
	if hostFlag == "all" {
		return []string{HostCodex, HostClaudeCode, HostOpenCode, HostAntigravity}, nil
	}
	var hosts []string
	for _, part := range strings.Split(hostFlag, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := driverConfigs[name]; !ok {
			return nil, fmt.Errorf("host %q must be one of codex, claude-code, opencode, antigravity, or a comma-separated list", name)
		}
		hosts = append(hosts, name)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("host must name at least one driver")
	}
	return hosts, nil
}

// runnerFor returns the host runner for a driver identifier.
func runnerFor(name string) HostRunner {
	return newCliHost(name)
}

// resolveModel returns the model provenance default from opencode.json
// (agent.low.model), or "unset" when the file is absent.
func resolveModel(root string) string {
	content, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		return "unset"
	}
	var config struct {
		Agent struct {
			Low struct {
				Model string `json:"model"`
			} `json:"low"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(content, &config); err != nil || config.Agent.Low.Model == "" {
		return "unset"
	}
	return config.Agent.Low.Model
}
