// Command check-repository runs the six independent repository checks
// concurrently and reports each result under a labeled section, so a failing
// check stays identifiable while the aggregate task still fails. It replaces
// the former sequential series of six separate "go run" commands in the
// check:repository mise task.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/hidekitux/skills/internal/check"
	"github.com/hidekitux/skills/internal/support"
	"github.com/hidekitux/skills/internal/validate"
)

// repoCheck pairs a repository check with the name used in its output label.
type repoCheck struct {
	name string
	fn   func(root string, out, errOut io.Writer) int
}

// repoChecks is the set of independent checks the check:repository mise task
// used to run one after another. Each check only reads repository files, so
// they can run concurrently without interfering.
var repoChecks = []repoCheck{
	{name: "validate-repository", fn: validate.CheckRepository},
	{name: "check-tool-licenses", fn: check.CheckToolLicenses},
	{name: "validate-script-tests", fn: validate.CheckScriptTests},
	{name: "check-sensitive-content", fn: check.CheckSensitiveContent},
	{name: "check-mutation-badges", fn: check.CheckMutationBadges},
	{name: "check-analyze-readonly", fn: check.CheckAnalyzeReadonly},
}

type checkResult struct {
	name string
	code int
	out  bytes.Buffer
	err  bytes.Buffer
}

// run executes every check concurrently, buffers its output, and prints the
// results in a stable order with one labeled section per check. It returns 1
// when any check fails so the aggregate task fails.
func run(root string, out, errOut io.Writer, checks []repoCheck) int {
	results := make([]*checkResult, len(checks))
	var wg sync.WaitGroup
	for i, rc := range checks {
		i, rc := i, rc
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := &checkResult{name: rc.name}
			res.code = rc.fn(root, &res.out, &res.err)
			results[i] = res
		}()
	}
	wg.Wait()

	failed := []string{}
	for _, res := range results {
		writeLabeled(out, res.name, res.out.String())
		writeLabeled(errOut, res.name, res.err.String())
		if res.code != 0 {
			failed = append(failed, res.name)
			fmt.Fprintf(errOut, "FAIL: %s\n", res.name)
		}
	}
	if len(failed) == 0 {
		fmt.Fprintf(out, "check:repository: all %d repository checks passed.\n", len(checks))
		return 0
	}
	fmt.Fprintf(errOut, "check:repository: FAILED (%d of %d repository checks failed: %s).\n",
		len(failed), len(checks), strings.Join(failed, ", "))
	return 1
}

// writeLabeled prefixes every line with the check name so output from
// concurrently running checks stays attributable.
func writeLabeled(w io.Writer, name, text string) {
	if text == "" {
		return
	}
	label := "[" + name + "]"
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fmt.Fprintf(w, "%s %s\n", label, scanner.Text())
	}
}

func main() {
	fs := flag.NewFlagSet("check-repository", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(run(root, os.Stdout, os.Stderr, repoChecks))
}
