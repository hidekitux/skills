package discount

import "testing"

func TestDiscount(t *testing.T) {
	if got := Discount(100, 10); got != 90 {
		t.Fatalf("Discount(100, 10) = %v, want 90", got)
	}
}
