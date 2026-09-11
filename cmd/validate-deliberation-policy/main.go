// Command validate-deliberation-policy validates the repository's bounded
// multi-agent deliberation policy.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/deliberation"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("validate-deliberation-policy", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(deliberation.Check(root, os.Stdout, os.Stderr))
}
