package strategy

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/graph"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCurrentPolicyIsValid(t *testing.T) {
	report := Validate(repositoryRoot(t))
	if !report.Valid {
		t.Fatalf("current policy is invalid: %s", strings.Join(report.Findings, "; "))
	}
	if report.StrategyCount != 6 || report.RuleCount != 9 {
		t.Fatalf("coverage = %d strategies, %d rules", report.StrategyCount, report.RuleCount)
	}
}

func TestSelectsDeterministicallyForRepresentativeSignals(t *testing.T) {
	root := repositoryRoot(t)
	base := Input{Skill: "debug-code", Impact: Low, Reversibility: High, Ambiguity: Low, SecuritySensitivity: Low, StateMutation: None, EvidenceQuality: Complete, ValidationCost: Low}
	cases := []struct {
		name        string
		input       Input
		wantRule    string
		wantProfile string
		wantTier    string
		wantOutcome string
	}{
		{"small deterministic", base, "default", "low-risk", "tier-1", "selected"},
		{"ambiguous debugging", with(base, func(i *Input) { i.Ambiguity = High }), "ambiguous", "high-risk-deliberated", "tier-2", "selected"},
		{"security sensitive", with(base, func(i *Input) { i.SecuritySensitivity = High }), "security-sensitive", "high-risk-deliberated", "tier-2", "selected"},
		{"migration impact", with(base, func(i *Input) { i.Impact = High; i.StateMutation = Repository }), "high-impact", "high-risk", "tier-2", "selected"},
		{"external mutation", with(Input{Skill: "create-pr", Impact: Low, Reversibility: High, Ambiguity: Low, SecuritySensitivity: Low, StateMutation: External, EvidenceQuality: Complete, ValidationCost: Low}, func(i *Input) {}), "external-mutation", "high-risk", "tier-2", "selected"},
		{"missing evidence", with(base, func(i *Input) { i.EvidenceQuality = Missing }), "missing-evidence", "blocked-evidence", "tier-2", "ask_user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			graphDocument, err := graph.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			first, err := SelectWithPolicy(policy, graphDocument, tc.input)
			if err != nil {
				t.Fatal(err)
			}
			second, err := SelectWithPolicy(policy, graphDocument, tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("decision changed: first=%#v second=%#v", first, second)
			}
			if first.Rule != tc.wantRule || first.Strategy != tc.wantProfile || first.ValidationTier != tc.wantTier || first.Outcome != tc.wantOutcome {
				t.Fatalf("decision = %#v", first)
			}
		})
	}
}

func TestSelectRejectsAuthorityExpansionAndUnsafeValidationOverride(t *testing.T) {
	root := repositoryRoot(t)
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	graphDocument, err := graph.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Skill: "analyze-project", Impact: High, Reversibility: Low, Ambiguity: Low, SecuritySensitivity: High, StateMutation: External, EvidenceQuality: Complete, ValidationCost: High, Overrides: UserChoices{ValidationTier: "tier-1"}}
	decision, err := SelectWithPolicy(policy, graphDocument, input)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != "blocked" || decision.Authority.ExternalMutation != "none" {
		t.Fatalf("decision = %#v", decision)
	}
	if !contains(decision.Overrides.Values, "unsafe-validation-tier-rejected") {
		t.Fatalf("unsafe override was not recorded: %#v", decision.Overrides)
	}
}

func TestValidateRejectsUnsafePolicy(t *testing.T) {
	policy, err := Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	graphDocument, err := graph.Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	policy.Strategies[0].ModelTier = "provider/model"
	policy.Rules[0].Strategy = "missing"
	findings := ValidatePolicy(policy, graphDocument)
	if !containsFinding(findings, "invalid model tier") || !containsFinding(findings, "unknown strategy") {
		t.Fatalf("findings = %v", findings)
	}
}

func TestLoadUsesStrictFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("schema_version: 1\nunknown: true\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(Path)), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("unknown field was accepted: %v", err)
	}
}

func TestValidateRejectsUnboundedStrategy(t *testing.T) {
	policy, err := Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	graphDocument, err := graph.Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	policy.Strategies[0].MaxRetries = 2
	policy.Strategies[0].MaxElapsedMillis = 0
	findings := ValidatePolicy(policy, graphDocument)
	if !containsFinding(findings, "invalid retry or elapsed bound") {
		t.Fatalf("findings = %v", findings)
	}
}

func with(input Input, change func(*Input)) Input {
	change(&input)
	return input
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsFinding(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
