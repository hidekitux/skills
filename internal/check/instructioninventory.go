package check

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/hidekitux/skills/internal/instructions"
)

// CheckInstructionInventory verifies the measured-skill inventory and every
// conditional reference without downloading tokenizer data. Exact counts are
// checked by cmd/measure-instructions --check because that check may need the
// tokenizer's encoding asset.
func CheckInstructionInventory(root string, out, errOut io.Writer) int {
	path := filepath.Join(root, "docs", "skill-instruction-inventory.yml")
	inventory, err := instructions.Load(path)
	if err != nil {
		fmt.Fprintf(errOut, "load instruction inventory: %v\n", err)
		return 1
	}
	if err := instructions.ValidateCoverage(root, inventory); err != nil {
		fmt.Fprintf(errOut, "validate instruction inventory: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "instruction inventory valid: %d skills, encoding %s.\n", len(inventory.Skills), inventory.Tokenizer.Encoding)
	return 0
}
