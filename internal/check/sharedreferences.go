package check

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/hidekitux/skills/internal/discover"
)

// sharedReferenceDir holds the source of every reference file that several
// skills share. Installation copies only a skill's own directory, so each
// consuming skill keeps a byte-identical copy below its references/.
const sharedReferenceDir = "shared/references"

// linksReference reports whether SKILL.md text names the path
// references/<name>, optionally prefixed with ./, as a whole path. Any mention
// counts, whatever Markdown link form surrounds it, so a link is never missed
// because of its syntax; a longer path such as other/references/<name> or
// references/<name>.bak does not count.
func linksReference(text, name string) bool {
	path := regexp.MustCompile(`(?:^|[^A-Za-z0-9_./-])(?:\./)?references/` + regexp.QuoteMeta(name) + `(?:$|[^A-Za-z0-9_.-]|\.(?:$|[^A-Za-z0-9_-]))`)
	return path.MatchString(text)
}

// CheckSharedReferences compares every skill's copy of a shared reference
// with its source. A skill consumes a shared reference when its SKILL.md
// names the path references/<name>. A drifted or missing copy fails the check
// with the cp command that repairs it. A copy that SKILL.md never names fails
// too, because it would ship unchecked; the fix is to name it or remove it.
// It returns 0 on success.
func CheckSharedReferences(root string, out, errOut io.Writer) int {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(sharedReferenceDir)))
	if os.IsNotExist(err) {
		fmt.Fprintln(out, "Shared-reference check skipped: no shared/references directory.")
		return 0
	}
	if err != nil {
		fmt.Fprintf(errOut, "Shared-reference check failed: %v\n", err)
		return 1
	}
	sources := map[string][]byte{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sharedReferenceDir), entry.Name()))
		if err != nil {
			fmt.Fprintf(errOut, "Shared-reference check failed: %v\n", err)
			return 1
		}
		sources[entry.Name()] = content
	}

	findings := []string{}
	copies := 0
	for _, skill := range discover.All(root) {
		instructions, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(skill.Dir), "SKILL.md"))
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s/SKILL.md cannot be read: %v", skill.Dir, err))
			continue
		}
		for name, source := range sources {
			copyPath := skill.Dir + "/references/" + name
			if !linksReference(string(instructions), name) {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(copyPath))); err == nil {
					findings = append(findings, fmt.Sprintf("%s is not named in %s/SKILL.md; name it there or remove the copy", copyPath, skill.Dir))
				}
				continue
			}
			repair := fmt.Sprintf("cp %s/%s %s", sharedReferenceDir, name, copyPath)
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(copyPath)))
			switch {
			case os.IsNotExist(err):
				findings = append(findings, fmt.Sprintf("%s is missing; run %s", copyPath, repair))
			case err != nil:
				findings = append(findings, fmt.Sprintf("%s cannot be read: %v", copyPath, err))
			case !bytes.Equal(content, source):
				findings = append(findings, fmt.Sprintf("%s differs from %s/%s; run %s", copyPath, sharedReferenceDir, name, repair))
			default:
				copies++
			}
		}
	}
	if len(findings) > 0 {
		sort.Strings(findings)
		fmt.Fprintln(errOut, "Shared-reference check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintf(out, "Shared-reference check passed: %d copy(ies) of %d shared reference(s) match their source.\n", copies, len(sources))
	return 0
}
