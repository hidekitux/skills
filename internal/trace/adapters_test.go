package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexAndClaudeAdaptersProduceEquivalentSemanticFields(t *testing.T) {
	root := filepath.Join("..", "..")
	codexData, err := os.ReadFile(filepath.Join(root, "hosts", "codex", "trace-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	claudeData, err := os.ReadFile(filepath.Join(root, "hosts", "claude-code", "trace-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	codex, err := AdaptCodex(codexData)
	if err != nil {
		t.Fatal(err)
	}
	claude, err := AdaptClaudeCode(claudeData)
	if err != nil {
		t.Fatal(err)
	}
	codex.Host, codex.HostVersion = "host", "version"
	claude.Host, claude.HostVersion = "host", "version"
	if codex.Model != claude.Model || codex.SkillID != claude.SkillID || codex.SkillVersion != claude.SkillVersion || codex.GraphVersion != claude.GraphVersion || codex.RepositoryRevision != claude.RepositoryRevision {
		t.Fatalf("identity differs: codex=%#v claude=%#v", codex, claude)
	}
	if codex.Deliberation == nil || claude.Deliberation == nil || codex.Deliberation.Pattern != claude.Deliberation.Pattern || len(codex.Deliberation.Candidates) != 2 || len(claude.Deliberation.Candidates) != 2 {
		t.Fatalf("deliberation summary was not normalized: codex=%#v claude=%#v", codex.Deliberation, claude.Deliberation)
	}
	codexJSON, _ := json.Marshal(codex.Events)
	claudeJSON, _ := json.Marshal(claude.Events)
	if string(codexJSON) != string(claudeJSON) {
		t.Fatalf("semantic events differ:\ncodex=%s\nclaude=%s", codexJSON, claudeJSON)
	}
	codexEnvironment, _ := json.Marshal(codex.Environment)
	claudeEnvironment, _ := json.Marshal(claude.Environment)
	if string(codexEnvironment) != string(claudeEnvironment) {
		t.Fatalf("environment manifests differ:\ncodex=%s\nclaude=%s", codexEnvironment, claudeEnvironment)
	}
	if codex.Environment == nil || codex.Environment.Profile != "read_only" {
		t.Fatalf("read-only environment was not normalized: %#v", codex.Environment)
	}
	validation := codex.Events[3].Validation
	if validation == nil || len(validation.Diagnostics) != 1 || validation.Diagnostics[0].Producer != "validate-issue-body" || validation.Diagnostics[0].Code != "validate-issue-body.invalid" {
		t.Fatalf("diagnostic reference was not normalized: %#v", validation)
	}
}

func TestAdaptersPreserveExecutionStrategy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "hosts", "codex", "trace-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["strategy"] = validStrategyDecision()
	data, err = json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	item, err := AdaptCodex(data)
	if err != nil {
		t.Fatal(err)
	}
	if item.Strategy == nil || item.Strategy.Strategy != "high-risk-deliberated" {
		t.Fatalf("strategy was not preserved: %#v", item.Strategy)
	}
}

func TestAdaptersRejectUnknownEvents(t *testing.T) {
	data := []byte(`{"run_id":"run-1","skill":{"id":"plan-issue","version":"0.1.0"},"graph_version":1,"host":{"name":"codex","version":"1"},"model":"gpt-5","repository_revision":"0123456789abcdef0123456789abcdef01234567","started_at":"2026-09-09T12:00:00Z","events":[{"sequence":1,"at":"2026-09-09T12:00:00Z","type":"unknown"}]}`)
	if _, err := AdaptCodex(data); err == nil {
		t.Fatal("unknown event was accepted")
	}
}

func TestAdaptersRejectRawHostFields(t *testing.T) {
	data := []byte(`{"run_id":"run-1","skill":{"id":"plan-issue","version":"0.1.0"},"graph_version":1,"host":{"name":"codex","version":"1"},"model":"gpt-5","repository_revision":"0123456789abcdef0123456789abcdef01234567","started_at":"2026-09-09T12:00:00Z","events":[],"raw_output":"do not persist"}`)
	if _, err := AdaptCodex(data); err == nil {
		t.Fatal("raw host field was accepted")
	}
}

func TestAdaptersRejectTrailingJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "hosts", "codex", "trace-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte(" trailing-json")...)
	if _, err := AdaptCodex(data); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

func TestAdaptersPreserveFailedHandoffStatus(t *testing.T) {
	for _, name := range []string{"codex", "claude-code"} {
		path := filepath.Join("..", "..", "hosts", name, "trace-fixture.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte(`"outcome": "success"`), []byte(`"outcome": "failed"`), 1)
		var item Trace
		if name == "codex" {
			item, err = AdaptCodex(data)
		} else {
			item, err = AdaptClaudeCode(data)
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		found := false
		for _, event := range item.Events {
			if event.Kind == KindHandoff {
				found = true
				if event.Status != StatusFailed {
					t.Fatalf("%s handoff status = %s, want failed", name, event.Status)
				}
			}
		}
		if !found {
			t.Fatalf("%s fixture has no handoff", name)
		}
	}
}
