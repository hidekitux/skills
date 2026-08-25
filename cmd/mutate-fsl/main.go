package main

import (
	"os"

	"github.com/hidekitux/skills/internal/fsl"
)

func main() {
	os.Exit(fsl.MutateFSL(".", os.Stdout, os.Stderr))
}
