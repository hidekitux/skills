// Command check-promotion verifies retained behavioral evidence for stable
// catalog entries before a release is published.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("check-promotion", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(eval.CheckPromotion(root, os.Stdout, os.Stderr))
}
