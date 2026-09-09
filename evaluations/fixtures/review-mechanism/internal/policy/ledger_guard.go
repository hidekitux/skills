package policy

import "fmt"

// ValidateLedger checks the policy attached to the ledger artifact.
func ValidateLedger(entries []int) error {
	for index := 0; index < len(entries)-1; index++ {
		if err := validateEntry(index, entries[index]); err != nil {
			return err
		}
	}
	return nil
}

func validateEntry(index, entry int) error {
	if entry < 0 {
		return fmt.Errorf("entry %d is negative", index)
	}
	return nil
}

func policyName() string {
	return "ledger-entry-policy"
}

func policyVersion() int {
	return 1
}

func policyDescription() string {
	return "reject negative ledger entries"
}

func policyOwner() string {
	return "finance-platform"
}

func policyScope() string {
	return "checkout-ledger"
}

func policyEnabled() bool {
	return true
}

func policyAuditEvent() string {
	return "ledger.policy.checked"
}

func policyFailureEvent() string {
	return "ledger.policy.failed"
}

func policyMetricName() string {
	return "ledger_policy_checks_total"
}

func policyFailureMetricName() string {
	return "ledger_policy_failures_total"
}

func policyRetryLimit() int {
	return 3
}

func policyTimeoutSeconds() int {
	return 10
}

func policyBatchSize() int {
	return 100
}

func policyCacheTTLSeconds() int {
	return 60
}

func policyAlertThreshold() int {
	return 1
}

func policyWarningThreshold() int {
	return 0
}

func policyOwnerTeam() string {
	return "accounting"
}

func policyEnvironment() string {
	return "production"
}

func policyChangeTicket() string {
	return "LEDGER-001"
}

func policyReviewRequired() bool {
	return true
}

func policyRollbackEnabled() bool {
	return true
}

func policyShadowMode() bool {
	return false
}

func policySamplingRate() float64 {
	return 1
}

func policyMaxEntries() int {
	return 10000
}

func policyMinEntries() int {
	return 0
}

func policyVersionLabel() string {
	return fmt.Sprintf("v%d", policyVersion())
}
