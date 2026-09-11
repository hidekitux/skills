package graph

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestValidateCurrentGraph(t *testing.T) {
	report := Validate(repositoryRoot(t))
	if !report.Valid {
		t.Fatalf("current graph is invalid: %s", strings.Join(report.Findings, "; "))
	}
	if report.SkillCount != 18 {
		t.Fatalf("expected 18 skills, got %d", report.SkillCount)
	}
}

func TestIssue175EntrypointsUseGraphRoutes(t *testing.T) {
	graph, err := Load(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"improve-project": "analyze-project",
		"deliver-change":  "plan-issue",
		"resolve-defect":  "debug-code",
	}
	for entrypoint, destination := range want {
		skill, ok := graph.Skill(entrypoint)
		if !ok {
			t.Fatalf("entrypoint %q is missing", entrypoint)
		}
		if len(skill.Transitions) == 0 || skill.Transitions[0].Destination.Skill != destination {
			t.Fatalf("entrypoint %q does not route to %q: %#v", entrypoint, destination, skill.Transitions)
		}
	}
}

func TestValidateRejectsDanglingSkillDestination(t *testing.T) {
	root := writeGraphFixture(t, Graph{
		SchemaVersion:    CurrentSchemaVersion,
		Artifacts:        []Definition{{ID: "result", Description: "A result."}},
		TerminalOutcomes: []string{"success"},
		Skills: []Skill{
			fixtureSkill("a", Transition{
				ID: "to-missing", Outcome: "success", Artifact: "result",
				Destination: Destination{Skill: "b"},
			}),
		},
	})

	report := Validate(root)
	if report.Valid || !containsFinding(report.Findings, `unknown skill destination "b"`) {
		t.Fatalf("expected dangling destination finding, got valid=%t findings=%v", report.Valid, report.Findings)
	}
}

func TestValidateRejectsUnboundedCycle(t *testing.T) {
	root := writeGraphFixture(t, Graph{
		SchemaVersion:    CurrentSchemaVersion,
		Artifacts:        []Definition{{ID: "result", Description: "A result."}},
		TerminalOutcomes: []string{"success"},
		Skills: []Skill{
			fixtureSkill("a", Transition{
				ID: "to-b", Outcome: "success", Artifact: "result",
				Destination: Destination{Skill: "b"},
			}),
			fixtureSkill("b", Transition{
				ID: "to-a", Outcome: "success", Artifact: "result",
				Destination: Destination{Skill: "a"},
			}),
		},
	})

	report := Validate(root)
	if report.Valid || !containsFinding(report.Findings, "cycle transition") {
		t.Fatalf("expected unbounded cycle finding, got valid=%t findings=%v", report.Valid, report.Findings)
	}
}

func TestValidateContextRejectsIncompleteCriticalInvariantCoverage(t *testing.T) {
	graph := &Graph{
		Skills: []Skill{{ID: "demo"}},
		Context: ContextConfig{
			SchemaVersion: CurrentContextSchemaVersion,
			Invariants: []Definition{
				{ID: "scope", Description: "Keep work in scope."},
				{ID: "privacy", Description: "Protect private content."},
			},
			Profiles: map[string]ContextProfile{
				"demo": {
					Budgets:            ContextBudget{CoreInstructions: 1, ConditionalRefs: 1, RepositoryEvidence: 1, ValidatorFeedback: 1},
					CriticalInvariants: []string{"scope"},
				},
			},
		},
	}
	findings := validateContext("", graph)
	if !containsFinding(findings, `profile "demo" omits critical invariant "privacy"`) {
		t.Fatalf("expected incomplete critical invariant finding, got %v", findings)
	}
}

func fixtureSkill(id string, transitions ...Transition) Skill {
	return Skill{
		ID: id, Path: filepath.ToSlash(filepath.Join("skills", id, "SKILL.md")),
		Layer: "process", Capabilities: []string{"test"}, Outputs: []string{"result"},
		Mutability: "read_only", Authority: Authority{
			Repository: "read", Git: "read", GitHub: "read", ExternalMutation: "none",
		}, Outcomes: []string{"success"}, Transitions: transitions,
	}
}

func writeGraphFixture(t *testing.T, graph Graph) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	content, err := yaml.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, root, Path, content)

	var catalog strings.Builder
	catalog.WriteString("catalog_version: 1\nlicense: Apache-2.0\nskills:\n")
	var inventory strings.Builder
	inventory.WriteString("# Fixture graph\n\n")
	inventory.WriteString("<!-- skills:graph-inventory:start -->\n")
	for _, skill := range graph.Skills {
		catalog.WriteString("  - name: " + skill.ID + "\n    layer: " + skill.Layer + "\n")
		inventory.WriteString("- `" + skill.ID + "`\n")
		skillPath := filepath.Join(root, filepath.FromSlash(skill.Path))
		if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, root, skill.Path, []byte("---\nname: "+skill.ID+"\ndescription: Fixture skill.\n---\n"))
	}
	inventory.WriteString("<!-- skills:graph-inventory:end -->\n")
	writeFixtureFile(t, root, "CATALOG.yml", []byte(catalog.String()))
	writeFixtureFile(t, root, "docs/skill-contract.md", []byte("# Contract\n\n"+Path+"\ncmd/read-skill-graph\ncmd/validate-skill-graph\n"))
	writeFixtureFile(t, root, "docs/skill-graph.md", []byte(inventory.String()))
	return root
}

func writeFixtureFile(t *testing.T, root, name string, content []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsFinding(findings []string, want string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return true
		}
	}
	return false
}
