package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mats/telegram-quiz-bot/internal/domain"
)

type Client struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// ChatMessage represents a single message in the chat API.
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation returned by the model.
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the function name and arguments from a tool call.
type ToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Tool defines a tool the model can use.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function schema for a tool.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Format   string        `json:"format,omitempty"`
	Stream   bool          `json:"stream"`
	Tools    []Tool        `json:"tools,omitempty"`
	Options  *ModelOptions `json:"options,omitempty"`
}

// ModelOptions holds tunable generation parameters.
type ModelOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ChatStreamChunk is a single streamed chunk from POST /api/chat.
type ChatStreamChunk struct {
	Model    string      `json:"model"`
	Message  ChatMessage `json:"message"`
	Done     bool        `json:"done"`
	TotalDur int64       `json:"total_duration"`
	EvalDur  int64       `json:"eval_duration"`
}

func NewClient(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen3.5:9b"
	}
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// questionTool defines the add_question tool schema for the model.
var questionTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "add_question",
		Description: "Add a Spanish quiz question for beginners. Call this once per question.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "The quiz question text, written in English"
				},
				"option_a": {
					"type": "string",
					"description": "First answer option"
				},
				"option_b": {
					"type": "string",
					"description": "Second answer option"
				},
				"option_c": {
					"type": "string",
					"description": "Third answer option"
				},
				"option_d": {
					"type": "string",
					"description": "Fourth answer option"
				},
				"correct_index": {
					"type": "integer",
					"description": "The 0-based index of the correct answer (0 for option_a, 1 for option_b, 2 for option_c, 3 for option_d)"
				},
				"tts_phrase": {
					"type": "string",
					"description": "A short Spanish phrase or sentence featuring the tested word"
				}
			},
			"required": ["text", "option_a", "option_b", "option_c", "option_d", "correct_index", "tts_phrase"]
		}`),
	},
}

const questionSystemPrompt = `You are an expert Spanish teacher creating quiz questions for beginners.
For each question, call the add_question tool. Generate all requested questions by calling the tool multiple times.
Keep vocabulary simple and beginner-friendly. Write question text in English.`

func (c *Client) GenerateSpanishQuestions(topic string, excludeQuestions []string, count int) ([]domain.Question, error) {
	userPrompt := fmt.Sprintf("Generate exactly %d quiz questions about the topic: \"%s\". Call add_question once per question.", count, topic)

	if len(excludeQuestions) > 0 {
		userPrompt += "\nDo NOT reuse these questions:\n"
		for _, q := range excludeQuestions {
			userPrompt += fmt.Sprintf("- %s\n", q)
		}
	}

	log.Printf("[LLM] Generating %d questions for topic: %s", count, topic)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("[LLM] Attempt %d", attempt)

		questions, err := c.generateWithTools(questionSystemPrompt, userPrompt, questionTool)
		if err != nil {
			lastErr = err
			log.Printf("[LLM] Attempt %d failed: %v", attempt, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(questions) == 0 {
			lastErr = fmt.Errorf("model returned 0 tool calls")
			log.Printf("[LLM] Attempt %d: no questions returned", attempt)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Printf("[LLM] Generated %d questions on attempt %d", len(questions), attempt)
		return questions, nil
	}

	return nil, fmt.Errorf("failed to generate questions after 3 attempts: %w", lastErr)
}

// SectionSuggestion represents an AI-suggested learning section.
type SectionSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

const suggestSystemPrompt = `You are a Spanish curriculum designer for beginners.
Rules:
- Output ONLY a JSON array of topic objects.
- Each object must have: "title" (topic name), "description" (brief description of the topic).
- Suggest topics that logically follow and build on what the student has already learned.
- ALWAYS return a JSON array.`

// SuggestSections asks the LLM to suggest new learning segments based on existing topics.
func (c *Client) SuggestSections(existingTopics []string) ([]SectionSuggestion, error) {
	topicList := "none yet"
	if len(existingTopics) > 0 {
		topicList = strings.Join(existingTopics, ", ")
	}

	userPrompt := fmt.Sprintf("The current learning plan covers: %s.\nSuggest 3 to 5 NEW topics. Return a JSON array.", topicList)

	raw, err := c.chatJSON(suggestSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate section suggestions: %w", err)
	}

	jsonContent := extractJSON(raw)
	var suggestions []SectionSuggestion
	if err := json.Unmarshal([]byte(jsonContent), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse suggestions JSON: %w", err)
	}

	return suggestions, nil
}

// generateWithTools sends a streaming chat request with a tool definition.
// Each tool call chunk is processed immediately as it arrives from the stream.
func (c *Client) generateWithTools(system, user string, tool Tool) ([]domain.Question, error) {
	reqBody := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: true,
		Tools:  []Tool{tool},
		Options: &ModelOptions{
			Temperature: 0.7,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var questions []domain.Question
	var textContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	chunks := 0

	for scanner.Scan() {
		var chunk ChatStreamChunk
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			log.Printf("[LLM] Warning: failed to parse stream chunk: %v", err)
			continue
		}
		chunks++

		// Collect any text content (thinking tokens, etc.)
		if chunk.Message.Content != "" {
			textContent.WriteString(chunk.Message.Content)
		}

		// Process tool calls as they arrive
		for _, tc := range chunk.Message.ToolCalls {
			if tc.Function.Name != "add_question" {
				log.Printf("[LLM] Warning: unexpected tool call %q, skipping", tc.Function.Name)
				continue
			}

			q, err := toolCallToQuestion(tc.Function.Arguments)
			if err != nil {
				log.Printf("[LLM] Warning: tool call invalid: %v", err)
				continue
			}

			questions = append(questions, q)
			log.Printf("[LLM] ✓ Question %d streamed (%.1fs): %s",
				len(questions), time.Since(start).Seconds(), q.Text)
		}

		if chunk.Done {
			log.Printf("[LLM] Stream done: %d chunks, %d questions, %.1fs (eval: %dms)",
				chunks, len(questions), time.Since(start).Seconds(), chunk.EvalDur/1_000_000)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ollama stream: %w", err)
	}

	// Log any text the model emitted (thinking, etc.) for debugging
	if text := textContent.String(); text != "" {
		log.Printf("[LLM] Model text output: %.500s", text)
	}

	return questions, nil
}

// toolCallToQuestion converts tool call arguments to a domain.Question.
func toolCallToQuestion(args map[string]interface{}) (domain.Question, error) {
	getString := func(key string) (string, error) {
		v, ok := args[key]
		if !ok {
			return "", fmt.Errorf("missing field %q", key)
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("field %q is not a string", key)
		}
		return s, nil
	}

	text, err := getString("text")
	if err != nil {
		return domain.Question{}, err
	}

	optA, err := getString("option_a")
	if err != nil {
		return domain.Question{}, err
	}
	optB, err := getString("option_b")
	if err != nil {
		return domain.Question{}, err
	}
	optC, err := getString("option_c")
	if err != nil {
		return domain.Question{}, err
	}
	optD, err := getString("option_d")
	if err != nil {
		return domain.Question{}, err
	}

	correctIndexVal, ok := args["correct_index"]
	if !ok {
		return domain.Question{}, fmt.Errorf("missing field \"correct_index\"")
	}

	var correctIndex int
	switch v := correctIndexVal.(type) {
	case float64:
		correctIndex = int(v)
	case int:
		correctIndex = v
	default:
		return domain.Question{}, fmt.Errorf("field \"correct_index\" is not a number")
	}

	ttsPhrase, err := getString("tts_phrase")
	if err != nil {
		return domain.Question{}, err
	}

	options := []string{optA, optB, optC, optD}

	if correctIndex < 0 || correctIndex >= len(options) {
		return domain.Question{}, fmt.Errorf("correct_index %d out of range (0-3)", correctIndex)
	}
	correct := options[correctIndex]

	return domain.Question{
		Text:          text,
		Options:       options,
		CorrectAnswer: correct,
		TTSPhrase:     ttsPhrase,
	}, nil
}

// chatJSON sends a system+user message pair to /api/chat with format:"json" (no tools).
// Used for non-tool-call requests like SuggestSections.
func (c *Client) chatJSON(system, user string) (string, error) {
	reqBody := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Format: "json",
		Stream: false,
		Options: &ModelOptions{
			Temperature: 0.7,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatStreamChunk
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	raw := chatResp.Message.Content
	log.Printf("[LLM] Response in %.1fs (%d chars): %.200s...", time.Since(start).Seconds(), len(raw), raw)
	return raw, nil
}

func extractJSON(s string) string {
	if start := strings.Index(s, "```json"); start != -1 {
		inner := s[start+7:]
		if end := strings.Index(inner, "```"); end != -1 {
			return strings.TrimSpace(inner[:end])
		}
	}
	if start := strings.Index(s, "```"); start != -1 {
		inner := s[start+3:]
		if end := strings.Index(inner, "```"); end != -1 {
			return strings.TrimSpace(inner[:end])
		}
	}

	startObj := strings.Index(s, "{")
	endObj := strings.LastIndex(s, "}")
	startArr := strings.Index(s, "[")
	endArr := strings.LastIndex(s, "]")

	if startArr != -1 && endArr != -1 && endArr > startArr {
		if startObj == -1 || startArr < startObj {
			return strings.TrimSpace(s[startArr : endArr+1])
		}
	}

	if startObj != -1 && endObj != -1 && endObj > startObj {
		return strings.TrimSpace(s[startObj : endObj+1])
	}

	return strings.TrimSpace(s)
}
