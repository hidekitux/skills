package eval

import "github.com/hidekitux/skills/internal/trace"

// TraceMetrics is a machine-readable run summary derived only from traces.
type TraceMetrics struct {
	TraceCount          int   `json:"trace_count"`
	SuccessCount        int   `json:"success_count"`
	FailureCount        int   `json:"failure_count"`
	SkippedCount        int   `json:"skipped_count"`
	InfrastructureCount int   `json:"infrastructure_error_count"`
	InterruptedCount    int   `json:"interrupted_count"`
	RetryCount          int   `json:"retry_count"`
	HandoffCount        int   `json:"handoff_count"`
	ElapsedMillis       int64 `json:"elapsed_millis"`
	ContextTokens       int64 `json:"context_tokens"`
	CostMicros          int64 `json:"cost_micros"`
	UsageRecords        int   `json:"usage_records"`
}

// CalculateTraceMetrics derives run metrics without reading prose or transcripts.
func CalculateTraceMetrics(traces []trace.Trace) TraceMetrics {
	metrics := TraceMetrics{TraceCount: len(traces)}
	for _, item := range traces {
		switch item.Terminal.Status {
		case "success":
			metrics.SuccessCount++
		case "failed":
			metrics.FailureCount++
		case "skipped":
			metrics.SkippedCount++
		case "interrupted":
			metrics.InterruptedCount++
		case "infrastructure_error":
			metrics.InfrastructureCount++
		}
		if item.Usage != nil {
			metrics.ElapsedMillis += item.Usage.ElapsedMillis
			if item.Usage.Available {
				metrics.UsageRecords++
			}
			if item.Usage.ContextTokens != nil {
				metrics.ContextTokens += *item.Usage.ContextTokens
			}
			if item.Usage.CostMicros != nil {
				metrics.CostMicros += *item.Usage.CostMicros
			}
		}
		for _, event := range item.Events {
			switch event.Kind {
			case trace.KindRetry:
				metrics.RetryCount++
			case trace.KindHandoff:
				metrics.HandoffCount++
			}
		}
	}
	return metrics
}
