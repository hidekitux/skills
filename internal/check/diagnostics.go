package check

import (
	"fmt"

	"github.com/hidekitux/skills/internal/diagnostic"
)

// DiagnosticForResult maps one repository check result to a stable diagnostic
// without copying the check's detailed output.
func DiagnosticForResult(name string, exitCode int) (diagnostic.Diagnostic, error) {
	return diagnostic.New(diagnostic.Diagnostic{
		Producer:      "check-repository",
		Code:          "check-repository." + name + ".failed",
		Category:      diagnostic.ValidationFailure,
		SourceCommand: "cmd/check-repository",
		Message:       "repository check failed: " + name,
		Rule:          "repository.checks",
		Expected:      "all repository checks pass",
		Observed:      fmt.Sprintf("%s exit_code=%d", name, exitCode),
		Retryable:     false,
		Remediation:   diagnostic.FixRepository,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	})
}
