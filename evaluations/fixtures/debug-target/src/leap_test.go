package leap

import "testing"

func TestIsLeapYearModern(t *testing.T) {
	if !IsLeapYear(2024) {
		t.Fatal("2024 should be a leap year")
	}
	if IsLeapYear(2023) {
		t.Fatal("2023 should not be a leap year")
	}
}

func TestIsLeapYearCentury196(t *testing.T) {
	if IsLeapYear(1900) {
		t.Fatal("1900 should not be a leap year: divisible by 100 but not by 400")
	}
}
