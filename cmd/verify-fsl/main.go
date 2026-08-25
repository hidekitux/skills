package main

import (
	"os"

	"github.com/hidekitux/skills/internal/fsl"
)

func main() {
	os.Exit(fsl.VerifyFSL(".", os.Stdout, os.Stderr))
}
