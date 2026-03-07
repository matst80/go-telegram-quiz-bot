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

func TestToolCallToQuestion(t *testing.T) {
	args := map[string]interface{}{
		"text":          "What does 'uno' mean?",
		"option_a":      "One",
		"option_b":      "Two",
		"option_c":      "Three",
		"option_d":      "Four",
		"correct_index": 0.0, // decoding from json can give float64
		"tts_phrase":    "El número uno",
	}

	q, err := toolCallToQuestion(args)
	if err != nil {
		t.Fatalf("toolCallToQuestion failed: %v", err)
	}

	if q.Text != "What does 'uno' mean?" {
		t.Errorf("unexpected text: %s", q.Text)
	}
	if len(q.Options) != 4 {
		t.Errorf("expected 4 options, got %d", len(q.Options))
	}
	if q.CorrectAnswer != "One" {
		t.Errorf("unexpected correct answer: %s", q.CorrectAnswer)
	}
	if q.TTSPhrase != "El número uno" {
		t.Errorf("unexpected tts_phrase: %s", q.TTSPhrase)
	}
}

func TestToolCallToQuestion_InvalidCorrectIndex(t *testing.T) {
	args := map[string]interface{}{
		"text":          "Test?",
		"option_a":      "A",
		"option_b":      "B",
		"option_c":      "C",
		"option_d":      "D",
		"correct_index": 4, // Out of range
		"tts_phrase":    "test",
	}

	_, err := toolCallToQuestion(args)
	if err == nil {
		t.Fatal("expected error for invalid correct_index, got nil")
	}
}

func TestToolCallToQuestion_MissingField(t *testing.T) {
	args := map[string]interface{}{
		"text": "Test?",
	}

	_, err := toolCallToQuestion(args)
	if err == nil {
		t.Fatal("expected error for missing fields, got nil")
	}
}

func makeQuestionToolCall(text, a, b, c, d string, correctIndex int, tts string) ToolCall {
	return ToolCall{
		Function: ToolCallFunction{
			Name: "add_question",
			Arguments: map[string]interface{}{
				"text":          text,
				"option_a":      a,
				"option_b":      b,
				"option_c":      c,
				"option_d":      d,
				"correct_index": correctIndex,
				"tts_phrase":    tts,
			},
		},
	}
}

// streamingToolHandler simulates Ollama's streaming tool call format.
// Each tool call arrives as a separate ndjson chunk with tool_calls in message.
func streamingToolHandler(t *testing.T, validateReq func(*ChatRequest), toolCalls []ToolCall) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if !req.Stream {
			t.Error("expected stream=true for streaming tool calls")
		}
		if len(req.Tools) == 0 {
			t.Error("expected tools to be provided")
		}

		if validateReq != nil {
			validateReq(&req)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)

		// Optionally emit a thinking chunk first (like qwen3 does)
		thinkChunk := ChatStreamChunk{
			Model:   req.Model,
			Message: ChatMessage{Role: "assistant", Content: "<think>planning questions</think>"},
			Done:    false,
		}
		data, _ := json.Marshal(thinkChunk)
		fmt.Fprintln(w, string(data))
		if flusher != nil {
			flusher.Flush()
		}

		// Stream each tool call as a separate chunk
		for _, tc := range toolCalls {
			chunk := ChatStreamChunk{
				Model: req.Model,
				Message: ChatMessage{
					Role:      "assistant",
					Content:   "",
					ToolCalls: []ToolCall{tc},
				},
				Done: false,
			}
			data, _ := json.Marshal(chunk)
			fmt.Fprintln(w, string(data))
			if flusher != nil {
				flusher.Flush()
			}
		}

		// Final done chunk
		doneChunk := ChatStreamChunk{
			Model:   req.Model,
			Message: ChatMessage{Role: "assistant"},
			Done:    true,
		}
		data, _ = json.Marshal(doneChunk)
		fmt.Fprintln(w, string(data))
	}
}

func TestClient_GenerateSpanishQuestions(t *testing.T) {
	topic := "Numbers"
	count := 2

	toolCalls := []ToolCall{
		makeQuestionToolCall("What does 'uno' mean?", "One", "Two", "Three", "Four", 0, "El número uno"),
		makeQuestionToolCall("What does 'dos' mean?", "One", "Two", "Three", "Four", 1, "El número dos"),
	}

	ts := httptest.NewServer(streamingToolHandler(t, func(req *ChatRequest) {
		if len(req.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected system role, got %s", req.Messages[0].Role)
		}
		if !strings.Contains(req.Messages[1].Content, fmt.Sprintf("Generate exactly %d", count)) {
			t.Errorf("user prompt should specify count %d", count)
		}
		if req.Tools[0].Function.Name != "add_question" {
			t.Errorf("expected tool name add_question, got %s", req.Tools[0].Function.Name)
		}
	}, toolCalls))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	questions, err := client.GenerateSpanishQuestions(topic, nil, count)
	if err != nil {
		t.Fatalf("GenerateSpanishQuestions failed: %v", err)
	}

	if len(questions) != count {
		t.Errorf("Expected %d questions, got %d", count, len(questions))
	}
	if questions[0].Text != "What does 'uno' mean?" {
		t.Errorf("unexpected first question text: %s", questions[0].Text)
	}
	if questions[1].CorrectAnswer != "Two" {
		t.Errorf("unexpected second correct answer: %s", questions[1].CorrectAnswer)
	}
}

func TestClient_NoToolCallsRetried(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/x-ndjson")
		// Model only emits text, no tool calls
		chunk := ChatStreamChunk{
			Message: ChatMessage{Role: "assistant", Content: "I can't do that"},
			Done:    false,
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintln(w, string(data))

		done := ChatStreamChunk{
			Message: ChatMessage{Role: "assistant"},
			Done:    true,
		}
		data, _ = json.Marshal(done)
		fmt.Fprintln(w, string(data))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	_, err := client.GenerateSpanishQuestions("Numbers", nil, 1)
	if err == nil {
		t.Fatal("expected error when model returns no tool calls")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/x-ndjson")
		done := ChatStreamChunk{
			Message: ChatMessage{Role: "assistant"},
			Done:    true,
		}
		data, _ := json.Marshal(done)
		fmt.Fprintln(w, string(data))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	client.HTTPClient.Timeout = 500 * time.Millisecond

	_, err := client.GenerateSpanishQuestions("Numbers", nil, 1)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestSuggestSections(t *testing.T) {
	content := `[{"title": "Weather", "description": "Learn weather vocabulary"}, {"title": "Clothing", "description": "Learn clothing items"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req ChatRequest
		json.Unmarshal(body, &req)

		if req.Format != "json" {
			t.Errorf("expected format=json for suggest, got %q", req.Format)
		}
		if !strings.Contains(req.Messages[1].Content, "Basic Greetings") {
			t.Errorf("user prompt should contain existing topics")
		}

		resp := ChatStreamChunk{
			Message: ChatMessage{Role: "assistant", Content: content},
			Done:    true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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
		t.Errorf("Expected 'Weather', got '%s'", suggestions[0].Title)
	}
}

func TestClient_SkipsBadToolCalls(t *testing.T) {
	toolCalls := []ToolCall{
		makeQuestionToolCall("Good question?", "A", "B", "C", "D", 0, "bueno"),
		{Function: ToolCallFunction{Name: "unknown_tool", Arguments: map[string]interface{}{}}},
		makeQuestionToolCall("Another good one?", "X", "Y", "Z", "W", 0, "otro"),
	}

	ts := httptest.NewServer(streamingToolHandler(t, nil, toolCalls))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	questions, err := client.GenerateSpanishQuestions("Test", nil, 2)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(questions) != 2 {
		t.Errorf("expected 2 valid questions (skipping bad tool call), got %d", len(questions))
	}
}

func TestClient_StreamingWithThinkTokens(t *testing.T) {
	// Verifies thinking content is captured but doesn't interfere with tool calls
	toolCalls := []ToolCall{
		makeQuestionToolCall("What is 'hola'?", "Hello", "Bye", "Yes", "No", 0, "Hola amigo"),
	}

	ts := httptest.NewServer(streamingToolHandler(t, nil, toolCalls))
	defer ts.Close()

	client := NewClient(ts.URL, "test-model")
	questions, err := client.GenerateSpanishQuestions("Greetings", nil, 1)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if len(questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(questions))
	}
	if questions[0].TTSPhrase != "Hola amigo" {
		t.Errorf("unexpected tts_phrase: %s", questions[0].TTSPhrase)
	}
}
