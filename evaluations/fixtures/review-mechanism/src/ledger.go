package ledger

// Total returns the sum of ledger entries.
func Total(entries []int) int {
	total := 0
	for _, entry := range entries {
		total += entry
	}
	return total
}
