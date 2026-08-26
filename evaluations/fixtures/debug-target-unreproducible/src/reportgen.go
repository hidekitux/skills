package report

// Render produces a formatted report line for a record.
func Render(record map[string]string) string {
	return record["name"] + ": " + record["value"]
}
