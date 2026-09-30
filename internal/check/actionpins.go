package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	// pinnedUsesPattern matches a uses: line whose action reference is pinned
	// to a full commit SHA, capturing the text after the SHA.
	pinnedUsesPattern = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*[^\s@$.][^\s@]*@[0-9a-f]{40}(.*)$`)
	// versionCommentPattern matches the trailing version comment that
	// Dependabot rewrites together with the SHA.
	versionCommentPattern = regexp.MustCompile(`^\s+#\s*v[0-9]+(?:\.[0-9]+)*\s*$`)
	// commentSHAPattern matches a comment that contains a full commit SHA.
	commentSHAPattern = regexp.MustCompile(`#.*\b[0-9a-f]{40}\b`)
)

// CheckActionPins requires every GitHub Actions step pinned to a commit SHA
// (Issue #424) to carry its release on the same line, as
// `uses: <action>@<sha> # v<version>`, which Dependabot updates with the SHA.
// It rejects any comment that repeats a SHA, because such a comment is not
// updated and drifts from the pin. It returns 0 on success or 1 when a pin
// lacks its version comment or a comment contains a SHA.
func CheckActionPins(root string, out, errOut io.Writer) int {
	files := []string{}
	for _, pattern := range []string{
		filepath.Join(root, ".github", "workflows", "*.yml"),
		filepath.Join(root, ".github", "actions", "*", "action.yml"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Fprintf(errOut, "Action-pin check failed: %v\n", err)
			return 1
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	errors := []string{}
	pins := 0
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(errOut, "Action-pin check failed: %v\n", err)
			return 1
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			name = path
		}
		name = filepath.ToSlash(name)
		for index, line := range strings.Split(string(content), "\n") {
			location := fmt.Sprintf("%s:%d", name, index+1)
			if match := pinnedUsesPattern.FindStringSubmatch(line); match != nil {
				pins++
				if !versionCommentPattern.MatchString(match[1]) {
					errors = append(errors, location+": pinned uses: needs a trailing `# v<version>` comment")
				}
				continue
			}
			if commentSHAPattern.MatchString(line) {
				errors = append(errors, location+": comment repeats a commit SHA; put the version on the uses: line instead")
			}
		}
	}
	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Action-pin check failed:")
		for _, message := range errors {
			fmt.Fprintf(errOut, "- %s\n", message)
		}
		return 1
	}
	fmt.Fprintf(out, "Action-pin check passed: %d pinned action reference(s) carry a version comment.\n", pins)
	return 0
}
