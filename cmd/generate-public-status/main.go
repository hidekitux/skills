// Command generate-public-status renders the public status section of
// README.md from CATALOG.yml and docs/release-evidence.yml. By default it
// verifies that the committed section is current (the same logic the
// check-public-status repository check runs); --write regenerates the section
// in place. The regeneration is idempotent.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/publicstatus"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("generate-public-status", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	write := fs.Bool("write", false, "regenerate the public status section in README.md")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *write {
		os.Exit(publicstatus.Write(root, os.Stdout, os.Stderr))
	}
	os.Exit(publicstatus.Check(root, os.Stdout, os.Stderr))
}
