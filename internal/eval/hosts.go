package eval

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hidekitux/skills/internal/provider"
)

// ResolveHosts resolves the --host flag value into a deterministic driver
// list. The driver vocabulary belongs to the host command line port, so this
// forwards to it and keeps cmd/evaluate reading one evaluation entry point.
func ResolveHosts(hostFlag string) ([]string, error) { return provider.ResolveHosts(hostFlag) }

// gitPort is the Git port the evaluation reads repository provenance
// through. A test substitutes it by assigning a Git built on a provider.Stub
// runner.
var gitPort = provider.NewGit(provider.OSRunner{})

// shellPort runs the bounded sh commands an evaluation scenario declares: an
// assertion command and an external rubric reviewer.
var shellPort = provider.NewShell(provider.OSRunner{})

// runnerFor returns the production host command line adapter for a driver.
func runnerFor(name string) provider.HostCLI {
	return provider.NewHostCLI(name, provider.OSRunner{})
}

// resolveModel returns the model provenance default from opencode.json
// (agent.low.model), or "unset" when the file is absent. The file is
// repository data rather than an external provider, so it is read here.
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
