package discount

import "testing"

func TestApply(t *testing.T) {
	if got := Apply(100, 10); got != 90 {
		t.Fatalf("Apply(100, 10) = %v, want 90", got)
	}
}
