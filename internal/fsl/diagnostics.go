package fsl

import (
	"fmt"
	"path/filepath"

	"github.com/hidekitux/skills/internal/diagnostic"
)

const (
	MutationSurvivorDiagnosticCode       = "fsl.mutation.survivor"
	MutationInfrastructureDiagnosticCode = "fsl.mutation.infrastructure"
	VerificationValidationDiagnosticCode = "fsl.verify.validation"
	VerificationInfrastructureCode       = "fsl.verify.infrastructure"
	VerificationDiagnosticCode           = "fsl.verify.failed"
)

// DiagnosticsForReport maps non-success FSL mutation results to diagnostics.
// The spec path is kept as a safe evidence reference; fslc output stays in the
// detailed report and never enters the diagnostic.
func DiagnosticsForReport(report MutationReport) ([]diagnostic.Diagnostic, error) {
	result := []diagnostic.Diagnostic{}
	for _, spec := range report.Specs {
		if spec.Status == "error" {
			item, err := mutationInfrastructureDiagnostic(spec.Spec)
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

// MutationInfrastructureDiagnostic maps a mutation invocation failure that
// has no per-spec report to a safe diagnostic.
func MutationInfrastructureDiagnostic(exitCode int) (diagnostic.Diagnostic, error) {
	return diagnostic.New(diagnostic.Diagnostic{
		Producer:      "fsl",
		Code:          MutationInfrastructureDiagnosticCode,
		Category:      diagnostic.InfrastructureError,
		SourceCommand: "cmd/mutate-fsl",
		Message:       "FSL mutation command failed",
		Rule:          "fsl.mutation.complete",
		Expected:      "fslc returns a mutation report",
		Observed:      fmt.Sprintf("exit_code=%d", exitCode),
		Retryable:     true,
		Remediation:   diagnostic.RetryOperation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	})
}

func mutationInfrastructureDiagnostic(spec string) (diagnostic.Diagnostic, error) {
	item := diagnostic.Diagnostic{
		Producer:      "fsl",
		Code:          MutationInfrastructureDiagnosticCode,
		Category:      diagnostic.InfrastructureError,
		SourceCommand: "cmd/mutate-fsl",
		Message:       "FSL mutation command failed",
		Rule:          "fsl.mutation.complete",
		Expected:      "fslc returns a mutation report",
		Observed:      "mutation report unavailable for selected specification",
		Retryable:     true,
		Remediation:   diagnostic.RetryOperation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	}
	if spec != "" {
		item.Evidence = []diagnostic.Evidence{{Kind: "path", Ref: filepath.ToSlash(spec)}}
	}
	return diagnostic.New(item)
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

// VerificationDiagnosticForResult maps the recorded FSL verification phase
// and process state to the appropriate diagnostic category.
func VerificationDiagnosticForResult(result VerificationResult) (diagnostic.Diagnostic, error) {
	category := diagnostic.ValidationFailure
	code := VerificationValidationDiagnosticCode
	retryable := false
	remediation := diagnostic.FixRepository
	message := "FSL specification failed verification"
	if result.Infrastructure {
		category = diagnostic.InfrastructureError
		code = VerificationInfrastructureCode
		retryable = true
		remediation = diagnostic.RetryOperation
		message = "FSL verification tool could not run"
	}
	item := diagnostic.Diagnostic{
		Producer:      "fsl",
		Code:          code,
		Category:      category,
		SourceCommand: "cmd/verify-fsl",
		Message:       message,
		Rule:          "fsl.specification.valid",
		Expected:      "fslc verifies every selected specification",
		Observed:      fmt.Sprintf("phase=%s exit_code=%d", result.Phase, result.ExitCode),
		Retryable:     retryable,
		Remediation:   remediation,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	}
	if result.Spec != "" {
		item.Evidence = []diagnostic.Evidence{{Kind: "path", Ref: result.Spec}}
	}
	return diagnostic.New(item)
}
