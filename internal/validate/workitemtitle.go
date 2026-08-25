package validate

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const dependabotAuthor = "dependabot[bot]"

const titleTypes = "Feature|Bug|Improvement|Documentation|Security|Maintenance"

var simpleTitleWordRE = regexp.MustCompile(`^[A-Z][a-z]+$`)

var preservedTitleWords = map[string]bool{
	"Actions": true, "Android": true, "Apple": true, "Codex": true,
	"GitHub": true, "Google": true, "iPhone": true, "iOS": true,
	"macOS": true, "OpenAI": true, "Windows": true,
}

func isDependabotAuthor(author string) bool {
	return author == dependabotAuthor
}

// sentenceCaseSummary flags sentence-case violations that are mechanically
// detectable: a run of three or more ordinary capitalized words after the
// leading imperative verb.
func sentenceCaseSummary(summary string) []string {
	problems := []string{}
	words := []string{}
	for _, word := range regexp.MustCompile(`[^A-Za-z0-9]+`).Split(summary, -1) {
		if word != "" {
			words = append(words, word)
		}
	}
	if len(words) == 0 {
		return problems
	}
	run := 0
	runStart := 1
	flush := func() {
		if run >= 3 {
			problems = append(problems, fmt.Sprintf(
				"capitalize later words only when ordinary English requires; title-case run: %q",
				words[runStart:runStart+run],
			))
		}
		run = 0
	}
	for index, word := range words[1:] {
		start := index + 1
		if simpleTitleWordRE.MatchString(word) && !preservedTitleWords[word] {
			if run == 0 {
				runStart = start
			}
			run++
		} else {
			flush()
		}
	}
	flush()
	return problems
}

func summaryErrors(summary string) []string {
	if summary == "" {
		return []string{"summary must contain at least one non-empty word after the [Type]: prefix"}
	}
	first := summary[0]
	if first < 'A' || first > 'Z' {
		return []string{"summary must begin with a capital letter"}
	}
	if summary == "Add" {
		return []string{`summary must name a concrete outcome; the bare template suffix "Add" is a placeholder`}
	}
	return sentenceCaseSummary(summary)
}

func titleErrors(title string) []string {
	release := regexp.MustCompile(`^\[Release\]: v[0-9]+\.[0-9]+\.[0-9]+(?:\+[0-9A-Za-z.-]+)?$`)
	standard := regexp.MustCompile(`^\[(?:` + titleTypes + `)\]: .*$`)
	errors := []string{}
	if release.MatchString(title) {
		return errors
	}
	if !standard.MatchString(title) {
		return []string{
			"title must be [Type]: Summary beginning with a capital letter; Release uses [Release]: vX.Y.Z or [Release]: vX.Y.Z+N",
		}
	}
	summary := strings.TrimSpace(strings.SplitN(title, ":", 2)[1])
	return summaryErrors(summary)
}

// CheckWorkItemTitle validates shared Issue and Pull Request title
// conventions, returning 0 on success or 1 on failure.
func CheckWorkItemTitle(title, author string, out, errOut io.Writer) int {
	if isDependabotAuthor(author) {
		fmt.Fprintln(out, "Dependabot pull request title exemption applies.")
		return 0
	}
	errors := titleErrors(title)
	if len(errors) > 0 {
		fmt.Fprintf(errOut, "error: %s\n", strings.Join(errors, "; "))
		return 1
	}
	fmt.Fprintln(out, "Work item title is valid.")
	return 0
}
