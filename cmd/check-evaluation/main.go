// Command check-evaluation validates the behavioral evaluation corpus:
// scenario schema, expected-answer leakage, Acceptance criterion 1 coverage
// for every cataloged skill, and the promotion evidence gate for
// status: stable entries. It is wired into the check:repository dispatcher.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("check-evaluation", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(eval.CheckCorpus(root, os.Stdout, os.Stderr))
}
