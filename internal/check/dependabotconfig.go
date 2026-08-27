package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// dependabotUpdate is one ecosystem entry of .github/dependabot.yml.
type dependabotUpdate struct {
	PackageEcosystem      string             `yaml:"package-ecosystem"`
	Directory             string             `yaml:"directory"`
	Schedule              dependabotSchedule `yaml:"schedule"`
	CommitMessage         *dependabotCommit  `yaml:"commit-message"`
	OpenPullRequestsLimit *int               `yaml:"open-pull-requests-limit"`
}

// dependabotSchedule is the schedule block of a Dependabot update entry.
type dependabotSchedule struct {
	Interval string `yaml:"interval"`
}

// dependabotCommit is the commit-message block of a Dependabot update entry.
type dependabotCommit struct {
	Prefix string `yaml:"prefix"`
}

// CheckDependabotConfig requires the dependency automation (Issue #178) to
// cover the Go module and GitHub Actions with a bounded weekly frequency,
// bounded pull-request limits, and a Conventional Commits prefix. It returns 0
// on success or 1 when an update entry is missing or unbounded.
func CheckDependabotConfig(root string, out, errOut io.Writer) int {
	path := filepath.Join(root, ".github", "dependabot.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(errOut, "Dependabot-config check failed: %v\n", err)
		return 1
	}
	var doc struct {
		Version int                `yaml:"version"`
		Updates []dependabotUpdate `yaml:"updates"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		fmt.Fprintf(errOut, "Dependabot-config check failed: cannot parse dependabot.yml: %v\n", err)
		return 1
	}
	if doc.Version != 2 {
		fmt.Fprintf(errOut, "Dependabot-config check failed: version must be 2, got %d\n", doc.Version)
		return 1
	}
	byEcosystem := map[string]dependabotUpdate{}
	for _, update := range doc.Updates {
		byEcosystem[update.PackageEcosystem] = update
	}
	errors := []string{}
	for _, ecosystem := range []string{"github-actions", "gomod"} {
		update, ok := byEcosystem[ecosystem]
		if !ok {
			errors = append(errors, ecosystem+": missing Dependabot update entry")
			continue
		}
		if update.Directory != "/" {
			errors = append(errors, fmt.Sprintf("%s: directory must be /, got %q", ecosystem, update.Directory))
		}
		if update.Schedule.Interval != "weekly" {
			errors = append(errors, fmt.Sprintf("%s: schedule.interval must be weekly, got %q", ecosystem, update.Schedule.Interval))
		}
		if update.OpenPullRequestsLimit == nil || *update.OpenPullRequestsLimit < 1 || *update.OpenPullRequestsLimit > 5 {
			errors = append(errors, fmt.Sprintf("%s: open-pull-requests-limit must be between 1 and 5", ecosystem))
		}
		if update.CommitMessage == nil || update.CommitMessage.Prefix != "ci" {
			errors = append(errors, fmt.Sprintf("%s: commit-message.prefix must be ci", ecosystem))
		}
	}
	sort.Strings(errors)
	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Dependabot-config check failed:")
		for _, message := range errors {
			fmt.Fprintf(errOut, "- %s\n", message)
		}
		return 1
	}
	fmt.Fprintf(out, "Dependabot-config check passed: %d ecosystem(s) covered with bounded weekly updates.\n", len(byEcosystem))
	return 0
}
