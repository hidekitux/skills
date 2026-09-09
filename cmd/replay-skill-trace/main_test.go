package main

import (
	"encoding/json"
	"testing"

	"github.com/hidekitux/skills/internal/replay"
)

func TestCommandReportIncludesFSLConformance(t *testing.T) {
	input := commandReport{
		Report:        replay.Report{Valid: true, Outcome: replay.OutcomeValid, Spec: "CrossSkillWorkflow", StepsChecked: 5},
		FSLConformant: true,
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"valid":true,"outcome":"valid","spec":"CrossSkillWorkflow","steps_checked":5,"fsl_conformant":true}`
	if string(data) != want {
		t.Fatalf("command report = %s, want %s", data, want)
	}
}
