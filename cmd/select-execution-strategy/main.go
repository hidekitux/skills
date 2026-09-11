// Command select-execution-strategy selects one bounded execution strategy
// from a privacy-safe JSON input document.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hidekitux/skills/internal/strategy"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("select-execution-strategy", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	inputFlag := fs.String("input", "", "JSON input file (default: stdin)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	input, err := readInput(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	decision, err := strategy.Select(root, input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := strategy.WriteJSON(os.Stdout, decision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if decision.Outcome != "selected" {
		os.Exit(1)
	}
}

func readInput(path string) (strategy.Input, error) {
	var reader io.Reader = os.Stdin
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return strategy.Input{}, fmt.Errorf("read input: %w", err)
		}
		defer file.Close()
		reader = file
	}
	var input strategy.Input
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return strategy.Input{}, fmt.Errorf("decode input: %w", err)
	}
	return input, nil
}
