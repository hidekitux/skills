package discount

// Discount returns the discounted price for a positive discount percent.
func Discount(price float64, percent int) float64 {
	return price * (100 - float64(percent)) / 100
}
