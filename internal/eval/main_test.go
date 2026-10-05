package eval

import (
	"os"
	"testing"

	"github.com/hidekitux/skills/internal/provider"
)

// TestMain replaces githubPort with a stub before any test runs, so no test
// in this package reaches a real GitHub repository through the sandbox issue
// cleanup. A test that sets EVAL_GITHUB_REPO would otherwise list and close
// issues in that repository with the developer's gh login. The stub reports
// no open issues; tests that need listings substitute their own stub.
func TestMain(m *testing.M) {
	githubPort = provider.NewGitHub(&provider.Stub{})
	os.Exit(m.Run())
}
