package check

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const badgeURLPrefix = "https://raw.githubusercontent.com/hidekitux/skills/badge-data/"

var endpointBadgeRE = regexp.MustCompile(`!\[[^\]]+\]\(https://img\.shields\.io/endpoint\?url=([^)\s]+)\)`)

var requiredBadgePayloads = []string{
	"fsl-killed.json",
	"fsl-kill-rate.json",
	"fsl-survived.json",
	"fslc-version.json",
	"tests-status.json",
	"tests-run.json",
}

const badgeCheckPassFormat = "All %d README badges reference the badge-data branch.\n"

// CheckMutationBadges requires every README FSL and test badge to be an
// auto-updating endpoint badge on the badge-data branch, returning 0 on
// success or 1 when a payload is missing or non-compliant.
func CheckMutationBadges(root string, out, errOut io.Writer) int {
	path := filepath.Join(root, "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(errOut, "Badge check failed: cannot read README.md: %v\n", err)
		return 1
	}
	text := string(content)

	referenced := map[string]bool{}
	errors := []string{}
	for _, match := range endpointBadgeRE.FindAllStringSubmatch(text, -1) {
		decoded, _ := url.QueryUnescape(match[1])
		if !strings.HasPrefix(decoded, badgeURLPrefix) {
			errors = append(errors, fmt.Sprintf("endpoint badge %q does not point at the badge-data branch", decoded))
			continue
		}
		payload := strings.TrimPrefix(decoded, badgeURLPrefix)
		if !containsString(requiredBadgePayloads, payload) {
			errors = append(errors, fmt.Sprintf("endpoint badge references unknown payload %q", payload))
			continue
		}
		referenced[payload] = true
	}
	missing := []string{}
	for _, payload := range requiredBadgePayloads {
		if !referenced[payload] {
			missing = append(missing, payload)
		}
	}
	if len(missing) > 0 {
		errors = append(errors, "README is missing endpoint badges for: "+strings.Join(missing, ", "))
	}
	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Badge check failed:")
		for _, error := range errors {
			fmt.Fprintf(errOut, "- %s\n", error)
		}
		return 1
	}
	fmt.Fprintf(out, badgeCheckPassFormat, len(requiredBadgePayloads))
	return 0
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
