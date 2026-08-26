package discount

import "testing"

func TestMemberDiscount(t *testing.T) {
	if got := MemberDiscount(100); got != 90 {
		t.Fatalf("MemberDiscount(100) = %v, want 90", got)
	}
}

func TestQuantityDiscount(t *testing.T) {
	if got := QuantityDiscount(100); got != 95 {
		t.Fatalf("QuantityDiscount(100) = %v, want 95", got)
	}
}
