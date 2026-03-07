package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

		if !strings.Contains(req.Prompt, fmt.Sprintf("Generate exactly %d", count)) {
			t.Errorf("Prompt should specify count %d", count)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[\n", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"text\": \"1?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"1\", \"tts_phrase\": \"1\"},\n", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"text\": \"2?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"2\", \"tts_phrase\": \"2\"}\n", "done": false}`)
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

func TestClient_StreamingTimeout(t *testing.T) {
	// A server that takes longer to respond than the client's timeout
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[\n", "done": false}`)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Wait longer than the client timeout
		time.Sleep(2 * time.Second)
		fmt.Fprintln(w, `{"response": "]\n", "done": true}`)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	// Set an artificially short overall timeout to reproduce the error
	client.HTTPClient.Timeout = 1 * time.Second

	_, err := client.GenerateSpanishQuestions("Numbers", nil, 1)

	if err == nil {
		t.Fatal("Expected an error due to timeout, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "Client.Timeout") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected context deadline exceeded or timeout error, got: %v", err)
	}
}

func TestClient_StreamingSuccessWithSufficientTimeout(t *testing.T) {
	// A server that responds slowly but within the client's timeout
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[\n", "done": false}`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(1 * time.Second)
		fmt.Fprintln(w, `{"response": "{\"text\": \"1?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"1\", \"tts_phrase\": \"1\"}\n", "done": false}`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(1 * time.Second)
		fmt.Fprintln(w, `{"response": "]\n", "done": true}`)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	// Set an artificially short timeout but sufficient for this test
	client.HTTPClient.Timeout = 5 * time.Second

	questions, err := client.GenerateSpanishQuestions("Numbers", nil, 1)

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if len(questions) != 1 {
		t.Errorf("Expected 1 question, got %d", len(questions))
	}
}

func TestSuggestSections(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req GenerateRequest
		json.Unmarshal(body, &req)

		if !strings.Contains(req.Prompt, "Basic Greetings") {
			t.Errorf("Prompt should contain existing topics, got: %s", req.Prompt)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"title\": \"Weather\", \"description\": \"Learn weather vocabulary\"},", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"title\": \"Clothing\", \"description\": \"Learn clothing items\"}", "done": false}`)
		fmt.Fprintln(w, `{"response": "]", "done": true}`)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	suggestions, err := client.SuggestSections([]string{"Basic Greetings", "Numbers"})
	if err != nil {
		t.Fatalf("SuggestSections failed: %v", err)
	}

	if len(suggestions) != 2 {
		t.Fatalf("Expected 2 suggestions, got %d", len(suggestions))
	}

	if suggestions[0].Title != "Weather" {
		t.Errorf("Expected first suggestion title 'Weather', got '%s'", suggestions[0].Title)
	}
	if suggestions[1].Title != "Clothing" {
		t.Errorf("Expected second suggestion title 'Clothing', got '%s'", suggestions[1].Title)
	}
}
