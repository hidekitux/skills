package validate

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

var branchPolicyH2RE = regexp.MustCompile(`(?m)^##[ \t]+([^\r\n]+?)[ \t]*$`)
var anyClosingLineRE = regexp.MustCompile(`(?im)^\s*(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#([1-9][0-9]*)\s*$`)
var anyReferenceLineRE = regexp.MustCompile(`(?im)^\s*(?:(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)|tracks)\s+#[1-9][0-9]*\s*$`)

type branchPolicyHeading struct {
	start, end int
	name       string
}

func branchPolicyHeadings(body string) []branchPolicyHeading {
	var out []branchPolicyHeading
	for _, m := range branchPolicyH2RE.FindAllStringSubmatchIndex(body, -1) {
		out = append(out, branchPolicyHeading{start: m[0], end: m[1], name: strings.TrimSpace(body[m[2]:m[3]])})
	}
	return out
}

// issueReferencesAtStart returns the opening Issue-section links for the
// keyword ("Closes" or "Tracks"), or ok=false for an invalid body.
func issueReferencesAtStart(body, keyword string) ([]int, bool) {
	referencePattern := regexp.MustCompile("^" + regexp.QuoteMeta(keyword) + " #([1-9][0-9]*)$")
	clean := withoutComments(body)
	headings := branchPolicyHeadings(clean)
	if len(headings) == 0 || headings[0].name != "Issue" {
		return nil, false
	}
	if strings.TrimSpace(clean[:headings[0].start]) != "" {
		return nil, false
	}
	end := len(clean)
	if len(headings) > 1 {
		end = headings[1].start
	}
	sectionLines := []string{}
	for _, line := range strings.Split(clean[headings[0].end:end], "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			sectionLines = append(sectionLines, trimmed)
		}
	}
	if len(sectionLines) == 0 {
		return nil, false
	}
	links := []int{}
	for _, line := range sectionLines {
		m := referencePattern.FindStringSubmatch(line)
		if m == nil {
			return nil, false
		}
		n, _ := strconv.Atoi(m[1])
		links = append(links, n)
	}
	if len(anyReferenceLineRE.FindAllStringSubmatch(clean, -1)) != len(links) {
		return nil, false
	}
	return links, true
}

func issueLinksAtStart(body string) ([]int, bool) {
	if !anyClosingLineRE.MatchString(body) {
		return nil, false
	}
	return issueReferencesAtStart(body, "Closes")
}

func matchingIssueLinkStartsBody(body string, issueNumber int) bool {
	links, ok := issueLinksAtStart(body)
	return ok && len(links) > 0 && links[0] == issueNumber
}

// branchPolicyRoute is a single route in .github/branch-policy.toml.
type branchPolicyRoute struct {
	HeadPattern       string `toml:"head_pattern"`
	BasePattern       string `toml:"base_pattern"`
	RequiresIssueLink bool   `toml:"requires_issue_link"`
}

func loadBranchPolicy(path string) ([]branchPolicyRoute, error) {
	var policy struct {
		Routes []branchPolicyRoute `toml:"routes"`
	}
	if err := support.LoadTOMLFile(path, &policy); err != nil {
		return nil, err
	}
	if len(policy.Routes) == 0 {
		return nil, errors.New("policy must define one or more [[routes]]")
	}
	for i, route := range policy.Routes {
		if route.HeadPattern == "" {
			return nil, fmt.Errorf("routes[%d].head_pattern must be non-empty text", i+1)
		}
		if route.BasePattern == "" {
			return nil, fmt.Errorf("routes[%d].base_pattern must be non-empty text", i+1)
		}
		if _, err := regexp.Compile(route.HeadPattern); err != nil {
			return nil, err
		}
		if _, err := regexp.Compile(route.BasePattern); err != nil {
			return nil, err
		}
	}
	return policy.Routes, nil
}

// CheckBranchPolicy validates an issue-based pull-request branch policy,
// returning the process exit code. A load or usage error returns 2; a refused
// direction returns 1; success returns 0.
func CheckBranchPolicy(configPath, base, head, body string, validateConfig bool, out, errOut io.Writer) int {
	routes, err := loadBranchPolicy(configPath)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 2
	}
	if validateConfig {
		fmt.Fprintf(out, "Branch policy configuration is valid: %s\n", configPath)
		return 0
	}
	if base == "" || head == "" {
		fmt.Fprintln(errOut, "error: --base and --head are required")
		return 2
	}
	ok := false
	for _, route := range routes {
		if !regexp.MustCompile(route.HeadPattern).MatchString(head) ||
			!regexp.MustCompile(route.BasePattern).MatchString(base) {
			continue
		}
		if route.RequiresIssueLink {
			parts := strings.Split(head, "/")
			issueNumber, _ := strconv.Atoi(parts[len(parts)-1])
			if !matchingIssueLinkStartsBody(body, issueNumber) {
				continue
			}
		}
		ok = true
		break
	}
	if !ok {
		fmt.Fprintf(errOut, "error: disallowed pull-request direction or issue linkage; "+
			"an Issue-backed Pull Request must start with an Issue section "+
			"whose first content line is the branch Issue's matching Closes "+
			"line, with every additional Closes line kept in that section: "+
			"%s -> %s\n", head, base)
		return 1
	}
	fmt.Fprintf(out, "Allowed pull-request direction: %s -> %s\n", head, base)
	return 0
}
