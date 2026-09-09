package main

import (
	"testing"

	"github.com/hidekitux/skills/internal/instructions"
)

func TestMeasurementUsesPinnedTokenizer(t *testing.T) {
	if instructions.EncodingName != "cl100k_base" {
		t.Fatalf("encoding = %q, want cl100k_base", instructions.EncodingName)
	}
	if instructions.TokenizerVersion != "v0.1.6" {
		t.Fatalf("tokenizer version = %q, want v0.1.6", instructions.TokenizerVersion)
	}
}
