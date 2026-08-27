// Command scan-go-vuln runs the pinned govulncheck scanner over the module
// and applies the Issue #178 failure policy: reachable findings fail (exit 1),
// non-reachable findings are reported but do not fail (exit 0), and
// infrastructure errors fail (exit 2). The JSON Config header reports the
// scanner and vulnerability database versions; -out retains the raw JSON
// stream as machine-readable evidence.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/govuln"
)

func main() {
	fs := flag.NewFlagSet("scan-go-vuln", flag.ContinueOnError)
	dir := fs.String("dir", ".", "module directory to scan")
	out := fs.String("out", "", "retain the raw govulncheck JSON stream at this path")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: scan-go-vuln [flags] [go package patterns]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	os.Exit(govuln.Run(os.Stdout, os.Stderr, *dir, *out, fs.Args()))
}
