package formatter

import "testing"

func TestLabel(t *testing.T) {
	if got := Label("Ada"); got != "label:Ada" {
		t.Fatalf("Label(Ada) = %q", got)
	}
}
