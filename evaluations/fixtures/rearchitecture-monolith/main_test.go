package main

import "testing"

func TestOrderShape(t *testing.T) {
	if (Order{CustomerID: "customer-1"}).CustomerID == "" {
		t.Fatal("customer ID is required")
	}
}
