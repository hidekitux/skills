package worker

// Run processes one billing job.
func Run(jobID string) string {
	return "processed:" + jobID
}
