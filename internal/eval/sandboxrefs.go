package eval

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// sandboxRefs is the state of the sandbox repository's pull requests and
// branches when a GitHub-dependent scenario starts.
type sandboxRefs struct {
	pulls         map[int]bool
	branches      map[string]bool
	defaultBranch string
}

// snapshotSandboxRefs records the open pull requests, the branch names, and
// the default branch of the sandbox repository.
func snapshotSandboxRefs(ctx context.Context, repo string) (sandboxRefs, error) {
	pulls, err := openSandboxPulls(ctx, repo)
	if err != nil {
		return sandboxRefs{}, err
	}
	branches, err := sandboxBranches(ctx, repo)
	if err != nil {
		return sandboxRefs{}, err
	}
	result, err := githubPort.Combined(ctx, "", "api", "repos/"+repo, "--jq", ".default_branch")
	if err != nil {
		return sandboxRefs{}, fmt.Errorf("cannot read the default branch of %s: %w", repo, err)
	}
	return sandboxRefs{pulls: pulls, branches: branches, defaultBranch: strings.TrimSpace(result.Combined)}, nil
}

// openSandboxPulls returns the numbers of the open pull requests in the
// sandbox repository.
func openSandboxPulls(ctx context.Context, repo string) (map[int]bool, error) {
	result, err := githubPort.Combined(ctx, "", "pr", "list", "--repo", repo,
		"--state", "open", "--limit", "1000", "--json", "number", "--jq", ".[].number")
	if err != nil {
		return nil, fmt.Errorf("cannot list open sandbox pull requests in %s: %w", repo, err)
	}
	numbers := map[int]bool{}
	for _, field := range strings.Fields(result.Combined) {
		number, convErr := strconv.Atoi(field)
		if convErr != nil {
			return nil, fmt.Errorf("cannot read sandbox pull request number %q from %s", field, repo)
		}
		numbers[number] = true
	}
	return numbers, nil
}

// sandboxBranches returns the branch names of the sandbox repository.
func sandboxBranches(ctx context.Context, repo string) (map[string]bool, error) {
	result, err := githubPort.Combined(ctx, "", "api", "--paginate", "repos/"+repo+"/branches", "--jq", ".[].name")
	if err != nil {
		return nil, fmt.Errorf("cannot list sandbox branches in %s: %w", repo, err)
	}
	names := map[string]bool{}
	for _, name := range strings.Fields(result.Combined) {
		names[name] = true
	}
	return names, nil
}

// cleanNewSandboxRefs closes every pull request that is open now but was not
// open in before, then deletes every branch that did not exist in before,
// except the default branch. Pull requests and branches that existed before
// the scenario stay as they were.
func cleanNewSandboxRefs(ctx context.Context, repo, scenario string, before sandboxRefs) error {
	var failures []string
	pulls, err := openSandboxPulls(ctx, repo)
	if err != nil {
		failures = append(failures, err.Error())
	}
	var created []int
	for number := range pulls {
		if !before.pulls[number] {
			created = append(created, number)
		}
	}
	sort.Ints(created)
	for _, number := range created {
		comment := fmt.Sprintf("Closed by the evaluation harness after scenario %s, which opened this pull request.", scenario)
		if _, err := githubPort.Combined(ctx, "", "pr", "close", strconv.Itoa(number),
			"--repo", repo, "--comment", comment); err != nil {
			failures = append(failures, fmt.Sprintf("pull request #%d: %v", number, err))
		}
	}

	branches, err := sandboxBranches(ctx, repo)
	if err != nil {
		failures = append(failures, err.Error())
	}
	var pushed []string
	for name := range branches {
		if !before.branches[name] && name != before.defaultBranch {
			pushed = append(pushed, name)
		}
	}
	sort.Strings(pushed)
	for _, name := range pushed {
		if _, err := githubPort.Combined(ctx, "", "api", "--method", "DELETE",
			"repos/"+repo+"/git/refs/heads/"+name); err != nil {
			failures = append(failures, fmt.Sprintf("branch %s: %v", name, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("cannot clean sandbox pull requests and branches in %s: %s", repo, strings.Join(failures, "; "))
	}
	return nil
}
