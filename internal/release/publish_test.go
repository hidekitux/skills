package release

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPublishReleaseUsesCanonicalMiseTasks(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test source")
	}
	content, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "publish.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, task := range []string{"validate:all", "validate:skill-creator", "verify:release"} {
		if !strings.Contains(source, `"`+task+`"`) {
			t.Errorf("publish.go must invoke canonical task %q", task)
		}
	}
	for _, retired := range []string{"validate", "validate-skill-creator", "verify-release"} {
		if strings.Contains(source, `"`+retired+`"`) {
			t.Errorf("publish.go must not invoke retired task %q", retired)
		}
	}
}
