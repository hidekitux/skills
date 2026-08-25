package commitlint

import "testing"

func TestValidateMessageAcceptsSingleLineWithIssue(t *testing.T) {
	if errors := ValidateMessage("feat: add login flow #61"); len(errors) != 0 {
		t.Fatalf("expected valid, got %v", errors)
	}
	if errors := ValidateMessage("feat(scope): add login flow #61"); len(errors) != 0 {
		t.Fatalf("expected valid, got %v", errors)
	}
}

func TestValidateMessageRequiresIssueNumber(t *testing.T) {
	if errors := ValidateMessage("feat: add login flow"); len(errors) == 0 {
		t.Fatal("expected missing-issue error")
	}
}

func TestValidateMessageRejectsBody(t *testing.T) {
	if errors := ValidateMessage("feat: add login flow #61\n\nbody"); len(errors) == 0 {
		t.Fatal("expected body error")
	}
}

func TestValidateMessageRejectsEmpty(t *testing.T) {
	if errors := ValidateMessage(""); len(errors) == 0 {
		t.Fatal("expected empty-message error")
	}
}

func TestValidateMessageRejectsTerminalPunctuation(t *testing.T) {
	if errors := ValidateMessage("feat: add login flow. #61"); len(errors) == 0 {
		t.Fatal("expected terminal-punctuation error")
	}
}
