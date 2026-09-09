package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hidekitux/skills/internal/diagnostic"
	"github.com/hidekitux/skills/internal/fsl"
)

func main() {
	fs := flag.NewFlagSet("verify-fsl", flag.ContinueOnError)
	diagnosticFormat := fs.String("diagnostic-format", "text", "diagnostic format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *diagnosticFormat != "text" && *diagnosticFormat != "json" {
		fmt.Fprintln(os.Stderr, "error: -diagnostic-format must be text or json")
		os.Exit(2)
	}
	if *diagnosticFormat == "json" {
		code := fsl.VerifyFSL(".", io.Discard, io.Discard)
		if code != 0 {
			item, err := fsl.VerificationDiagnostic(code)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			if err := diagnostic.WriteJSON(os.Stdout, item); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		os.Exit(code)
	}
	os.Exit(fsl.VerifyFSL(".", os.Stdout, os.Stderr))
}
