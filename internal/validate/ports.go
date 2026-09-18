package validate

import "github.com/hidekitux/skills/internal/provider"

// The ports every validation in this package reaches an external system
// through. A test substitutes any of them by assigning a port built on a
// provider.Stub runner.
var (
	gitPort    = provider.NewGit(provider.OSRunner{})
	githubPort = provider.NewGitHub(provider.OSRunner{})
	toolPort   = provider.NewTool(provider.OSRunner{})
	apiPort    = provider.NewGitHubAPI("")
)
