package validate

import (
	"fmt"
	"regexp"
	"strings"
)

var commonIssueH2 = []string{"Context", "Goal", "Scope", "Acceptance criteria", "Validation"}
var changelogH3 = []string{"Added", "Changed", "Fixed", "Removed"}

var h2RE = regexp.MustCompile(`(?m)^##[ \t]+([^\r\n]+?)[ \t]*$`)
var h3RE = regexp.MustCompile(`(?m)^###[ \t]+([^\r\n]+?)[ \t]*$`)
var checkboxRE = regexp.MustCompile(`(?m)^- \[[ xX]\][ \t]+\S`)
var scopeMarkerRE = regexp.MustCompile(`(?m)^- (In|Out):(?:[ \t].*)?$`)
var commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)

type heading struct {
	start, end int
	name       string
}

// headingMatch records the full-match span and the captured heading name.
func collectHeadings(re *regexp.Regexp, body string) []heading {
	var out []heading
	for _, m := range re.FindAllStringSubmatchIndex(body, -1) {
		out = append(out, heading{start: m[0], end: m[1], name: strings.TrimSpace(body[m[2]:m[3]])})
	}
	return out
}

func withoutComments(text string) string {
	return commentRE.ReplaceAllString(text, "")
}

func sectionContent(body string, headings []heading, index int) string {
	end := len(body)
	if index+1 < len(headings) {
		end = headings[index+1].start
	}
	return strings.TrimSpace(withoutComments(body[headings[index].end:end]))
}

// IssueBodyValidationErrors returns the validation problems for an Issue body.
func IssueBodyValidationErrors(title, body string) []string {
	release := strings.HasPrefix(title, "[Release]:")
	expectedH2 := append([]string{}, commonIssueH2...)
	if release {
		expectedH2 = append(expectedH2, "Changelog")
	}
	h2Headings := collectHeadings(h2RE, body)
	actualH2 := make([]string, 0, len(h2Headings))
	for _, h := range h2Headings {
		actualH2 = append(actualH2, h.name)
	}
	if !stringSliceEqual(actualH2, expectedH2) {
		return []string{"level-two headings must appear exactly once in this order: " + strings.Join(expectedH2, ", ")}
	}

	errors := []string{}
	sections := map[string]string{}
	for index, name := range actualH2 {
		sections[name] = sectionContent(body, h2Headings, index)
	}
	for _, name := range commonIssueH2 {
		if sections[name] == "" {
			errors = append(errors, name+" must contain non-comment content")
		}
	}

	scope := sections["Scope"]
	scopeMatches := scopeMarkerRE.FindAllStringSubmatchIndex(scope, -1)
	markers := make([]string, 0, len(scopeMatches))
	for _, m := range scopeMatches {
		markers = append(markers, scope[m[2]:m[3]])
	}
	if !stringSliceEqual(markers, []string{"In", "Out"}) {
		errors = append(errors, "Scope must contain exactly one - In: then one - Out: marker")
	} else {
		for index, m := range scopeMatches {
			end := len(scope)
			if index+1 < len(scopeMatches) {
				end = scopeMatches[index+1][0]
			}
			line := scope[m[0]:m[1]]
			_, after, _ := strings.Cut(line, ":")
			inline := strings.TrimSpace(after)
			nested := strings.TrimSpace(scope[m[1]:end])
			if inline == "" && nested == "" {
				errors = append(errors, fmt.Sprintf("Scope %s must contain concrete content", markers[index]))
			}
		}
	}

	for _, name := range []string{"Acceptance criteria", "Validation"} {
		if !checkboxRE.MatchString(sections[name]) {
			errors = append(errors, name+" must contain at least one non-empty checkbox")
		}
	}

	h3Headings := collectHeadings(h3RE, body)
	actualH3 := make([]string, 0, len(h3Headings))
	for _, h := range h3Headings {
		actualH3 = append(actualH3, h.name)
	}
	expectedH3 := []string{}
	if release {
		expectedH3 = changelogH3
	}
	if !stringSliceEqual(actualH3, expectedH3) {
		label := "none"
		if len(expectedH3) > 0 {
			label = strings.Join(expectedH3, ", ")
		}
		errors = append(errors, "level-three headings must appear exactly once in this order: "+label)
	} else if release {
		for index, name := range actualH3 {
			if sectionContent(body, h3Headings, index) == "" {
				errors = append(errors, name+" must contain an entry or - None.")
			}
		}
	}
	return errors
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
