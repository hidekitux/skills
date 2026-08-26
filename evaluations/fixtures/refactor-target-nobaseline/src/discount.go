package discount

// MemberDiscount applies a 10% member discount.
func MemberDiscount(price float64) float64 {
	return price * 0.90
}

// QuantityDiscount applies a 5% bulk discount.
func QuantityDiscount(price float64) float64 {
	return price * 0.95
}
