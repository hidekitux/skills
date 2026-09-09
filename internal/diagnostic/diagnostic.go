// Package diagnostic defines the versioned, redacted validator diagnostic
// contract.
package diagnostic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const SchemaVersion = 1

type Category string

const (
	ValidationFailure   Category = "validation_failure"
	InfrastructureError Category = "infrastructure_error"
	UnsupportedInput    Category = "unsupported_input"
	IncompleteEvidence  Category = "incomplete_evidence"
)

type Remediation string

const (
	InspectInput     Remediation = "inspect_input"
	RetryOperation   Remediation = "retry_operation"
	FixConfiguration Remediation = "fix_configuration"
	FixRepository    Remediation = "fix_repository"
	RequestAuthority Remediation = "request_authority"
	ProvideContext   Remediation = "provide_context"
	ContactOwner     Remediation = "contact_owner"
)

type Location struct {
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type Evidence struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type RedactionSummary struct {
	Mode          string   `json:"mode"`
	RedactedCount int      `json:"redacted_count"`
	OmittedFields []string `json:"omitted_fields"`
}

// Diagnostic is one safe, versioned account of a validation or execution
// problem. Fields intentionally contain summaries and references, not raw
// command output, prompts, reasoning, source content, credentials, or user
// data.
type Diagnostic struct {
	SchemaVersion int              `json:"schema_version"`
	Producer      string           `json:"producer"`
	Code          string           `json:"code"`
	Category      Category         `json:"category"`
	SourceCommand string           `json:"source_command"`
	Message       string           `json:"message"`
	Location      *Location        `json:"location,omitempty"`
	Rule          string           `json:"rule,omitempty"`
	Invariant     string           `json:"invariant,omitempty"`
	Expected      string           `json:"expected,omitempty"`
	Observed      string           `json:"observed,omitempty"`
	Evidence      []Evidence       `json:"evidence,omitempty"`
	Retryable     bool             `json:"retryable"`
	Remediation   Remediation      `json:"remediation"`
	Redaction     RedactionSummary `json:"redaction"`
}

// DiagnosticRef is the only diagnostic data stored in a structured trace.
type DiagnosticRef struct {
	Producer string `json:"producer"`
	Code     string `json:"code"`
}

type Filter struct {
	Producers    []string
	Codes        []string
	Categories   []Category
	Rules        []string
	Invariants   []string
	Remediations []Remediation
	Retryable    *bool
}

var (
	identifierRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._/:+-]*$`)
	textRE       = regexp.MustCompile(`^[^\r\n]+$`)
	credentialRE = regexp.MustCompile(`(?i)(bearer\s+|password\s*=\s*|token\s*=\s*|secret\s*=\s*|api[_-]?key\s*=\s*)([^\s,;]+)|(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+|AKIA[0-9A-Z]{16})`)
	urlRE        = regexp.MustCompile(`https?://[^\s"']+`)
)

// New sanitizes and validates a diagnostic before a caller persists or
// renders it.
func New(input Diagnostic) (Diagnostic, error) {
	clean, err := Sanitize(input)
	if err != nil {
		return Diagnostic{}, err
	}
	if findings := Validate(clean); len(findings) > 0 {
		return Diagnostic{}, errors.New(strings.Join(findings, "; "))
	}
	return clean, nil
}

// Validate returns deterministic contract findings. It never prints the
// observed value, because the value may be unsafe input.
func Validate(d Diagnostic) []string {
	findings := []string{}
	if d.SchemaVersion != SchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version %d is unsupported", d.SchemaVersion))
	}
	for name, value := range map[string]string{
		"producer":       d.Producer,
		"code":           d.Code,
		"source_command": d.SourceCommand,
	} {
		if !validIdentifier(value) {
			findings = append(findings, name+" is missing or invalid")
		}
	}
	if !validCategory(d.Category) {
		findings = append(findings, "category is invalid")
	}
	if !validText(d.Message, 512) {
		findings = append(findings, "message is missing or invalid")
	}
	if d.Location != nil {
		if d.Location.Path == "" || len(d.Location.Path) > 512 || strings.HasPrefix(d.Location.Path, "/") || hasParentPath(d.Location.Path) || strings.ContainsAny(d.Location.Path, "\r\n") {
			findings = append(findings, "location.path is unsafe or invalid")
		}
		if d.Location.Line < 0 || d.Location.Column < 0 || (d.Location.Column > 0 && d.Location.Line == 0) {
			findings = append(findings, "location line and column are invalid")
		}
	}
	for name, value := range map[string]string{
		"rule":      d.Rule,
		"invariant": d.Invariant,
		"expected":  d.Expected,
		"observed":  d.Observed,
	} {
		if value != "" && !validText(value, 512) {
			findings = append(findings, name+" is invalid")
		}
	}
	if len(d.Evidence) > 8 {
		findings = append(findings, "evidence contains more than 8 references")
	}
	for index, evidence := range d.Evidence {
		if !validEvidenceKind(evidence.Kind) {
			findings = append(findings, fmt.Sprintf("evidence[%d].kind is invalid", index))
		}
		if !validEvidenceRef(evidence.Kind, evidence.Ref) {
			findings = append(findings, fmt.Sprintf("evidence[%d].ref is unsafe or invalid", index))
		}
	}
	if !validRemediation(d.Remediation) {
		findings = append(findings, "remediation is invalid")
	}
	findings = append(findings, validateRedaction(d.Redaction)...)
	return findings
}

// Sanitize returns an allowlisted copy. Unsafe evidence is omitted, while an
// unsafe identity field returns an error so a stable code cannot change during
// redaction.
func Sanitize(input Diagnostic) (Diagnostic, error) {
	output := input
	output.SchemaVersion = SchemaVersion
	output.Redaction = RedactionSummary{Mode: "allowlist", OmittedFields: []string{}}
	redact := func(field *string) {
		clean, changed := redactText(*field)
		if changed {
			output.Redaction.RedactedCount++
		}
		*field = clean
	}
	for name, field := range map[string]*string{
		"producer":       &output.Producer,
		"code":           &output.Code,
		"source_command": &output.SourceCommand,
	} {
		clean, changed := redactText(*field)
		if changed {
			return Diagnostic{}, fmt.Errorf("%s contains unsafe data", name)
		}
		*field = clean
	}
	for _, field := range []*string{&output.Message, &output.Rule, &output.Invariant, &output.Expected, &output.Observed} {
		redact(field)
	}
	if output.Location != nil {
		location := *output.Location
		if strings.HasPrefix(location.Path, "/") || hasParentPath(location.Path) || strings.ContainsAny(location.Path, "\r\n") {
			output.Location = nil
			output.Redaction.RedactedCount++
			output.Redaction.OmittedFields = append(output.Redaction.OmittedFields, "location")
		}
	}
	output.Evidence = output.Evidence[:0]
	for index, evidence := range input.Evidence {
		cleanRef, changed := redactText(evidence.Ref)
		if changed || !validEvidenceRef(evidence.Kind, cleanRef) {
			output.Redaction.RedactedCount++
			output.Redaction.OmittedFields = append(output.Redaction.OmittedFields, fmt.Sprintf("evidence[%d]", index))
			continue
		}
		output.Evidence = append(output.Evidence, Evidence{Kind: evidence.Kind, Ref: cleanRef})
	}
	output.Redaction.OmittedFields = uniqueSorted(output.Redaction.OmittedFields)
	return output, nil
}

// RenderText renders all present semantic fields from one diagnostic value.
func RenderText(d Diagnostic) string {
	parts := []string{"[" + d.Producer + "/" + d.Code + "]", string(d.Category) + ":", d.Message}
	if d.SourceCommand != "" {
		parts = append(parts, "command="+d.SourceCommand)
	}
	if d.Location != nil {
		location := d.Location.Path
		if d.Location.Line > 0 {
			location += ":" + strconv.Itoa(d.Location.Line)
			if d.Location.Column > 0 {
				location += ":" + strconv.Itoa(d.Location.Column)
			}
		}
		parts = append(parts, "location="+location)
	}
	for _, field := range []struct{ name, value string }{
		{"rule", d.Rule}, {"invariant", d.Invariant}, {"expected", d.Expected}, {"observed", d.Observed},
	} {
		if field.value != "" {
			parts = append(parts, field.name+"="+field.value)
		}
	}
	if len(d.Evidence) > 0 {
		references := make([]string, 0, len(d.Evidence))
		for _, evidence := range d.Evidence {
			references = append(references, evidence.Kind+":"+evidence.Ref)
		}
		parts = append(parts, "evidence="+strings.Join(references, ","))
	}
	parts = append(parts, "retryable="+strconv.FormatBool(d.Retryable), "remediation="+string(d.Remediation))
	return strings.Join(parts, " ")
}

// Marshal returns one sanitized JSON diagnostic.
func Marshal(d Diagnostic) ([]byte, error) {
	clean, err := New(d)
	if err != nil {
		return nil, err
	}
	return json.Marshal(clean)
}

// WriteJSON writes one sanitized JSON diagnostic.
func WriteJSON(w io.Writer, d Diagnostic) error {
	encoded, err := Marshal(d)
	if err != nil {
		return err
	}
	_, err = w.Write(append(encoded, '\n'))
	return err
}

// WriteJSONL writes one sanitized diagnostic per line.
func WriteJSONL(w io.Writer, diagnostics []Diagnostic) error {
	for _, d := range diagnostics {
		if err := WriteJSON(w, d); err != nil {
			return err
		}
	}
	return nil
}

// ReadJSONL reads strict JSONL diagnostics and validates every record.
func ReadJSONL(data []byte) ([]Diagnostic, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	result := []Diagnostic{}
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var diagnostic Diagnostic
		if err := decodeStrict([]byte(text), &diagnostic); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if findings := Validate(diagnostic); len(findings) > 0 {
			return nil, fmt.Errorf("line %d: %s", line, strings.Join(findings, "; "))
		}
		result = append(result, diagnostic)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("diagnostic stream contains no records")
	}
	return result, nil
}

// Select returns diagnostics that match at least one supplied structured
// filter. An empty filter matches every diagnostic. Within a field, values are
// ORed; across fields, values are ANDed.
func Select(diagnostics []Diagnostic, filter Filter) []Diagnostic {
	selected := []Diagnostic{}
	for _, diagnostic := range diagnostics {
		if !matchesFilter(diagnostic, filter) {
			continue
		}
		selected = append(selected, diagnostic)
	}
	return selected
}

func matchesFilter(d Diagnostic, filter Filter) bool {
	if len(filter.Producers) > 0 && !containsString(filter.Producers, d.Producer) {
		return false
	}
	if len(filter.Codes) > 0 && !containsString(filter.Codes, d.Code) {
		return false
	}
	if len(filter.Categories) > 0 && !containsCategory(filter.Categories, d.Category) {
		return false
	}
	if len(filter.Rules) > 0 && !containsString(filter.Rules, d.Rule) {
		return false
	}
	if len(filter.Invariants) > 0 && !containsString(filter.Invariants, d.Invariant) {
		return false
	}
	if len(filter.Remediations) > 0 && !containsRemediation(filter.Remediations, d.Remediation) {
		return false
	}
	return filter.Retryable == nil || *filter.Retryable == d.Retryable
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func redactText(value string) (string, bool) {
	if value == "" {
		return value, false
	}
	changed := false
	clean := credentialRE.ReplaceAllStringFunc(value, func(string) string {
		changed = true
		return "[REDACTED]"
	})
	clean = urlRE.ReplaceAllStringFunc(clean, func(match string) string {
		parsed, err := url.Parse(match)
		if err == nil && parsed.Scheme == "https" && parsed.Host == "github.com" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" {
			return match
		}
		changed = true
		return "[REDACTED_URL]"
	})
	if strings.Contains(clean, "-----BEGIN") {
		changed = true
		clean = regexp.MustCompile(`-----BEGIN [^-]+-----[\s\S]*?-----END [^-]+-----`).ReplaceAllString(clean, "[REDACTED_KEY]")
	}
	return clean, changed
}

func validIdentifier(value string) bool {
	return value != "" && len(value) <= 128 && identifierRE.MatchString(value)
}

func validText(value string, max int) bool {
	return value != "" && len(value) <= max && textRE.MatchString(value)
}

func validCategory(value Category) bool {
	return value == ValidationFailure || value == InfrastructureError || value == UnsupportedInput || value == IncompleteEvidence
}

func validRemediation(value Remediation) bool {
	switch value {
	case InspectInput, RetryOperation, FixConfiguration, FixRepository, RequestAuthority, ProvideContext, ContactOwner:
		return true
	default:
		return false
	}
}

func validEvidenceKind(value string) bool {
	switch value {
	case "path", "command", "commit", "issue", "pull_request", "validation", "trace":
		return true
	default:
		return false
	}
}

func validEvidenceRef(kind, value string) bool {
	if !validText(value, 512) || credentialRE.MatchString(value) {
		return false
	}
	switch kind {
	case "path":
		return !strings.HasPrefix(value, "/") && !hasParentPath(value)
	case "command", "validation", "trace":
		return validIdentifier(value)
	case "commit":
		return regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(value)
	case "issue", "pull_request":
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return false
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		want := "issues"
		if kind == "pull_request" {
			want = "pull"
		}
		return len(parts) == 4 && parts[0] != "" && parts[1] != "" && parts[2] == want && parts[3] != ""
	default:
		return false
	}
}

func hasParentPath(value string) bool {
	for _, part := range strings.Split(strings.ReplaceAll(value, "\\", "/"), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func validateRedaction(redaction RedactionSummary) []string {
	findings := []string{}
	if redaction.Mode != "allowlist" {
		findings = append(findings, "redaction.mode must be allowlist")
	}
	if redaction.RedactedCount < 0 {
		findings = append(findings, "redaction.redacted_count must not be negative")
	}
	if !sort.StringsAreSorted(redaction.OmittedFields) {
		findings = append(findings, "redaction.omitted_fields must be sorted")
	}
	for _, field := range redaction.OmittedFields {
		if !validIdentifier(field) {
			findings = append(findings, "redaction.omitted_fields contains an invalid field")
		}
	}
	return findings
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsCategory(values []Category, want Category) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsRemediation(values []Remediation, want Remediation) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// IsPrivateHost is exported for adapters that receive URLs from external
// tools and must reject local or private addresses before creating evidence.
func IsPrivateHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback()
	}
	return host == "localhost" || strings.HasSuffix(host, ".local")
}
