package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfigTOML = `[project]
owner = "acme"
title = "Skills Issues"
default_priority = "Medium"

[fields.status]
name = "Status"
options = ["Backlog", "Planned", "In progress", "In review", "Done"]

[fields.priority]
name = "Priority"
options = ["High", "Medium", "Low"]

[fields.scope]
name = "Scope"
options = ["Feature", "Bug", "Docs", "Maintenance", "Improvement", "Security", "Release"]
`

func loadTestConfig(t *testing.T, text string) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "issue-project.toml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

func TestLoadConfigAcceptsRepositoryContract(t *testing.T) {
	cfg := loadTestConfig(t, validConfigTOML)
	if cfg.Project.Owner != "acme" || cfg.Project.Title != "Skills Issues" {
		t.Fatalf("unexpected project identity: %+v", cfg.Project)
	}
	for _, role := range RequiredFields {
		if _, err := cfg.Field(role); err != nil {
			t.Fatalf("Field(%q): %v", role, err)
		}
	}
	if !cfg.HasOption("status", "Done") || !cfg.HasOption("scope", "Security") {
		t.Fatal("declared options missing")
	}
}

func TestLoadConfigRejectsIncompleteContract(t *testing.T) {
	cases := map[string]string{
		"missing owner":        strings.Replace(validConfigTOML, `owner = "acme"`, "", 1),
		"missing title":        strings.Replace(validConfigTOML, `title = "Skills Issues"`, "", 1),
		"missing status field": strings.Replace(validConfigTOML, "[fields.status]", "[fields.status-typo]", 1),
		"empty options":        strings.Replace(validConfigTOML, `options = ["High", "Medium", "Low"]`, "options = []", 1),
		"duplicate option":     strings.Replace(validConfigTOML, `"In progress"`, `"In progress", "In progress"`, 1),
		"default priority outside options": strings.Replace(validConfigTOML,
			`default_priority = "Medium"`, `default_priority = "Urgent"`, 1),
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "issue-project.toml")
			if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			if _, err := LoadConfig(path); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}

func TestLoadConfigRejectsMissingFile(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "absent.toml")); err == nil {
		t.Fatal("expected missing file to fail")
	}
	if _, err := LoadConfig(""); err == nil {
		t.Fatal("expected empty path to fail")
	}
}

func TestLoadRepositoryConfig(t *testing.T) {
	// The committed repository contract must stay loadable and valid so the
	// workflow skills and tooling resolve the same declared names.
	cfg, err := LoadConfig(filepath.Join("..", "..", ".github", "issue-project.toml"))
	if err != nil {
		t.Fatalf("repository issue-project.toml cannot be loaded: %v", err)
	}
	priority, _ := cfg.Field("priority")
	for _, option := range []string{"High", "Medium", "Low"} {
		if !containsString(priority.Options, option) {
			t.Fatalf("priority option %q missing", option)
		}
	}
}
