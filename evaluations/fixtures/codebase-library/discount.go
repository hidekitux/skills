package discount

// Apply returns the price after applying a percentage discount.
func Apply(price float64, percent int) float64 {
	return price * (100 - float64(percent)) / 100
}
