package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hidekitux/skills/internal/commitlint"
)

func main() {
	fs := flag.NewFlagSet("validate-commit-message", flag.ContinueOnError)
	message := fs.String("message", "", "commit message to validate")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["message"] {
		fmt.Fprintln(os.Stderr, "flag: -message is required")
		fs.Usage()
		os.Exit(2)
	}
	errors := commitlint.ValidateMessage(*message)
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "error: %s\n", strings.Join(errors, "; "))
		os.Exit(1)
	}
	fmt.Println("Commit message shape is valid.")
}
