package tax

// Rate returns the applicable tax rate for a bracket index.
// Brackets: 0 = reduced, 1 = standard, 2 = luxury.
func Rate(bracket int) float64 {
	switch bracket {
	case 0:
		return 0.05
	case 1:
		return 0.10
	case 2:
		return 0.20
	default:
		return 0.10
	}
}
