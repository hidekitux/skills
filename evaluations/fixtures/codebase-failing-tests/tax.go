package tax

// Total returns the amount after tax.
func Total(amount float64, rate float64) float64 {
	return amount + amount*rate
}
