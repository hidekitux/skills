// Command read-skill-trace returns validated structured traces as JSON.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/trace"
)

func main() {
	fs := flag.NewFlagSet("read-skill-trace", flag.ContinueOnError)
	inputFlag := fs.String("input", "", "JSONL trace file")
	runIDFlag := fs.String("run-id", "", "return only one run identifier")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *inputFlag == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(2)
	}
	traces, err := trace.ReadJSONL(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *runIDFlag != "" {
		filtered := make([]trace.Trace, 0, 1)
		for _, item := range traces {
			if item.RunID == *runIDFlag {
				filtered = append(filtered, item)
			}
		}
		traces = filtered
	}
	if len(traces) == 0 {
		fmt.Fprintln(os.Stderr, "no matching trace records")
		os.Exit(1)
	}
	var value any = traces
	if len(traces) == 1 {
		value = traces[0]
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
