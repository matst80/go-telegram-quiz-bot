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

func TestClient_GenerateSpanishQuiz_PromptAndResponse(t *testing.T) {
	topic := "Colors"
	exclude := []string{"What is red?"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Validate Prompt in Request Body
		body, _ := io.ReadAll(r.Body)
		var req GenerateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}

		if !strings.Contains(req.Prompt, topic) {
			t.Errorf("Prompt should contain topic '%s'", topic)
		}
		for _, ex := range exclude {
			if !strings.Contains(req.Prompt, ex) {
				t.Errorf("Prompt should contain excluded question '%s'", ex)
			}
		}

		// 2. Return Mocked Response
		w.Header().Set("Content-Type", "application/x-ndjson")
		response := fmt.Sprintf(`{"response": "{\n  \"topic\": \"%s\",\n  \"question\": \"What is Blue?\",\n  \"options\": [\"Azul\", \"Rojo\", \"Verde\", \"Amarillo\"],\n  \"correct_answer\": \"Azul\"\n}", "done": true}`, topic)
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	quiz, err := client.GenerateSpanishQuiz(topic, exclude)
	if err != nil {
		t.Fatalf("GenerateSpanishQuiz failed: %v", err)
	}

	if quiz.Topic != topic {
		t.Errorf("Expected topic %s, got %s", topic, quiz.Topic)
	}
	if quiz.Question != "What is Blue?" {
		t.Errorf("Expected question 'What is Blue?', got %s", quiz.Question)
	}
	if len(quiz.Options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(quiz.Options))
	}
	if quiz.CorrectAnswer != "Azul" {
		t.Errorf("Expected correct answer Azul, got %s", quiz.CorrectAnswer)
	}
}

func TestClient_GenerateSpanishQuizzes_Multi(t *testing.T) {
	topic := "Numbers"
	count := 2

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Count in Prompt
		body, _ := io.ReadAll(r.Body)
		var req GenerateRequest
		json.Unmarshal(body, &req)

		if !strings.Contains(req.Prompt, fmt.Sprintf("Generate %d", count)) {
			t.Errorf("Prompt should specify count %d", count)
		}

		// Return Mocked Response (JSON Array)
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"response": "[", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"topic\": \"Numbers\", \"question\": \"1?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"1\"},", "done": false}`)
		fmt.Fprintln(w, `{"response": "{\"topic\": \"Numbers\", \"question\": \"2?\", \"options\": [\"1\",\"2\",\"3\",\"4\"], \"correct_answer\": \"2\"}", "done": false}`)
		fmt.Fprintln(w, `{"response": "]", "done": true}`)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	quizzes, err := client.GenerateSpanishQuizzes(topic, nil, count)
	if err != nil {
		t.Fatalf("GenerateSpanishQuizzes failed: %v", err)
	}

	if len(quizzes) != count {
		t.Errorf("Expected %d quizzes, got %d", count, len(quizzes))
	}
}
