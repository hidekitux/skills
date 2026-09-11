package eval

import (
	"fmt"

	executionstrategy "github.com/hidekitux/skills/internal/strategy"
)

func selectExecutionStrategy(root string, sc *Scenario) (*executionstrategy.Decision, *StrategyComparison, error) {
	if sc.ExecutionStrategy == nil {
		return nil, nil, nil
	}
	input := sc.ExecutionStrategy.Input
	if input.Skill == "" {
		input.Skill = sc.Skill
	}
	decision, err := executionstrategy.Select(root, input)
	if err != nil {
		return nil, nil, err
	}
	if !sc.ExecutionStrategy.CompareBaseline {
		return &decision, nil, nil
	}
	policy, err := executionstrategy.Load(root)
	if err != nil {
		return nil, nil, err
	}
	baseline, ok := executionstrategy.Profile(policy, sc.ExecutionStrategy.FixedStrategy)
	if !ok {
		return nil, nil, fmt.Errorf("fixed strategy %q is not defined", sc.ExecutionStrategy.FixedStrategy)
	}
	comparison := &StrategyComparison{
		Status:                     "unavailable",
		UnavailableReason:          "host runner does not expose strategy-aware baseline execution",
		FixedStrategy:              baseline.ID,
		AdaptiveStrategy:           decision.Strategy,
		AdaptiveOutcome:            decision.Outcome,
		Changed:                    baseline.ID != decision.Strategy,
		PolicySafetyFloorPreserved: strategyTierRank(decision.ValidationTier) >= strategyTierRank(baseline.ValidationTier),
		ValidationTierDelta:        strategyTierRank(decision.ValidationTier) - strategyTierRank(baseline.ValidationTier),
		RetryBoundDelta:            decision.MaxRetries - baseline.MaxRetries,
		ElapsedBoundDelta:          decision.MaxElapsedMillis - baseline.MaxElapsedMillis,
		ParallelismChanged:         baseline.Parallelism != decision.Parallelism,
	}
	return &decision, comparison, nil
}

func strategyTierRank(value string) int {
	switch value {
	case "tier-1":
		return 1
	case "tier-2":
		return 2
	case "tier-3":
		return 3
	case "tier-4":
		return 4
	default:
		return 0
	}
}
