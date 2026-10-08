package textutil

import "testing"

func TestSlug(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"words", "Hello World", "hello-world"},
		{"punctuation", "Go, Kotlin & Rust!", "go-kotlin-rust"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slug(tt.in); got != tt.want {
				t.Errorf("Slug(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
