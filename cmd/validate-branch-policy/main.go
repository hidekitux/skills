package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/diagnostic"
	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-branch-policy", flag.ContinueOnError)
	config := fs.String("config", ".github/branch-policy.toml", "branch policy configuration path")
	base := fs.String("base", os.Getenv("PR_BASE_REF"), "pull request base branch")
	head := fs.String("head", os.Getenv("PR_HEAD_REF"), "pull request head branch")
	body := fs.String("body", os.Getenv("PR_BODY"), "pull request body")
	validateConfig := fs.Bool("validate-config", false, "only validate the configuration file")
	diagnosticFormat := fs.String("diagnostic-format", "text", "diagnostic format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *diagnosticFormat != "text" && *diagnosticFormat != "json" {
		fmt.Fprintln(os.Stderr, "error: -diagnostic-format must be text or json")
		os.Exit(2)
	}
	if *diagnosticFormat == "json" {
		diagnostics, code, _, _ := validate.BranchPolicyDiagnostics(*config, *base, *head, *body, *validateConfig)
		if code != 0 {
			if err := diagnostic.WriteJSONL(os.Stdout, diagnostics); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		os.Exit(code)
	}
	os.Exit(validate.CheckBranchPolicy(*config, *base, *head, *body, *validateConfig, os.Stdout, os.Stderr))
}
