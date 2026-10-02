package currency

// Total returns the line total before taxes.
func Total(price float64, quantity int) float64 {
	return price * float64(quantity)
}
