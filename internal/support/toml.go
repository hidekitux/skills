package support

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// LoadTOMLFile decodes the TOML document at path into out. It wraps read and
// decode errors so callers can report a single combined message.
func LoadTOMLFile(path string, out any) error {
	text, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	if err := toml.Unmarshal(text, out); err != nil {
		return fmt.Errorf("%s cannot be parsed: %w", path, err)
	}
	return nil
}
