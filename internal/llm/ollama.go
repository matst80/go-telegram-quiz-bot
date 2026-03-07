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
	NumCtx      int     `json:"num_ctx,omitempty"`
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
				"options": {
					"type": "array",
					"items": { "type": "string" },
					"description": "The answer options (usually 4, but can be fewer or more)"
				},
				"correct_index": {
					"type": "integer",
					"description": "The 0-based index of the correct answer in the options array"
				},
				"tts_phrase": {
					"type": "string",
					"description": "A short Spanish phrase or sentence featuring the tested word"
				}
			},
			"required": ["text", "options", "correct_index", "tts_phrase"]
		}`),
	},
}

// segmentTool defines the add_segment tool schema for the model.
var segmentTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "add_segment",
		Description: "Suggest a new Spanish learning segment for beginners. Call this once per segment.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "Short, descriptive title of the segment (e.g. 'At the Restaurant')"
				},
				"description": {
					"type": "string",
					"description": "Brief explanation of what the segment covers"
				}
			},
			"required": ["title", "description"]
		}`),
	},
}

// quizTool defines the add_quiz tool schema for the model.
var quizTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "add_quiz",
		Description: "Suggest a new Spanish quiz topic within a segment. Call this once per quiz.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "Short, descriptive title of the quiz topic (e.g. 'Greeting People')"
				},
				"description": {
					"type": "string",
					"description": "Brief explanation of what the quiz covers"
				}
			},
			"required": ["title", "description"]
		}`),
	},
}

const suggestSegmentSystemPrompt = `You are a Spanish curriculum designer for beginners.
Suggest new learning segments that logically follow the existing plan.
For each segment, call the add_segment tool.`

const suggestQuizSystemPrompt = `You are a Spanish curriculum designer for beginners.
Suggest new quiz topics within a specific learning segment.
For each quiz topic, call the add_quiz tool.`

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

		var questions []domain.Question
		err := c.generateStreamingTools(questionSystemPrompt, userPrompt, []Tool{questionTool}, func(tc ToolCall) error {
			if tc.Function.Name != "add_question" {
				return nil
			}
			q, err := toolCallToQuestion(tc.Function.Arguments)
			if err != nil {
				return err
			}
			questions = append(questions, q)
			return nil
		})

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

// SuggestSectionsWithPrompt asks the LLM to suggest new learning segments based on existing topics and an optional custom prompt.
func (c *Client) SuggestSectionsWithPrompt(existingTopics []string, customPrompt string) ([]SectionSuggestion, error) {
	topicList := "none yet"
	if len(existingTopics) > 0 {
		topicList = strings.Join(existingTopics, ", ")
	}

	userPrompt := fmt.Sprintf("The current learning plan covers: %s.", topicList)
	if customPrompt != "" {
		userPrompt += "\nSpecial Instructions: " + customPrompt
	} else {
		userPrompt += "\nSuggest 3 to 5 NEW topics."
	}
	userPrompt += "\nCall add_segment for each suggestion."

	log.Printf("[LLM] Suggesting sections based on: %s (custom prompt: %q)", topicList, customPrompt)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("[LLM] Attempt %d", attempt)

		var suggestions []SectionSuggestion
		err := c.generateStreamingTools(suggestSegmentSystemPrompt, userPrompt, []Tool{segmentTool}, func(tc ToolCall) error {
			if tc.Function.Name != "add_segment" {
				return nil
			}
			title, _ := tc.Function.Arguments["title"].(string)
			desc, _ := tc.Function.Arguments["description"].(string)
			if title != "" {
				suggestions = append(suggestions, SectionSuggestion{Title: title, Description: desc})
			}
			return nil
		})

		if err != nil {
			lastErr = err
			log.Printf("[LLM] Attempt %d failed: %v", attempt, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(suggestions) == 0 {
			lastErr = fmt.Errorf("model returned 0 tool calls")
			log.Printf("[LLM] Attempt %d: no suggestions returned", attempt)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Printf("[LLM] Generated %d suggestions on attempt %d", len(suggestions), attempt)
		return suggestions, nil
	}

	return nil, fmt.Errorf("failed to suggest sections after 3 attempts: %w", lastErr)
}

// SuggestQuizzesWithPrompt asks the LLM to suggest new quiz topics for a specific segment.
func (c *Client) SuggestQuizzesWithPrompt(segmentTitle string, existingQuizzes []string, customPrompt string) ([]SectionSuggestion, error) {
	quizList := "none yet"
	if len(existingQuizzes) > 0 {
		quizList = strings.Join(existingQuizzes, ", ")
	}

	userPrompt := fmt.Sprintf("The segment is \"%s\". Existing quiz topics are: %s.", segmentTitle, quizList)
	if customPrompt != "" {
		userPrompt += "\nSpecial Instructions: " + customPrompt
	} else {
		userPrompt += "\nSuggest 3 to 5 NEW quiz topics for this segment."
	}
	userPrompt += "\nCall add_quiz for each suggestion."

	log.Printf("[LLM] Suggesting quizzes for segment %q based on: %s (custom prompt: %q)", segmentTitle, quizList, customPrompt)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("[LLM] Attempt %d", attempt)

		var suggestions []SectionSuggestion
		err := c.generateStreamingTools(suggestQuizSystemPrompt, userPrompt, []Tool{quizTool}, func(tc ToolCall) error {
			if tc.Function.Name != "add_quiz" {
				return nil
			}
			title, _ := tc.Function.Arguments["title"].(string)
			desc, _ := tc.Function.Arguments["description"].(string)
			if title != "" {
				suggestions = append(suggestions, SectionSuggestion{Title: title, Description: desc})
			}
			return nil
		})

		if err != nil {
			lastErr = err
			log.Printf("[LLM] Attempt %d failed: %v", attempt, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(suggestions) == 0 {
			lastErr = fmt.Errorf("model returned 0 tool calls")
			log.Printf("[LLM] Attempt %d: no suggestions returned", attempt)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Printf("[LLM] Generated %d quiz suggestions on attempt %d", len(suggestions), attempt)
		return suggestions, nil
	}

	return nil, fmt.Errorf("failed to suggest quizzes after 3 attempts: %w", lastErr)
}

// SuggestSections is a convenience wrapper for backward compatibility.
func (c *Client) SuggestSections(existingTopics []string) ([]SectionSuggestion, error) {
	return c.SuggestSectionsWithPrompt(existingTopics, "")
}

// generateStreamingTools sends a streaming chat request with tool definitions and invokes the callback for each tool call.
func (c *Client) generateStreamingTools(system, user string, tools []Tool, onToolCall func(ToolCall) error) error {
	reqBody := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: true,
		Tools:  tools,
		Options: &ModelOptions{
			Temperature: 0.7,
			NumCtx:      16000,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		var chunk ChatStreamChunk
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}

		for _, tc := range chunk.Message.ToolCalls {
			if err := onToolCall(tc); err != nil {
				log.Printf("[LLM] Tool call handler error: %v", err)
				return err
			}
		}

		if chunk.Done {
			break
		}
	}

	return scanner.Err()
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

	rawOptions, ok := args["options"]
	if !ok {
		return domain.Question{}, fmt.Errorf("missing field \"options\"")
	}

	optionsArr, ok := rawOptions.([]interface{})
	if !ok {
		return domain.Question{}, fmt.Errorf("field \"options\" is not an array")
	}

	var options []string
	for i, opt := range optionsArr {
		s, ok := opt.(string)
		if !ok {
			return domain.Question{}, fmt.Errorf("option at index %d is not a string", i)
		}
		options = append(options, s)
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

	if correctIndex < 0 || correctIndex >= len(options) {
		return domain.Question{}, fmt.Errorf("correct_index %d out of range (0-%d)", correctIndex, len(options)-1)
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
			NumCtx:      16000,
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
