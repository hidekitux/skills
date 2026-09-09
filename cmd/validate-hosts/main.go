package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/diagnostic"
	"github.com/hidekitux/skills/internal/support"
	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-hosts", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	diagnosticFormat := fs.String("diagnostic-format", "text", "diagnostic format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *diagnosticFormat != "text" && *diagnosticFormat != "json" {
		fmt.Fprintln(os.Stderr, "error: -diagnostic-format must be text or json")
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *diagnosticFormat == "json" {
		diagnostics, code, _, _ := validate.HostsDiagnostics(root)
		if code != 0 {
			if err := diagnostic.WriteJSONL(os.Stdout, diagnostics); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		os.Exit(code)
	}
	os.Exit(validate.CheckHosts(root, os.Stdout, os.Stderr))
}
