package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/release"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("verify-release", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: verify-release [-root ROOT] vX.Y.Z")
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(release.VerifyRelease(fs.Arg(0), root, os.Stdout, os.Stderr))
}
