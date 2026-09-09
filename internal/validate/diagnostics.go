package validate

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hidekitux/skills/internal/diagnostic"
)

const (
	BranchPolicyDiagnosticCode = "validate-branch-policy.invalid"
	IssueBodyDiagnosticCode    = "validate-issue-body.invalid"
	HostsDiagnosticCode        = "validate-hosts.installation"
)

// BranchPolicyDiagnostics runs the branch-policy validator and maps its
// failure to a safe diagnostic while leaving detailed output available to the
// caller through the existing validator API.
func BranchPolicyDiagnostics(configPath, base, head, body string, validateConfig bool) ([]diagnostic.Diagnostic, int, string, string) {
	var out, errOut bytes.Buffer
	code := CheckBranchPolicy(configPath, base, head, body, validateConfig, &out, &errOut)
	if code == 0 {
		return nil, code, out.String(), errOut.String()
	}
	category := diagnostic.ValidationFailure
	remediation := diagnostic.FixRepository
	if code == 2 {
		category = diagnostic.UnsupportedInput
		remediation = diagnostic.InspectInput
	}
	message := firstDiagnosticLine(errOut.String(), "branch policy validation failed")
	item, err := diagnostic.New(diagnostic.Diagnostic{
		Producer:      "validate-branch-policy",
		Code:          BranchPolicyDiagnosticCode,
		Category:      category,
		SourceCommand: "cmd/validate-branch-policy",
		Message:       message,
		Rule:          "branch_policy.route",
		Expected:      "an allowed branch direction and Issue linkage",
		Observed:      fmt.Sprintf("exit_code=%d", code),
		Retryable:     false,
		Remediation:   remediation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	})
	if err != nil {
		return nil, code, out.String(), errOut.String()
	}
	return []diagnostic.Diagnostic{item}, code, out.String(), errOut.String()
}

// IssueBodyDiagnostics maps each Issue-body validation finding to one
// structured diagnostic. The original prose findings remain available from
// IssueBodyValidationErrors for callers that need them.
func IssueBodyDiagnostics(title, body string) ([]diagnostic.Diagnostic, error) {
	findings := IssueBodyValidationErrors(title, body)
	result := make([]diagnostic.Diagnostic, 0, len(findings))
	for _, finding := range findings {
		item, err := diagnostic.New(diagnostic.Diagnostic{
			Producer:      "validate-issue-body",
			Code:          IssueBodyDiagnosticCode,
			Category:      diagnostic.ValidationFailure,
			SourceCommand: "cmd/validate-issue-body",
			Message:       finding,
			Rule:          "issue_body.structure",
			Expected:      "the required Issue body structure",
			Observed:      "issue body rejected",
			Retryable:     false,
			Remediation:   diagnostic.FixRepository,
			Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

// HostsDiagnostics runs host installation validation and returns one
// infrastructure diagnostic when a supported host cannot install the skills.
func HostsDiagnostics(root string) ([]diagnostic.Diagnostic, int, string, string) {
	var out, errOut bytes.Buffer
	code := CheckHosts(root, &out, &errOut)
	if code == 0 {
		return nil, code, out.String(), errOut.String()
	}
	item, err := diagnostic.New(diagnostic.Diagnostic{
		Producer:      "validate-hosts",
		Code:          HostsDiagnosticCode,
		Category:      diagnostic.InfrastructureError,
		SourceCommand: "cmd/validate-hosts",
		Message:       "host skill installation could not be validated",
		Rule:          "host.installation",
		Expected:      "every supported host installs every cataloged skill",
		Observed:      fmt.Sprintf("exit_code=%d", code),
		Retryable:     true,
		Remediation:   diagnostic.RetryOperation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	})
	if err != nil {
		return nil, code, out.String(), errOut.String()
	}
	return []diagnostic.Diagnostic{item}, code, out.String(), errOut.String()
}

func firstDiagnosticLine(text, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "error:")
		if line != "" {
			return strings.TrimSpace(line)
		}
	}
	return fallback
}
