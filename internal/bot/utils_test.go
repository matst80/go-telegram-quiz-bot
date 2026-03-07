package bot

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Hello_World", "Hello\\_World"},
		{"Hello*World", "Hello\\*World"},
		{"[Hello] World", "\\[Hello] World"},
		{"`Hello` World", "\\`Hello\\` World"},
		{"Multiple *stars* and _underscores_", "Multiple \\*stars\\* and \\_underscores\\_"},
	}

	for _, tt := range tests {
		got := escapeMarkdown(tt.input)
		if got != tt.expected {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
