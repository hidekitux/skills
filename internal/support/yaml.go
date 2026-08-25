package support

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAMLFile decodes the YAML document at path into out. It wraps read and
// decode errors so callers can report a single combined message.
func LoadYAMLFile(path string, out any) error {
	text, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(text, out); err != nil {
		return fmt.Errorf("%s cannot be parsed: %w", path, err)
	}
	return nil
}
