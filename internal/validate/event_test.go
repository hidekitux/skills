package validate

import (
	"bytes"
	"os"
	"testing"
)

const changeBody = `## Context

The generated work-item structure is ambiguous.

## Goal

Make generated bodies deterministic.

## Scope

- In:
  - Body ordering.
- Out:
  - Prose linting.

## Acceptance criteria

- [ ] Required sections are ordered.

## Validation

- [ ] Run the body validator tests.
`

const releaseBody = `## Context

The verified changes are ready to publish.

## Goal

Publish version v1.2.3.

## Scope

- In:
  - Version v1.2.3.
- Out:
  - Later changes.

## Acceptance criteria

- [ ] The release is published.

## Validation

- [ ] Run the release verification task.

## Changelog

### Added

- Deterministic body validation.

### Changed

- None.

### Fixed

- None.

### Removed

- None.
`

func runEventCheck(t *testing.T, fn func(out, errOut *bytes.Buffer) int) int {
	t.Helper()
	var out, errOut bytes.Buffer
	return fn(&out, &errOut)
}

// --- Branch policy ---

func TestBranchPolicyAcceptsMatchingClosesBlockAsFirstSection(t *testing.T) {
	body := "## Issue\n\nCloses #35\nCloses #36\n\n## Summary\n\n- Standardize generated bodies.\n"
	links, ok := issueLinksAtStart(body)
	if !ok || len(links) != 2 || links[0] != 35 || links[1] != 36 {
		t.Fatalf("unexpected links %v ok=%v", links, ok)
	}
	if !matchingIssueLinkStartsBody(body, 35) {
		t.Fatal("expected first link to match 35")
	}
}

func TestBranchPolicyRejectsWhenBranchIssueIsNotFirst(t *testing.T) {
	body := "## Issue\n\nCloses #36\nCloses #35\n\n## Summary\n"
	if matchingIssueLinkStartsBody(body, 35) {
		t.Fatal("expected no match")
	}
}

func TestBranchPolicyRejectsAdditionalClosesOutsideIssueSection(t *testing.T) {
	body := "## Issue\n\nCloses #35\n\n## Summary\n\nCloses #36\n"
	if matchingIssueLinkStartsBody(body, 35) {
		t.Fatal("expected no match")
	}
}

func TestBranchPolicyRejectsClosesLineAtEnd(t *testing.T) {
	body := "## Summary\n\n- Standardize.\n\nCloses #35\n"
	if _, ok := issueLinksAtStart(body); ok {
		t.Fatal("expected invalid")
	}
}

func TestBranchPolicyRejectsNonReferenceContentInIssueSection(t *testing.T) {
	body := "## Issue\n\nThis work closes the linked Issue.\nCloses #35\n\n## Summary\n"
	if _, ok := issueLinksAtStart(body); ok {
		t.Fatal("expected invalid")
	}
}

func TestBranchPolicyAcceptsReleaseTracksBlockAsFirstSection(t *testing.T) {
	body := "## Issue\n\nTracks #35\n\n## Summary\n\n- Prepare the release.\n"
	links, ok := issueReferencesAtStart(body, "Tracks")
	if !ok || len(links) != 1 || links[0] != 35 {
		t.Fatalf("unexpected links %v ok=%v", links, ok)
	}
}

const testBranchPolicyConfig = `[[routes]]
head_pattern = "^issue/[1-9][0-9]*$"
base_pattern = "^main$"
requires_issue_link = true

[[routes]]
head_pattern = "^dependabot/.+$"
base_pattern = "^main$"
requires_issue_link = false
`

func branchPolicyConfigPath(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/branch-policy.toml"
	if err := os.WriteFile(path, []byte(testBranchPolicyConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBranchPolicyAcceptsDependabotRouteWithoutIssueLink(t *testing.T) {
	config := branchPolicyConfigPath(t)
	code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckBranchPolicy(config, "main", "dependabot/example", "", false, out, errOut)
	})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestBranchPolicyRejectsIssueBranchWithoutMatchingLink(t *testing.T) {
	config := branchPolicyConfigPath(t)
	code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckBranchPolicy(config, "main", "issue/123", "", false, out, errOut)
	})
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

// --- Work item title ---

func TestWorkItemTitleAcceptsSentenceCaseWithProperNouns(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Feature]: Add GitHub Actions login flow", "", out, errOut)
	}); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestWorkItemTitleAcceptsAcronyms(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Improvement]: Add CI validation for YAML workflows", "", out, errOut)
	}); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestWorkItemTitleAcceptsBuildIdentifierRelease(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Release]: v1.2.3+42", "", out, errOut)
	}); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestWorkItemTitleRejectsLowercaseSummaryStart(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Feature]: add user login flow", "", out, errOut)
	}); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestWorkItemTitleRejectsUnknownType(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Enhancement]: Add user login flow", "", out, errOut)
	}); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestWorkItemTitleRejectsTitleCaseSummary(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Feature]: Add User Login Flow", "", out, errOut)
	}); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestWorkItemTitleAcceptsConcreteSummaryWithTemplateVerb(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("[Feature]: Add analyze-project skill", "", out, errOut)
	}); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestWorkItemTitleRejectsEmptySummary(t *testing.T) {
	for _, title := range []string{"[Feature]:", "[Feature]: ", "[Feature]:    "} {
		if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
			return CheckWorkItemTitle(title, "", out, errOut)
		}); code != 1 {
			t.Fatalf("expected 1 for %q, got %d", title, code)
		}
	}
}

func TestWorkItemTitleRejectsBareTemplatePlaceholderSummary(t *testing.T) {
	for _, title := range []string{"[Feature]: Add", "[Feature]: Add ", "[Feature]: Add\t"} {
		if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
			return CheckWorkItemTitle(title, "", out, errOut)
		}); code != 1 {
			t.Fatalf("expected 1 for %q, got %d", title, code)
		}
	}
}

func TestWorkItemTitleAcceptsDependabotTitleWithAuthor(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("Bump actions/checkout from 4 to 5", "dependabot[bot]", out, errOut)
	}); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestWorkItemTitleRejectsBotStyleTitleWithoutDependabotAuthor(t *testing.T) {
	if code := runEventCheck(t, func(out, errOut *bytes.Buffer) int {
		return CheckWorkItemTitle("Bump actions/checkout from 4 to 5", "octocat", out, errOut)
	}); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

// --- Issue body ---

func TestIssueBodyAcceptsOrderedChangeBody(t *testing.T) {
	if errors := IssueBodyValidationErrors("[Improvement]: Standardize bodies", changeBody); len(errors) != 0 {
		t.Fatalf("expected valid, got %v", errors)
	}
}

func TestIssueBodyRejectsReorderedOrDuplicateChangeHeadings(t *testing.T) {
	reordered := replaceFirst(changeBody, "## Context", "## Temporary")
	reordered = replaceFirst(reordered, "## Goal", "## Context")
	reordered = replaceFirst(reordered, "## Temporary", "## Goal")
	duplicate := changeBody + "\n## Validation\n\n- [ ] Run it again.\n"
	if errors := IssueBodyValidationErrors("[Improvement]: Standardize bodies", reordered); len(errors) == 0 {
		t.Fatal("expected reordered headings to fail")
	}
	if errors := IssueBodyValidationErrors("[Improvement]: Standardize bodies", duplicate); len(errors) == 0 {
		t.Fatal("expected duplicate headings to fail")
	}
}

func TestIssueBodyRejectsEmptyChecklistItems(t *testing.T) {
	body := replaceFirst(changeBody, "- [ ] Required sections are ordered.", "- [ ]")
	found := false
	for _, error := range IssueBodyValidationErrors("[Improvement]: Standardize bodies", body) {
		if error == "Acceptance criteria must contain at least one non-empty checkbox" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected empty checkbox error")
	}
}

func TestIssueBodyRejectsEmptyScopeMarkers(t *testing.T) {
	body := replaceFirst(changeBody, "- In:\n  - Body ordering.\n- Out:\n  - Prose linting.", "- In:\n- Out:")
	errors := IssueBodyValidationErrors("[Improvement]: Standardize bodies", body)
	for _, want := range []string{"Scope In must contain concrete content", "Scope Out must contain concrete content"} {
		if !containsErr(errors, want) {
			t.Fatalf("missing error %q in %v", want, errors)
		}
	}
}

func TestIssueBodyAcceptsOrderedReleaseBody(t *testing.T) {
	if errors := IssueBodyValidationErrors("[Release]: v1.2.3", releaseBody); len(errors) != 0 {
		t.Fatalf("expected valid, got %v", errors)
	}
}

func TestIssueBodyRejectsReorderedReleaseChangelog(t *testing.T) {
	body := replaceFirst(releaseBody, "### Added", "### Temporary")
	body = replaceFirst(body, "### Changed", "### Added")
	body = replaceFirst(body, "### Temporary", "### Changed")
	if errors := IssueBodyValidationErrors("[Release]: v1.2.3", body); len(errors) == 0 {
		t.Fatal("expected reordered changelog to fail")
	}
}

// --- Pull-request commit signatures ---

func TestPrSignaturesAcceptsOnlyVerifiedCommits(t *testing.T) {
	commits := []map[string]any{
		{"sha": "verified", "commit": map[string]any{"verification": map[string]any{"verified": true}}},
		{"sha": "unsigned", "commit": map[string]any{"verification": map[string]any{"verified": false, "reason": "unsigned"}}},
		{"sha": "missing", "commit": map[string]any{}},
	}
	invalid := invalidCommits(commits)
	if len(invalid) != 2 || invalid[0] != "unsigned: unsigned" || invalid[1] != "missing: missing verification" {
		t.Fatalf("unexpected invalid list %v", invalid)
	}
}

func replaceFirst(s, old, new string) string {
	index := indexOf(s, old)
	if index < 0 {
		return s
	}
	return s[:index] + new + s[index+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func containsErr(errors []string, target string) bool {
	for _, error := range errors {
		if error == target {
			return true
		}
	}
	return false
}
