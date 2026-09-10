// Command compile-context emits a bounded, task-aware context package.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	skillcontext "github.com/hidekitux/skills/internal/context"
	"github.com/hidekitux/skills/internal/instructions"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("compile-context", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	skillFlag := fs.String("skill", "", "skill ID to compile")
	signalsFlag := fs.String("signals", "", "JSON file containing observable task signals")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *skillFlag == "" {
		fmt.Fprintln(os.Stderr, "compile-context: --skill is required")
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	signals := skillcontext.Signals{}
	if *signalsFlag != "" {
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(*signalsFlag)))
		if filepath.IsAbs(*signalsFlag) {
			data, readErr = os.ReadFile(*signalsFlag)
		}
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "compile-context: read signals: %v\n", readErr)
			os.Exit(1)
		}
		signals, err = skillcontext.DecodeSignals(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "compile-context: decode signals: %v\n", err)
			os.Exit(1)
		}
	}
	counter, err := instructions.NewCounter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile-context: %v\n", err)
		os.Exit(2)
	}
	compiled, err := (skillcontext.Compiler{Root: root, Counter: counter}).Compile(*skillFlag, signals)
	if err != nil && compiled.Manifest.Overflow == nil {
		fmt.Fprintf(os.Stderr, "compile-context: %v\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(compiled); encodeErr != nil {
		fmt.Fprintf(os.Stderr, "compile-context: encode output: %v\n", encodeErr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile-context: %v\n", err)
		if errors.Is(err, skillcontext.ErrRequiredOverflow) {
			os.Exit(4)
		}
		os.Exit(1)
	}
}
