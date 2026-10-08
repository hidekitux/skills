// Command generate-skill-lists renders the generated skill list in README.md,
// docs/skill-layers.md, and skills/README.md from CATALOG.yml. By default it
// verifies that every committed list is current (the same logic the
// check-skill-lists repository check runs); --write regenerates the lists in
// place. The regeneration is idempotent.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/check"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("generate-skill-lists", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	write := fs.Bool("write", false, "regenerate the skill lists in place")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *write {
		os.Exit(check.WriteSkillLists(root, os.Stdout, os.Stderr))
	}
	os.Exit(check.CheckSkillLists(root, os.Stdout, os.Stderr))
}
