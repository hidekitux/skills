package eval

import (
	"fmt"

	"github.com/hidekitux/skills/internal/trace"
)

// traceForRecord reports one structured evaluation result as the typed domain
// result the evidence module converts. This package decides what the run was;
// internal/trace decides how the run is persisted.
func traceForRecord(sc *Scenario, record Record, graphVersion int, skillVersion string) (trace.Trace, error) {
	result := trace.RunResult{
		RunID:              record.RunID,
		ScenarioID:         record.Scenario,
		SkillID:            traceSkill(sc, record),
		SkillVersion:       skillVersion,
		GraphVersion:       graphVersion,
		Host:               record.Host,
		Model:              record.Model,
		RepositoryRevision: record.Commit,
		StartedAt:          record.StartedAt,
		FinishedAt:         record.FinishedAt,
		ElapsedMillis:      record.ElapsedMillis,
		Context:            record.Context,
		Deliberation:       record.Deliberation,
		Strategy:           record.Strategy,
		Outcome:            trace.RunOutcome(record.Verdict),
		CorrectionAttempts: record.CorrectionsUsed,
	}
	if record.HandoffObserved && sc.Expectations.Handoff != "" {
		result.HandoffDestination = sc.Expectations.Handoff
	}
	return trace.FromEvaluationRun(result), nil
}

func traceSkill(sc *Scenario, record Record) string {
	if record.Skill != E2ESkill || len(sc.Stages) == 0 {
		return record.Skill
	}
	return sc.Stages[0].Skill
}

func traceSkillVersion(root string, sc *Scenario, record Record) (string, error) {
	version, err := trace.SkillVersion(root, traceSkill(sc, record))
	if err != nil {
		return "", fmt.Errorf("skill version for %q: %w", traceSkill(sc, record), err)
	}
	return version, nil
}
