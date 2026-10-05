package eval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// transcriptPath names the transcript of one scenario on one driver as
// <host>/<scenario>[.<role>].txt under dir. The deliberation role keeps
// concurrent baseline and candidate runs in separate files.
func transcriptPath(dir, scenario, host, role string) string {
	name := scenario
	if role != "" {
		name += "." + role
	}
	return filepath.Join(dir, host, name+".txt")
}

// writeTranscript writes text to transcriptPath. A transcript can
// hold repository content and model output, so it goes only to the local
// report directory. A write failure is reported on errOut and never changes
// the scenario verdict.
func writeTranscript(dir, scenario, host, role, text string, errOut io.Writer) {
	if dir == "" {
		return
	}
	path := transcriptPath(dir, scenario, host, role)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(errOut, "evaluate: cannot write transcript %s: %v\n", path, err)
		return
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		fmt.Fprintf(errOut, "evaluate: cannot write transcript %s: %v\n", path, err)
	}
}
