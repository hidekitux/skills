package leap

// IsLeapYear reports whether year is a leap year.
func IsLeapYear(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}
