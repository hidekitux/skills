package check

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const triageRegisterPath = "docs/mutation-triage.md"

// triageBlockRE matches the JSON register block in docs/mutation-triage.md.
// The markdown code fence is expressed as \x60 (backtick) because Go raw
// string literals cannot contain a backtick.
var triageBlockRE = regexp.MustCompile(`(?s)<!-- mutation-triage:start -->\s*\x60\x60\x60json\s*(.+?)\s*\x60\x60\x60\s*<!-- mutation-triage:end -->`)

type triageSurvivor struct {
	Op          string `json:"op"`
	Target      string `json:"target"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
}

type triageSpec struct {
	Spec      string           `json:"spec"`
	Survivors []triageSurvivor `json:"survivors"`
}

type triageRegister struct {
	Specs []triageSpec `json:"specs"`
}

const (
	triageAccepted    = "accepted"
	triageFixPlanned  = "fix-planned"
	triageNeedsReview = "needs-review"
)

func validDisposition(d string) bool {
	return d == triageAccepted || d == triageFixPlanned || d == triageNeedsReview
}

// CheckMutationTriage requires the survivor triage register to exist and to
// cover every surviving mutant with an explicit disposition and reason, so a
// badge count alone is never sufficient triage evidence. needs-review entries
// (untriaged survivors) fail the check.
func CheckMutationTriage(root string, out, errOut io.Writer) int {
	path := filepath.Join(root, filepath.FromSlash(triageRegisterPath))
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(errOut, "Triage check failed: cannot read %s: %v\n", triageRegisterPath, err)
		return 1
	}
	match := triageBlockRE.FindSubmatch(content)
	if match == nil {
		fmt.Fprintf(errOut, "Triage check failed: %s has no <!-- mutation-triage:start --> JSON block\n", triageRegisterPath)
		return 1
	}
	var register triageRegister
	if err := json.Unmarshal(match[1], &register); err != nil {
		fmt.Fprintf(errOut, "Triage check failed: register JSON is invalid: %v\n", err)
		return 1
	}
	if len(register.Specs) == 0 {
		fmt.Fprintln(errOut, "Triage check failed: register has no spec entries")
		return 1
	}

	total := 0
	problems := []string{}
	for _, spec := range register.Specs {
		if spec.Spec == "" {
			problems = append(problems, "a spec entry has no spec path")
		}
		for _, s := range spec.Survivors {
			total++
			switch {
			case s.Disposition == "":
				problems = append(problems, fmt.Sprintf("%s: survivor %q has no disposition", spec.Spec, s.Target))
			case !validDisposition(s.Disposition):
				problems = append(problems, fmt.Sprintf("%s: survivor %q has unknown disposition %q", spec.Spec, s.Target, s.Disposition))
			case s.Disposition == triageNeedsReview:
				problems = append(problems, fmt.Sprintf("%s: survivor %q is untriaged (needs-review)", spec.Spec, s.Target))
			}
			if s.Reason == "" {
				problems = append(problems, fmt.Sprintf("%s: survivor %q has no reason", spec.Spec, s.Target))
			}
		}
	}
	if len(problems) > 0 {
		fmt.Fprintln(errOut, "Triage check failed:")
		for _, p := range problems {
			fmt.Fprintf(errOut, "- %s\n", p)
		}
		return 1
	}
	fmt.Fprintf(out, "Triage register covers %d survivor(s) with explicit dispositions.\n", total)
	return 0
}
