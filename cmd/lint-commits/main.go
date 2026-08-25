package main

import (
	"os"

	"github.com/hidekitux/skills/internal/commitlint"
)

func main() {
	os.Exit(commitlint.LintCommits(commitlint.ExecRunner(), os.Stdout, os.Stderr))
}
