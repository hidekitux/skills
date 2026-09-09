package validate

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runPrSignatureCheck(t *testing.T, fixture string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckPrCommitSignatures("", 0, filepath.Join("testdata", fixture), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCheckPrCommitSignaturesAcceptsFullyVerifiedFixture(t *testing.T) {
	code, out, errOut := runPrSignatureCheck(t, "commits-all-verified.json")
	if code != 0 {
		t.Fatalf("expected success, got exit %d: %s", code, errOut)
	}
	if out != "All 2 pull-request commits have GitHub-verified signatures.\n" {
		t.Fatalf("unexpected output %q", out)
	}
	if errOut != "" {
		t.Fatalf("unexpected error output %q", errOut)
	}
}

func TestCheckPrCommitSignaturesRejectsUnsignedFixture(t *testing.T) {
	code, out, errOut := runPrSignatureCheck(t, "commits-unsigned.json")
	if code != 1 {
		t.Fatalf("expected rejection, got exit %d: %s", code, errOut)
	}
	if out != "" {
		t.Fatalf("unexpected output %q", out)
	}
	for _, want := range []string{
		"error: pull request contains unverified commits:",
		"- unsigned-1: unsigned",
	} {
		if !strings.Contains(errOut, want) {
			t.Errorf("error output %q does not contain %q", errOut, want)
		}
	}
}

func TestCheckPrCommitSignaturesRejectsPartiallyVerifiedFixture(t *testing.T) {
	code, out, errOut := runPrSignatureCheck(t, "commits-partially-verified.json")
	if code != 1 {
		t.Fatalf("expected rejection, got exit %d: %s", code, errOut)
	}
	if out != "" {
		t.Fatalf("unexpected output %q", out)
	}
	if !strings.Contains(errOut, "- unknown-key-1: unknown_key") {
		t.Fatalf("error output %q does not identify the unverified commit", errOut)
	}
}

func TestCheckPrCommitSignaturesRejectsUnusableFixture(t *testing.T) {
	code, out, errOut := runPrSignatureCheck(t, "commits-malformed.json")
	if code != 2 {
		t.Fatalf("expected fixture error, got exit %d: %s", code, errOut)
	}
	if out != "" {
		t.Fatalf("unexpected output %q", out)
	}
	if !strings.Contains(errOut, "cannot load commits fixture") ||
		!strings.Contains(errOut, "cannot unmarshal object") {
		t.Fatalf("error output %q does not identify the unusable fixture", errOut)
	}
}

func TestCheckPrCommitSignaturesRejectsMissingFixture(t *testing.T) {
	var out, errOut bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing.json")
	code := CheckPrCommitSignatures("", 0, missing, &out, &errOut)
	if code != 2 {
		t.Fatalf("expected fixture error, got exit %d: %s", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected output %q", out.String())
	}
	if !strings.Contains(errOut.String(), "cannot load commits fixture") ||
		!strings.Contains(errOut.String(), "missing.json") {
		t.Fatalf("error output %q does not identify the missing fixture", errOut.String())
	}
}
