package wiktionary

import (
	"testing"
)

func TestStripMarkup(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "simple bold tags",
			in:   "<b>word</b>",
			want: "word",
		},
		{
			name: "simple italic tags",
			in:   "<i>word</i>",
			want: "word",
		},
		{
			name: "anchor tag with attributes",
			in:   `<a href="http://example.com">link text</a>`,
			want: "link text",
		},
		{
			name: "span tag",
			in:   `<span class="foo">content</span>`,
			want: "content",
		},
		{
			name: "self-closing tag",
			in:   "before<br/>after",
			want: "beforeafter",
		},
		{
			name: "wiki link simple",
			in:   "[[word]]",
			want: "word",
		},
		{
			name: "wiki link with display text",
			in:   "[[link|display]]",
			want: "display",
		},
		{
			name: "nested HTML and wiki",
			in:   "<b>[[word]]</b>",
			want: "word",
		},
		{
			name: "nested HTML and wiki with display",
			in:   "<i>[[link|shown]]</i>",
			want: "shown",
		},
		{
			name: "multiple spaces collapsed",
			in:   "word   with   spaces",
			want: "word with spaces",
		},
		{
			name: "spaces after tag removal",
			in:   "<b>hello</b>  <i>world</i>",
			want: "hello world",
		},
		{
			name: "leading and trailing whitespace trimmed",
			in:   "  hello  ",
			want: "hello",
		},
		{
			name: "multi-tag mixed content",
			in:   "<b>word</b> and <i>other</i>",
			want: "word and other",
		},
		{
			name: "complex mixed markup",
			in:   `The <b>quick</b> [[brown|brown]] <i>fox</i> [[jumped]]`,
			want: "The quick brown fox jumped",
		},
		{
			name: "wiki link in sentence",
			in:   "to [[run]] fast",
			want: "to run fast",
		},
		{
			name: "multiple wiki links",
			in:   "[[one]] and [[two|2]]",
			want: "one and 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripMarkup(tt.in)
			if got != tt.want {
				t.Errorf("StripMarkup(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncateDefinition(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			in:     "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exactly maxLen unchanged",
			in:     "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "over maxLen truncated with ellipsis",
			in:     "hello world",
			maxLen: 5,
			want:   "hello…",
		},
		{
			name:   "empty string",
			in:     "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "very long string",
			in:     "abcdefghijklmnopqrstuvwxyz",
			maxLen: 3,
			want:   "abc…",
		},
		{
			name:   "maxLen of 1",
			in:     "hello",
			maxLen: 1,
			want:   "h…",
		},
		{
			name:   "one char over",
			in:     "abcdef",
			maxLen: 5,
			want:   "abcde…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateDefinition(tt.in, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateDefinition(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "unique list unchanged",
			in:   []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "duplicates removed preserving order",
			in:   []string{"a", "b", "a"},
			want: []string{"a", "b"},
		},
		{
			name: "all same",
			in:   []string{"x", "x", "x"},
			want: []string{"x"},
		},
		{
			name: "nil input returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty slice returns empty",
			in:   []string{},
			want: []string{},
		},
		{
			name: "multiple duplicates mixed",
			in:   []string{"a", "b", "c", "b", "a", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "single element",
			in:   []string{"only"},
			want: []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeduplicateStrings(tt.in)

			// Check nil case
			if tt.want == nil {
				if got != nil {
					t.Errorf("DeduplicateStrings(nil) = %v, want nil", got)
				}
				return
			}

			// Check length
			if len(got) != len(tt.want) {
				t.Errorf("DeduplicateStrings(%v) length = %d, want %d; got %v", tt.in, len(got), len(tt.want), got)
				return
			}

			// Check elements
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("DeduplicateStrings(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}
