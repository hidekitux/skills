package tax

import "testing"

func TestTotalZeroRate(t *testing.T) {
	if got := Total(100, 0); got != 90 {
		t.Fatalf("Total(100, 0) = %v, want 90", got)
	}
}
