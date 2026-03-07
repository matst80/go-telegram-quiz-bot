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

type GenerateRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Stream   bool   `json:"stream"`
	Thinking bool   `json:"thinking"`
}

type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context"`
	TotalDur  int64  `json:"total_duration"`
	LoadDur   int64  `json:"load_duration"`
	PromptDur int64  `json:"prompt_eval_duration"`
	EvalDur   int64  `json:"eval_duration"`
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

func (c *Client) GenerateSpanishQuestions(topic string, excludeQuestions []string, count int) ([]domain.Question, error) {
	prompt := fmt.Sprintf(`You are an expert Spanish teacher. Generate exactly %d basic Spanish quiz questions for beginners.
Respond ONLY with a valid JSON array. Do not include any explanations, greetings, or a thinking process. Do NOT output <think> blocks.
Format your output exactly like this:
[
  {
    "text": "What does the Spanish word 'Hola' mean?",
    "options": ["Hello", "Goodbye", "Please", "Thank you"],
    "correct_answer": "Hello",
    "tts_phrase": "Hola"
  }
]
Ensure each quiz question has exactly 4 options. All quiz questions MUST be about the given topic: "%s".
The "text" field MUST be written in English so beginners can understand.
The "tts_phrase" MUST be a short Spanish phrase or sentence featuring the Spanish word or concept being tested.
Keep the vocabulary simple.`, count, topic)

	if len(excludeQuestions) > 0 {
		prompt += "\nDo NOT use these questions again:\n"
		for _, q := range excludeQuestions {
			prompt += fmt.Sprintf("- %s\n", q)
		}
	}
	prompt += "\nRespond ONLY with the JSON array."
	log.Print(prompt)
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("[LLM] Attempt %d: Generating %d questions for topic: %s\n", attempt, count, topic)

		questions, err := c.generateMulti(prompt)
		if err == nil {
			fmt.Printf("[LLM] Successfully generated %d questions on attempt %d\n", len(questions), attempt)
			return questions, nil
		}

		lastErr = err
		fmt.Printf("[LLM] Attempt %d failed: %v\n", attempt, err)
		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("failed to generate questions after 3 attempts: %w", lastErr)
}

// SectionSuggestion represents an AI-suggested learning section.
type SectionSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SuggestSections asks the LLM to suggest new learning segments based on existing topics.
func (c *Client) SuggestSections(existingTopics []string) ([]SectionSuggestion, error) {
	topicList := "none yet"
	if len(existingTopics) > 0 {
		topicList = strings.Join(existingTopics, ", ")
	}

	prompt := fmt.Sprintf(`You are a Spanish curriculum designer for beginners.
The current learning plan already covers these topics: %s.
Suggest 3 to 5 NEW topics that logically follow and build on what the student has already learned.
Respond ONLY with a valid JSON array. Do not include any explanations, greetings, or thinking process. Do NOT output <think> blocks.
Format your output exactly like this:
[
  {"title": "Topic Name", "description": "A brief description of what this topic covers and why it follows logically."}
]
Respond ONLY with the JSON array.`, topicList)

	raw, err := c.generateRaw(prompt)
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

// generateRaw sends a prompt to Ollama and returns the raw concatenated text response.
func (c *Client) generateRaw(prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var genResp GenerateResponse
		if err := json.Unmarshal(scanner.Bytes(), &genResp); err != nil {
			continue
		}
		fullResponse.WriteString(genResp.Response)
		if genResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading ollama stream: %w", err)
	}

	raw := fullResponse.String()
	log.Print(raw)
	return raw, nil
}

func (c *Client) generateMulti(prompt string) ([]domain.Question, error) {
	raw, err := c.generateRaw(prompt)
	if err != nil {
		return nil, err
	}

	jsonContent := extractJSON(raw)

	var questions []domain.Question
	if err := json.Unmarshal([]byte(jsonContent), &questions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON array from llm response: %w", err)
	}

	for _, q := range questions {
		if len(q.Options) != 4 || q.Text == "" || q.CorrectAnswer == "" || q.TTSPhrase == "" {
			return nil, fmt.Errorf("invalid question format in array: %s", jsonContent)
		}
	}

	return questions, nil
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
