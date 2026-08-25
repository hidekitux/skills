package main

import (
	"os"

	"github.com/hidekitux/skills/internal/release"
)

func main() {
	os.Exit(release.PublishRelease(os.Args[1:], os.Stdout, os.Stderr))
}
