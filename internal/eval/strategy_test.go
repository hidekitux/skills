package eval

import (
	"testing"

	executionstrategy "github.com/hidekitux/skills/internal/strategy"
)

func TestSelectExecutionStrategyRecordsAdaptiveBaselineComparison(t *testing.T) {
	root := repositoryRootForEval(t)
	scenario := &Scenario{
		ID:    "strategy-ambiguous",
		Skill: "debug-code",
		ExecutionStrategy: &ExecutionStrategySpec{
			Input: executionstrategy.Input{
				Skill: "debug-code", Impact: executionstrategy.Low, Reversibility: executionstrategy.High,
				Ambiguity: executionstrategy.High, SecuritySensitivity: executionstrategy.Low,
				StateMutation: executionstrategy.None, EvidenceQuality: executionstrategy.Complete,
				ValidationCost: executionstrategy.Low,
			},
			FixedStrategy: "low-risk", CompareBaseline: true,
		},
	}
	decision, comparison, err := selectExecutionStrategy(root, scenario)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Strategy != "high-risk-deliberated" || comparison == nil {
		t.Fatalf("decision=%#v comparison=%#v", decision, comparison)
	}
	if !comparison.Changed || !comparison.SafetyPreserved || comparison.ValidationTierDelta != 1 || !comparison.ParallelismChanged {
		t.Fatalf("comparison=%#v", comparison)
	}
}

func TestSelectExecutionStrategyRecordsConfiguredModelFallback(t *testing.T) {
	root := repositoryRootForEval(t)
	scenario := &Scenario{
		ID:    "strategy-fallback",
		Skill: "debug-code",
		ExecutionStrategy: &ExecutionStrategySpec{
			Input: executionstrategy.Input{
				Skill: "debug-code", Impact: executionstrategy.High, Reversibility: executionstrategy.Low,
				Ambiguity: executionstrategy.Low, SecuritySensitivity: executionstrategy.Low,
				StateMutation: executionstrategy.Repository, EvidenceQuality: executionstrategy.Complete,
				ValidationCost: executionstrategy.Medium, ConfiguredModelTiers: []string{"low"},
			},
			FixedStrategy: "low-risk", CompareBaseline: true,
		},
	}
	decision, _, err := selectExecutionStrategy(root, scenario)
	if err != nil {
		t.Fatal(err)
	}
	if decision.ModelTier != "low" || !containsStrategyValue(decision.Overrides.Values, "model-unavailable-fallback") {
		t.Fatalf("decision=%#v", decision)
	}
}

func containsStrategyValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func repositoryRootForEval(t *testing.T) string {
	t.Helper()
	return "../.."
}
