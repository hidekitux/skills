package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// CommandReviewer delegates rubric review to an external bounded command. The
// command receives a JSON payload on stdin and must return a JSON object of
// integer 1-5 scores keyed by the seven rubric dimensions. This lets a
// maintainer plug in any read-only reviewer without the harness assuming a
// host-specific subagent invocation.
type CommandReviewer struct {
	Command string
}

// Review implements RubricReviewer.
func (r *CommandReviewer) Review(ctx context.Context, sc *Scenario, transcript, sandboxDir string) (map[string]int, error) {
	payload, err := json.Marshal(map[string]any{
		"scenario":   sc.ID,
		"skill":      sc.Skill,
		"kind":       sc.Kind,
		"prompt_sha": promptSHA(sc),
		"sandbox":    sandboxDir,
		"transcript": transcript,
		"rubric":     sc.Rubric,
	})
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", r.Command)
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("reviewer command failed: %v", err)
	}
	var scores map[string]int
	if err := json.Unmarshal(output, &scores); err != nil {
		return nil, fmt.Errorf("reviewer output is not a JSON score object: %v", err)
	}
	for dimension, score := range scores {
		if score < 1 || score > 5 {
			return nil, fmt.Errorf("reviewer score for %s (%d) is outside 1-5", dimension, score)
		}
	}
	return scores, nil
}
