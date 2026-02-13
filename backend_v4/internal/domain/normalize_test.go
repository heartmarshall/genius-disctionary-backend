package domain

import "testing"

func TestNormalizeText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim spaces", input: "  hello  ", want: "hello"},
		{name: "lowercase", input: "Hello World", want: "hello world"},
		{name: "compress multiple spaces", input: "hello   world", want: "hello world"},
		{name: "diacritics preserved", input: "Café", want: "café"},
		{name: "hyphens preserved", input: "well-known", want: "well-known"},
		{name: "apostrophes preserved", input: "don't", want: "don't"},
		{name: "empty string", input: "", want: ""},
		{name: "only spaces", input: "   ", want: ""},
		{name: "mixed", input: "  Hello   World  ", want: "hello world"},
		{name: "tabs and spaces", input: "\t hello \t", want: "hello"},
		{name: "unicode diacritics", input: "Naïve Résumé", want: "naïve résumé"},
		{name: "single word", input: "ABANDON", want: "abandon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeText(tt.input); got != tt.want {
				t.Errorf("NormalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
