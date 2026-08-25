package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// invalidCommits returns a "SHA: reason" summary for every pull-request commit
// without a GitHub-verified signature.
func invalidCommits(commits []map[string]any) []string {
	invalid := []string{}
	for _, commit := range commits {
		sha, _ := commit["sha"].(string)
		if sha == "" {
			sha = "unknown"
		}
		var verification map[string]any
		if commitData, ok := commit["commit"].(map[string]any); ok {
			verification, _ = commitData["verification"].(map[string]any)
		}
		verified := false
		if v, ok := verification["verified"].(bool); ok {
			verified = v
		}
		if !verified {
			reason := "missing verification"
			if reasonValue, ok := verification["reason"].(string); ok {
				reason = reasonValue
			}
			invalid = append(invalid, fmt.Sprintf("%s: %s", sha, reason))
		}
	}
	return invalid
}

// loadCommitsFixture decodes a local API response fixture into a commit list.
func loadCommitsFixture(path string) ([]map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot load commits fixture: %v", err)
	}
	var payload []map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("cannot load commits fixture: %v", err)
	}
	return payload, nil
}

// fetchPullRequestCommits paginates the GitHub API for a pull request's
// commits with a per-page limit of 100.
func fetchPullRequestCommits(repo string, pullRequest int, token string) ([]map[string]any, error) {
	client := &http.Client{Timeout: 30 * 1e9}
	var commits []map[string]any
	for page := 1; ; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/commits?per_page=100&page=%d", repo, pullRequest, page)
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch pull-request commits: %v", err)
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch pull-request commits: %v", err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("cannot fetch pull-request commits: %v", readErr)
		}
		var payload []map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("GitHub returned an invalid pull-request commits response")
		}
		if payload == nil {
			payload = []map[string]any{}
		}
		commits = append(commits, payload...)
		if len(payload) < 100 {
			return commits, nil
		}
	}
}

// CheckPrCommitSignatures requires every commit in a pull request to have a
// GitHub-verified signature, returning the process exit code.
func CheckPrCommitSignatures(repo string, pullRequest int, commitsJSON string, out, errOut io.Writer) int {
	var commits []map[string]any
	if commitsJSON != "" {
		loaded, err := loadCommitsFixture(commitsJSON)
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 2
		}
		commits = loaded
	} else {
		if repo == "" || strings.Count(repo, "/") != 1 {
			fmt.Fprintln(errOut, "error: --repo must be OWNER/REPO")
			return 2
		}
		if pullRequest < 1 {
			fmt.Fprintln(errOut, "error: --pull-request must be positive")
			return 2
		}
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			fmt.Fprintln(errOut, "error: GITHUB_TOKEN is required")
			return 2
		}
		fetched, err := fetchPullRequestCommits(repo, pullRequest, token)
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 2
		}
		commits = fetched
	}

	if len(commits) == 0 {
		fmt.Fprintln(errOut, "error: pull request contains no commits")
		return 1
	}
	invalid := invalidCommits(commits)
	if len(invalid) > 0 {
		fmt.Fprintln(errOut, "error: pull request contains unverified commits:")
		for _, entry := range invalid {
			fmt.Fprintf(errOut, "- %s\n", entry)
		}
		return 1
	}
	fmt.Fprintf(out, "All %d pull-request commits have GitHub-verified signatures.\n", len(commits))
	return 0
}
