// Package evidence owns the primitives that more than one evidence producer
// needs. A producer is the package that writes a persisted artifact:
// internal/trace for a structured trace, internal/diagnostic for a validator
// diagnostic, and internal/replay for a replay report.
//
// The package holds only primitives with no producer-specific field. A type
// whose field set differs between producers stays with its producer, because
// merging it would widen a persisted contract.
//
// The redaction rules a producer applies before it persists a value live in
// internal/redact instead, because the provider and policy modules apply the
// same rules and cannot import this package.
package evidence

// DiagnosticRef identifies a diagnostic without persisting its message,
// location, or any other content the diagnostic itself carries.
// internal/trace and internal/diagnostic both alias it.
type DiagnosticRef struct {
	Producer string `json:"producer"`
	Code     string `json:"code"`
}
