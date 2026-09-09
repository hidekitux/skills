package fsl

import (
	"fmt"
	"path/filepath"

	"github.com/hidekitux/skills/internal/diagnostic"
)

const (
	MutationSurvivorDiagnosticCode       = "fsl.mutation.survivor"
	MutationInfrastructureDiagnosticCode = "fsl.mutation.infrastructure"
	VerificationDiagnosticCode           = "fsl.verify.failed"
)

// DiagnosticsForReport maps non-success FSL mutation results to diagnostics.
// The spec path is kept as a safe evidence reference; fslc output stays in the
// detailed report and never enters the diagnostic.
func DiagnosticsForReport(report MutationReport) ([]diagnostic.Diagnostic, error) {
	result := []diagnostic.Diagnostic{}
	for _, spec := range report.Specs {
		if spec.Status == "error" {
			item, err := diagnostic.New(diagnostic.Diagnostic{
				Producer:      "fsl",
				Code:          MutationInfrastructureDiagnosticCode,
				Category:      diagnostic.InfrastructureError,
				SourceCommand: "cmd/mutate-fsl",
				Message:       "FSL mutation command failed",
				Rule:          "fsl.mutation.complete",
				Expected:      "fslc returns a mutation report",
				Observed:      fmt.Sprintf("spec=%s status=error", filepath.ToSlash(spec.Spec)),
				Evidence:      []diagnostic.Evidence{{Kind: "path", Ref: filepath.ToSlash(spec.Spec)}},
				Retryable:     true,
				Remediation:   diagnostic.RetryOperation,
				Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
			})
			if err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		if spec.Survived > 0 {
			item, err := diagnostic.New(diagnostic.Diagnostic{
				Producer:      "fsl",
				Code:          MutationSurvivorDiagnosticCode,
				Category:      diagnostic.ValidationFailure,
				SourceCommand: "cmd/mutate-fsl",
				Message:       "FSL mutation survivors remain",
				Invariant:     "fsl.mutation_survivors_are_zero",
				Expected:      "survived=0",
				Observed:      fmt.Sprintf("spec=%s survived=%d", filepath.ToSlash(spec.Spec), spec.Survived),
				Evidence:      []diagnostic.Evidence{{Kind: "path", Ref: filepath.ToSlash(spec.Spec)}},
				Retryable:     false,
				Remediation:   diagnostic.FixRepository,
				Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
			})
			if err != nil {
				return nil, err
			}
			result = append(result, item)
		}
	}
	return result, nil
}

// VerificationDiagnostic maps a failed fslc verification to a safe
// infrastructure diagnostic. The verifier's complete output remains with the
// caller and is not copied into the diagnostic.
func VerificationDiagnostic(exitCode int) (diagnostic.Diagnostic, error) {
	return diagnostic.New(diagnostic.Diagnostic{
		Producer:      "fsl",
		Code:          VerificationDiagnosticCode,
		Category:      diagnostic.InfrastructureError,
		SourceCommand: "cmd/verify-fsl",
		Message:       "FSL verification command failed",
		Rule:          "fsl.specification.valid",
		Expected:      "fslc verifies every selected specification",
		Observed:      fmt.Sprintf("exit_code=%d", exitCode),
		Retryable:     true,
		Remediation:   diagnostic.RetryOperation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	})
}
