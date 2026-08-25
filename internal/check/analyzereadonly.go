package check

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Only analyze-* skills are governed by the read-only contract.
const analyzePrefix = "analyze-"

// Frontmatter is stripped so a defensive summary such as "does not create
// issues" in the description does not trip the instruction detector.
var frontmatterRE = regexp.MustCompile(`(?s)^---\r?\n.*?\r?\n---\r?\n`)

type forbiddenPattern struct {
	label   string
	pattern *regexp.Regexp
}

var forbiddenCreationPatterns = []forbiddenPattern{
	{
		label:   "Issue-creation instruction",
		pattern: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])(?:create|open|file|submit|raise|log)\s+(?:an?\s+|the\s+)?(?:github\s+)?issues?(?:$|[^A-Za-z0-9_])`),
	},
	{
		label:   "Pull-request-creation instruction",
		pattern: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])(?:create|open|submit|raise)\s+(?:an?\s+|the\s+)?(?:github\s+)?pull\s+requests?(?:$|[^A-Za-z0-9_])`),
	},
	{
		label:   "PR-creation instruction",
		pattern: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])(?:create|open|submit|raise)\s+(?:an?\s+|the\s+)?pr(?:$|[^A-Za-z0-9_])`),
	},
}

// skillBody returns the Markdown body of a SKILL.md, excluding frontmatter.
func skillBody(text string) string {
	if loc := frontmatterRE.FindStringIndex(text); loc != nil {
		return text[loc[1]:]
	}
	return text
}

// CheckAnalyzeReadonly rejects Issue- or PR-creation instructions in
// analyze-* skill bodies, returning 0 on success or 1 when findings exist.
func CheckAnalyzeReadonly(root string, out, errOut io.Writer) int {
	findings := []string{}
	skillsRoot := filepath.Join(root, "skills")
	_ = filepath.WalkDir(skillsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		if !strings.HasPrefix(filepath.Base(filepath.Dir(path)), analyzePrefix) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: cannot read: %v", relPath(root, path), err))
			return nil
		}
		if !utf8.Valid(content) {
			findings = append(findings, fmt.Sprintf("%s: cannot read: invalid UTF-8", relPath(root, path)))
			return nil
		}
		body := skillBody(string(content))
		rel := relPath(root, path)
		for number, line := range strings.Split(body, "\n") {
			line = strings.TrimSuffix(line, "\r")
			for _, candidate := range forbiddenCreationPatterns {
				if candidate.pattern.MatchString(line) {
					findings = append(findings, fmt.Sprintf("%s:%d: %s", rel, number+1, candidate.label))
				}
			}
		}
		return nil
	})
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Analyze read-only check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintln(out, "Analyze read-only check passed: no analyze-* skill instructs Issue or PR creation.")
	return 0
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
