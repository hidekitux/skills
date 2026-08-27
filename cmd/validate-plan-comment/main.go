// Command validate-plan-comment validates the machine-readable plan comment
// that advances an Issue's Project Status to Planned.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/hidekitux/skills/internal/project"
)

func main() {
	fs := flag.NewFlagSet("validate-plan-comment", flag.ExitOnError)
	issue := fs.Int64("issue", 0, "Issue number")
	body := fs.String("body", "", "plan comment body")
	bodyEnv := fs.String("body-env", "", "environment variable containing the plan comment body")
	fs.Parse(os.Args[1:])
	if *issue <= 0 {
		fmt.Fprintln(os.Stderr, "error: --issue must be a positive Issue number")
		os.Exit(2)
	}
	comment := *body
	if *bodyEnv != "" {
		comment = os.Getenv(*bodyEnv)
	}
	if !project.IsAuthoritativePlanComment(comment, *issue) {
		fmt.Fprintf(os.Stderr, "error: comment is not an authoritative plan comment for Issue #%s\n", strconv.FormatInt(*issue, 10))
		os.Exit(1)
	}
	fmt.Printf("authoritative plan comment verified for Issue #%d\n", *issue)
}
