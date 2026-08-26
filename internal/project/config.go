// Package project implements the repository-owned GitHub Project
// configuration and idempotent Project mutations that replace triage labels
// as the Issue workflow contract. Repository tooling and the workflow skills
// read .github/issue-project.toml and resolve mutable Project, field, and
// option IDs from declared names at runtime, so no repository-specific
// Project identity is embedded in publishable skill behavior.
package project

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// DefaultConfigPath is the repository-relative Project configuration file.
const DefaultConfigPath = ".github/issue-project.toml"

// RequiredFields are the single-select field roles the Issue workflow uses.
var RequiredFields = []string{"status", "priority", "scope"}

// FieldConfig declares one single-select Project field and its options.
type FieldConfig struct {
	Name    string   `toml:"name"`
	Options []string `toml:"options"`
}

// Config is the decoded form of .github/issue-project.toml.
type Config struct {
	Project struct {
		Owner           string `toml:"owner"`
		Title           string `toml:"title"`
		Number          int64  `toml:"number"`
		DefaultPriority string `toml:"default_priority"`
	} `toml:"project"`
	Fields map[string]FieldConfig `toml:"fields"`
}

// LoadConfig reads and validates the repository-owned Project configuration.
func LoadConfig(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("config path is empty")
	}
	var cfg Config
	if err := support.LoadTOMLFile(path, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ConfigPath returns the Project configuration path below a repository root.
func ConfigPath(root string) string {
	return filepath.Join(root, DefaultConfigPath)
}

// Field returns the declared single-select field for a required role.
func (c *Config) Field(role string) (FieldConfig, error) {
	field, ok := c.Fields[role]
	if !ok {
		return FieldConfig{}, fmt.Errorf("fields.%s is missing from the Project configuration", role)
	}
	if strings.TrimSpace(field.Name) == "" {
		return FieldConfig{}, fmt.Errorf("fields.%s.name is empty in the Project configuration", role)
	}
	return field, nil
}

// Validate checks that the configuration names a complete, unambiguous
// field-and-option contract before any live mutation.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Project.Owner) == "" {
		return errors.New("project.owner must name the Project owner")
	}
	if strings.TrimSpace(c.Project.Title) == "" {
		return errors.New("project.title must name the Project")
	}
	for _, role := range RequiredFields {
		field, err := c.Field(role)
		if err != nil {
			return err
		}
		if len(field.Options) == 0 {
			return fmt.Errorf("fields.%s.options must list at least one option", role)
		}
		seen := map[string]bool{}
		for _, option := range field.Options {
			if strings.TrimSpace(option) == "" {
				return fmt.Errorf("fields.%s.options must not contain empty values", role)
			}
			if seen[option] {
				return fmt.Errorf("fields.%s.options contains duplicate option %q", role, option)
			}
			seen[option] = true
		}
	}
	priority, _ := c.Field("priority")
	if !containsString(priority.Options, c.Project.DefaultPriority) {
		return fmt.Errorf("project.default_priority %q must be one of fields.priority.options %v",
			c.Project.DefaultPriority, priority.Options)
	}
	return nil
}

// HasOption reports whether the field role declares the option name.
func (c *Config) HasOption(role, option string) bool {
	field, err := c.Field(role)
	if err != nil {
		return false
	}
	return containsString(field.Options, option)
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
