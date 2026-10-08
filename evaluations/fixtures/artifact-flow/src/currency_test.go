package currency

import "testing"

func TestTotal(t *testing.T) {
	if got := Total(10, 2); got != 20 {
		t.Fatalf("Total(10, 2) = %v, want 20", got)
	}
}
