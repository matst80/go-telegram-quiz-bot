package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Markdown JSON block",
			input:    "Here is your quiz:\n```json\n{\"q\": \"a\"}\n```\nHope you like it!",
			expected: "{\"q\": \"a\"}",
		},
		{
			name:     "Markdown block without language",
			input:    "```\n{\"q\": \"a\"}\n```",
			expected: "{\"q\": \"a\"}",
		},
		{
			name:     "Raw JSON object",
			input:    "  {\"q\": \"a\"}  ",
			expected: "{\"q\": \"a\"}",
		},
		{
			name:     "JSON object with surrounding text",
			input:    "Sure! Here it is: {\"q\": \"a\"} - done.",
			expected: "{\"q\": \"a\"}",
		},
		{
			name:     "Markdown JSON array",
			input:    "Array:\n```json\n[{\"q\": \"a\"}]\n```",
			expected: "[{\"q\": \"a\"}]",
		},
		{
			name:     "Raw JSON array",
			input:    "  [{\"q\": \"a\"}]  ",
			expected: "[{\"q\": \"a\"}]",
		},
		{
			name:     "Raw string (fallback)",
			input:    "simple string",
			expected: "simple string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.expected {
				t.Errorf("extractJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClient_GenerateSpanishQuizzes_Multi(t *testing.T) {
	topic := "Numbers"
	count := 2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req GenerateRequest
		json.Unmarshal(body, &req)

		if !strings.Contains(req.Prompt, fmt.Sprintf("Generate %d", count)) {
			t.Errorf("Prompt should specify count %d", count)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[\n", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"text\": \"1?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"1\"},\n", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"text\": \"2?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"2\"}\n", "done": false}`)
		fmt.Fprintln(w, `{"response": "]\n", "done": true}`)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	questions, err := client.GenerateSpanishQuestions(topic, nil, count)
	if err != nil {
		t.Fatalf("GenerateSpanishQuestions failed: %v", err)
	}

	if len(questions) != count {
		t.Errorf("Expected %d questions, got %d", count, len(questions))
	}
}
