// Package textutil holds small text helpers.
package textutil

import (
	"strings"
	"unicode"
)

// Slug lowercases s and joins its letter and digit runs with hyphens.
func Slug(s string) string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(fields, "-")
}
