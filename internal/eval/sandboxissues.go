package eval

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// openSandboxIssues returns the numbers of the open issues in the sandbox
// repository.
func openSandboxIssues(ctx context.Context, repo string) (map[int]bool, error) {
	result, err := githubPort.Combined(ctx, "", "issue", "list", "--repo", repo,
		"--state", "open", "--limit", "1000", "--json", "number", "--jq", ".[].number")
	if err != nil {
		return nil, fmt.Errorf("cannot list open sandbox issues in %s: %w", repo, err)
	}
	numbers := map[int]bool{}
	for _, field := range strings.Fields(result.Combined) {
		number, convErr := strconv.Atoi(field)
		if convErr != nil {
			return nil, fmt.Errorf("cannot read sandbox issue number %q from %s", field, repo)
		}
		numbers[number] = true
	}
	return numbers, nil
}

// closeNewSandboxIssues closes every issue that is open now but was not open
// in before, so the next scenario does not see it. An issue that was open
// before the scenario started stays open.
func closeNewSandboxIssues(ctx context.Context, repo, scenario string, before map[int]bool) error {
	after, err := openSandboxIssues(ctx, repo)
	if err != nil {
		return err
	}
	var created []int
	for number := range after {
		if !before[number] {
			created = append(created, number)
		}
	}
	sort.Ints(created)
	var failures []string
	for _, number := range created {
		comment := fmt.Sprintf("Closed by the evaluation harness after scenario %s, which created this issue.", scenario)
		if _, err := githubPort.Combined(ctx, "", "issue", "close", strconv.Itoa(number),
			"--repo", repo, "--comment", comment); err != nil {
			failures = append(failures, fmt.Sprintf("#%d: %v", number, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("cannot close sandbox issues in %s: %s", repo, strings.Join(failures, "; "))
	}
	return nil
}
