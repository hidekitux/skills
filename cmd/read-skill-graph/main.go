// Command read-skill-graph returns the versioned JSON graph read view.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("read-skill-graph", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	skillID := fs.String("skill", "", "return one named skill instead of the full graph")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	loaded, err := graph.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *skillID == "" {
		if err := graph.WriteJSON(os.Stdout, loaded); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	skill, ok := loaded.Skill(*skillID)
	if !ok {
		fmt.Fprintf(os.Stderr, "skill %q is not in the graph\n", *skillID)
		os.Exit(1)
	}
	view := struct {
		SchemaVersion int          `json:"schema_version"`
		Skill         *graph.Skill `json:"skill"`
	}{SchemaVersion: loaded.SchemaVersion, Skill: skill}
	if err := graph.WriteJSON(os.Stdout, view); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
