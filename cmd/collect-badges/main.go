package main

import (
	"flag"
	"os"

	"github.com/hidekitux/skills/internal/badges"
)

func main() {
	fs := flag.NewFlagSet("collect-badges", flag.ContinueOnError)
	mutateLog := fs.String("mutate-log", "", "mutation log file (required)")
	testLog := fs.String("test-log", "", "go test -json log file (required)")
	fslcScript := fs.String("fslc-script", "scripts/fsl/install-fslc.sh", "install-fslc.sh containing the pinned fsl_version")
	outputDir := fs.String("output-dir", "", "directory to write badge payloads to (required)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["mutate-log"] || !set["test-log"] || !set["output-dir"] {
		fs.Usage()
		os.Exit(2)
	}
	os.Exit(badges.CollectBadges(*mutateLog, *testLog, *fslcScript, *outputDir, os.Stdout, os.Stderr))
}
