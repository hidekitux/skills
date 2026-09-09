// Command measure-instructions checks the published-skill instruction
// inventory and prints fixed-encoding token counts for every skill.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hidekitux/skills/internal/instructions"
	"github.com/hidekitux/skills/internal/support"
)

type output struct {
	Encoding string         `json:"encoding"`
	Module   string         `json:"module"`
	Version  string         `json:"version"`
	Counts   map[string]int `json:"counts"`
}

func main() {
	fs := flag.NewFlagSet("measure-instructions", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	inventoryFlag := fs.String("inventory", "docs/skill-instruction-inventory.yml", "instruction inventory path")
	checkFlag := fs.Bool("check", false, "require the measured counts to match after_tokens")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	inventoryPath := *inventoryFlag
	if !filepath.IsAbs(inventoryPath) {
		inventoryPath = filepath.Join(root, inventoryPath)
	}
	inventory, err := instructions.Load(inventoryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "measure-instructions: %v\n", err)
		os.Exit(1)
	}
	counter, err := instructions.NewCounter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "measure-instructions: %v\n", err)
		os.Exit(2)
	}
	counts, err := instructions.Measure(root, inventory, counter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "measure-instructions: %v\n", err)
		os.Exit(1)
	}
	if *checkFlag {
		for _, skill := range inventory.Skills {
			if counts[skill.Name] != skill.AfterTokens {
				fmt.Fprintf(os.Stderr, "measure-instructions: %s measured %d tokens, inventory has %d\n", skill.Name, counts[skill.Name], skill.AfterTokens)
				os.Exit(1)
			}
		}
	}
	encoded, err := json.MarshalIndent(output{
		Encoding: instructions.EncodingName,
		Module:   instructions.TokenizerModule,
		Version:  instructions.TokenizerVersion,
		Counts:   counts,
	}, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "measure-instructions: encode output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}
