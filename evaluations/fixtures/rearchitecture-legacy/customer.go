package customer

// LegacyRecord is the stored shape used by existing clients.
type LegacyRecord struct {
	Name string
}

// Customer is the normalized shape proposed for new code.
type Customer struct {
	DisplayName string
}
