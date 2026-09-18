package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDecisions is the contract record the cutover fixture must cover. The
// matrix fails when a recorded contract appears in no participant.
const fixtureDecisions = `schema_version: 1
contracts:
  - id: mise-task-names
    surface: The task names declared in mise.toml.
    classification: preserved
    owner: composition
    evidence:
      - mise.toml
    reason: No task was renamed, added, or removed.
    check: check:tasks validates the names and references.
`

// fixtureCutover is a complete record. A test that exercises one finding edits
// a single field of it.
const fixtureCutover = `schema_version: 1
baseline: 4cce0641bbc9bc28c9bba47522a6071b0acded69
cutover_point: 9f04db42a68e5140e4dc4067a42fdc8b02b230f0
order:
  - issue: 328
    delivers: Module ownership and the boundary check.
    depends_on: []
  - issue: 331
    delivers: The contract decisions and the check that enforces them.
    depends_on:
      - 328
participants:
  - id: module-foundation
    kind: module
    surface: Shared primitives every other module imports.
    owner: foundation
    contracts: []
    evidence:
      - mise.toml
    readiness: The foundation module imports no other internal module.
    readiness_check: go run ./cmd/check-repository
    recovery: The foundation packages keep their paths, so no caller changes.
  - id: module-policy
    kind: module
    surface: Policy decisions about execution strategy.
    owner: policy
    contracts: []
    evidence:
      - mise.toml
    readiness: The committed execution-strategy policy validates.
    readiness_check: go run ./cmd/validate-execution-strategy
    recovery: The policy files are unchanged, so reverting restores only the call sites.
  - id: mise-tasks
    kind: workflow
    surface: The task names mise.toml defines.
    owner: composition
    contracts:
      - mise-task-names
    evidence:
      - mise.toml
    readiness: Every task name the documentation calls exists.
    readiness_check: go run ./cmd/validate-mise-tasks
    recovery: mise.toml is unchanged, so reverting leaves every task name.
recovery:
  safe_state: The measured baseline, reached by reverting the redesign range.
  method: revert
  triggers:
    - id: readiness-unmet
      condition: A readiness condition fails at the cutover point.
      action: Stop and run the revert steps.
  steps:
    - id: revert-range
      action: Revert the redesign commit range.
      observation: Every revert applies without a conflict.
  rehearsal:
    - command: go run ./cmd/check-repository
      observation: The reverted tree reports the baseline total and exits zero.
post_cutover:
  - id: public-command-behavior
    aspect: public-behavior
    command: go run ./cmd/check-repository
    observation: Every named check runs and the total matches.
  - id: evidence-privacy
    aspect: privacy
    command: go run ./cmd/check-sensitive-content
    observation: No credential reaches a committed artifact.
  - id: mutation-authority
    aspect: authority
    command: go run ./cmd/check-analyze-readonly
    observation: No read-only skill mutates the repository.
  - id: provider-failure-classification
    aspect: provider-failure
    command: go test ./internal/provider/
    observation: Each provider failure keeps its own classification.
  - id: terminal-outcomes
    aspect: terminal-outcome
    command: go test ./internal/trace/
    observation: The terminal outcomes stay separate.
  - id: deterministic-replay
    aspect: deterministic-replay
    command: go test ./internal/replay/
    observation: A second replay returns the same outcome.
  - id: aggregate-validation
    aspect: compatibility
    command: mise run validate:all
    observation: The aggregate validation exits zero.
`

// fixtureRecord is an architecture record that names every identifier the
// cutover fixture uses.
const fixtureRecord = `# Architecture

The cutover point is 9f04db42a68e5140e4dc4067a42fdc8b02b230f0 and the baseline
is 4cce0641bbc9bc28c9bba47522a6071b0acded69.

| module-foundation | module | foundation |
| module-policy | module | policy |
| mise-tasks | workflow | composition |
| mise-task-names | preserved |
`

// writeCutoverTree builds a repository root holding the cutover record, the
// module model, the contract decisions, the architecture record, and the
// evidence path the fixture cites.
func writeCutoverTree(t *testing.T, cutover, record string) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"workflow", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join("workflow", "cutover-record.yml"):     cutover,
		filepath.Join("workflow", "module-ownership.yml"):   fixtureOwnership,
		filepath.Join("workflow", "contract-decisions.yml"): fixtureDecisions,
		filepath.Join("docs", "architecture.md"):            record,
		"mise.toml":                                         "# fixture\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// runCutover runs the check against a written tree and returns its status and
// the writer the status selects.
func runCutover(t *testing.T, cutover, record string) (int, string) {
	t.Helper()
	root := writeCutoverTree(t, cutover, record)
	var out, errOut bytes.Buffer
	status := CheckCutoverRecord(root, &out, &errOut)
	if status == 0 {
		return status, out.String()
	}
	return status, errOut.String()
}

func TestCutoverRecordAcceptsACompleteRecord(t *testing.T) {
	status, output := runCutover(t, fixtureCutover, fixtureRecord)
	if status != 0 {
		t.Fatalf("status = %d, want 0: %s", status, output)
	}
	if !strings.Contains(output, "3 participant(s), 1 recovery step(s), 7 post-cutover check(s)") {
		t.Fatalf("output does not report the counted rows: %s", output)
	}
}

func TestCutoverRecordRejectsAnUnknownKey(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "  method: revert\n", "  method: revert\n  methd: revert\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "methd") {
		t.Fatalf("status = %d, output = %q; want the misspelled key rejected", status, output)
	}
}

func TestCutoverRecordRejectsAnUncoveredContract(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "    contracts:\n      - mise-task-names\n", "    contracts: []\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "contract mise-task-names appears in no participant") {
		t.Fatalf("status = %d, output = %q; want the uncovered contract reported", status, output)
	}
}

func TestCutoverRecordRejectsAModuleWithNoParticipant(t *testing.T) {
	// The shared ownership fixture declares foundation and policy. Dropping the
	// foundation row leaves that module without one.
	cutover := strings.Replace(fixtureCutover, `  - id: module-foundation
    kind: module
    surface: Shared primitives every other module imports.
    owner: foundation
    contracts: []
    evidence:
      - mise.toml
    readiness: The foundation module imports no other internal module.
    readiness_check: go run ./cmd/check-repository
    recovery: The foundation packages keep their paths, so no caller changes.
`, "", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "module foundation owns no participant") {
		t.Fatalf("status = %d, output = %q; want the uncovered module reported", status, output)
	}
}

func TestCutoverRecordRejectsAnUnknownContract(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "      - mise-task-names\n", "      - mise-task-names\n      - invented-contract\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "invented-contract") {
		t.Fatalf("status = %d, output = %q; want the unrecorded contract reported", status, output)
	}
}

func TestCutoverRecordRejectsAnUnknownOwner(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "    owner: composition\n", "    owner: assembly\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, `names the owner "assembly"`) {
		t.Fatalf("status = %d, output = %q; want the unknown owner reported", status, output)
	}
}

func TestCutoverRecordRejectsAnAbsentEvidencePath(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "      - mise.toml\n", "      - workflow/absent.yml\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "which does not exist") {
		t.Fatalf("status = %d, output = %q; want the absent evidence path reported", status, output)
	}
}

func TestCutoverRecordRejectsADependencyThatDoesNotPrecedeItsSlice(t *testing.T) {
	cutover := strings.Replace(fixtureCutover, "    depends_on:\n      - 328\n", "    depends_on:\n      - 333\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "does not appear earlier in the migration order") {
		t.Fatalf("status = %d, output = %q; want the out-of-order dependency reported", status, output)
	}
}

func TestCutoverRecordRejectsAnUntestedRecovery(t *testing.T) {
	cutover := strings.Replace(fixtureCutover,
		"  rehearsal:\n    - command: go run ./cmd/check-repository\n      observation: The reverted tree reports the baseline total and exits zero.\n",
		"  rehearsal: []\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "records no rehearsal, so it is untested") {
		t.Fatalf("status = %d, output = %q; want the untested recovery reported", status, output)
	}
}

func TestCutoverRecordRejectsAMissingPostCutoverAspect(t *testing.T) {
	cutover := strings.Replace(fixtureCutover,
		"  - id: mutation-authority\n    aspect: authority\n    command: go run ./cmd/check-analyze-readonly\n    observation: No read-only skill mutates the repository.\n",
		"", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "no post-cutover check covers the authority aspect") {
		t.Fatalf("status = %d, output = %q; want the missing aspect reported", status, output)
	}
}

func TestCutoverRecordRejectsDriftFromTheArchitectureRecord(t *testing.T) {
	record := strings.Replace(fixtureRecord, "| mise-tasks | workflow | composition |\n", "", 1)
	status, output := runCutover(t, fixtureCutover, record)
	if status != 1 || !strings.Contains(output, "participant mise-tasks does not appear in docs/architecture.md") {
		t.Fatalf("status = %d, output = %q; want the drift reported", status, output)
	}
}

func TestCutoverRecordRejectsACutoverPointTheRecordDoesNotName(t *testing.T) {
	record := strings.Replace(fixtureRecord, "9f04db42a68e5140e4dc4067a42fdc8b02b230f0", "an unnamed commit", 1)
	status, output := runCutover(t, fixtureCutover, record)
	if status != 1 || !strings.Contains(output, "does not appear in docs/architecture.md") {
		t.Fatalf("status = %d, output = %q; want the unnamed cutover point reported", status, output)
	}
}

func TestCutoverRecordRejectsAnUnapprovedVersionIdentifier(t *testing.T) {
	cutover := strings.Replace(fixtureCutover,
		"  method: revert\n", "  method: revert to the v2 boundary\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "names the version identifier v2") {
		t.Fatalf("status = %d, output = %q; want the version identifier reported", status, output)
	}
}

func TestCutoverRecordAcceptsAPinnedToolVersion(t *testing.T) {
	cutover := strings.Replace(fixtureCutover,
		"    observation: A second replay returns the same outcome.\n",
		"    observation: A second replay returns the same outcome under fslc v4.2.0.\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 0 {
		t.Fatalf("status = %d, want 0: %s", status, output)
	}
}

func TestCutoverRecordRejectsABaselineEqualToTheCutoverPoint(t *testing.T) {
	cutover := strings.Replace(fixtureCutover,
		"baseline: 4cce0641bbc9bc28c9bba47522a6071b0acded69\n",
		"baseline: 9f04db42a68e5140e4dc4067a42fdc8b02b230f0\n", 1)
	status, output := runCutover(t, cutover, fixtureRecord)
	if status != 1 || !strings.Contains(output, "so the record describes no cutover") {
		t.Fatalf("status = %d, output = %q; want the empty range reported", status, output)
	}
}
