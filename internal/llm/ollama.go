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

	"github.com/mats/telegram-quiz-bot/internal/db"
)

type Client struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// GenerateRequest represents the body sent to Ollama
type GenerateRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Stream   bool   `json:"stream"`
	Thinking bool   `json:"thinking"`
}

// GenerateResponse represents the response from Ollama
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
		model = "qwen3.5:4b"
	}
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second, // Generation can take time
		},
	}
}

// GenerateSpanishQuiz asks the LLM to generate a random basic Spanish quiz natively in JSON
func (c *Client) GenerateSpanishQuiz(topic string, excludeQuestions []string) (*db.Quiz, error) {
	prompt := fmt.Sprintf(`You are an expert Spanish teacher. Generate a basic Spanish quiz question for beginners.
Respond exactly with a JSON block.
{
  "topic": "%s",
  "question": "What is the Spanish word for 'Apple'?",
  "options": ["Manzana", "Naranja", "Plátano", "Pera"],
  "correct_answer": "Manzana"
}
Ensure there are exactly 4 options. The quiz question MUST be about the given topic: "%s".
Keep the vocabulary simple.`, topic, topic)

	if len(excludeQuestions) > 0 {
		prompt += "\nDo NOT use these questions again:\n"
		for _, q := range excludeQuestions {
			prompt += fmt.Sprintf("- %s\n", q)
		}
	}
	prompt += "\nRespond ONLY with the JSON block."

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("[LLM] Attempt %d: Generating quiz for topic: %s\n", attempt, topic)
		fmt.Printf("[LLM] Prompt:\n%s\n", prompt)

		quiz, err := c.generate(prompt)
		if err == nil {
			fmt.Printf("[LLM] Successfully generated quiz on attempt %d\n", attempt)
			return quiz, nil
		}

		lastErr = err
		fmt.Printf("[LLM] Attempt %d failed: %v\n", attempt, err)
		time.Sleep(500 * time.Millisecond) // Short backoff
	}

	return nil, fmt.Errorf("failed to generate quiz after 3 attempts: %w", lastErr)
}

// GenerateSpanishQuizzes asks the LLM to generate multiple Spanish quiz questions
func (c *Client) GenerateSpanishQuizzes(topic string, excludeQuestions []string, count int) ([]db.Quiz, error) {
	prompt := fmt.Sprintf(`You are an expert Spanish teacher. Generate %d basic Spanish quiz questions for beginners.
Respond with a json block.
[
  {
    "topic": "%s",
    "question": "Question?",
    "options": ["Option 1", "Option 2", "Option 3", "Option 4"],
    "correct_answer": "Option 1"
  }
]
Ensure each quiz has exactly 4 options. All quiz questions MUST be about the given topic: "%s".
Keep the vocabulary simple.`, count, topic, topic)

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
		fmt.Printf("[LLM] Attempt %d: Generating %d quizzes for topic: %s\n", attempt, count, topic)

		quizzes, err := c.generateMulti(prompt)
		if err == nil {
			fmt.Printf("[LLM] Successfully generated %d quizzes on attempt %d\n", len(quizzes), attempt)
			return quizzes, nil
		}

		lastErr = err
		fmt.Printf("[LLM] Attempt %d failed: %v\n", attempt, err)
		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("failed to generate quizzes after 3 attempts: %w", lastErr)
}

func (c *Client) generate(prompt string) (*db.Quiz, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var genResp GenerateResponse
		if err := json.Unmarshal(scanner.Bytes(), &genResp); err != nil {
			continue // Skip unparseable lines
		}
		fullResponse.WriteString(genResp.Response)
		if genResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ollama stream: %w", err)
	}

	rawContent := fullResponse.String()
	fmt.Printf("[LLM] Raw response: %s\n", rawContent)

	jsonContent := extractJSON(rawContent)
	fmt.Printf("[LLM] Extracted JSON: %s\n", jsonContent)

	var quiz db.Quiz
	if err := json.Unmarshal([]byte(jsonContent), &quiz); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from llm response (extracted '%s' from raw '%s'): %w", jsonContent, rawContent, err)
	}

	// Validate quiz
	if len(quiz.Options) != 4 || quiz.Question == "" || quiz.CorrectAnswer == "" {
		return nil, fmt.Errorf("invalid quiz format (missing fields or wrong option count): %s", jsonContent)
	}

	return &quiz, nil
}

func (c *Client) generateMulti(prompt string) ([]db.Quiz, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
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
		return nil, fmt.Errorf("error reading ollama stream: %w", err)
	}

	rawContent := fullResponse.String()
	log.Print(rawContent)
	jsonContent := extractJSON(rawContent)

	var quizzes []db.Quiz
	if err := json.Unmarshal([]byte(jsonContent), &quizzes); err != nil {
		return nil, fmt.Errorf("failed to parse JSON array from llm response: %w", err)
	}

	for _, q := range quizzes {
		if len(q.Options) != 4 || q.Question == "" || q.CorrectAnswer == "" {
			return nil, fmt.Errorf("invalid quiz format in array: %s", jsonContent)
		}
	}

	return quizzes, nil
}

func extractJSON(s string) string {
	// Try to find content between ```json and ```
	if start := strings.Index(s, "```json"); start != -1 {
		inner := s[start+7:]
		if end := strings.Index(inner, "```"); end != -1 {
			return strings.TrimSpace(inner[:end])
		}
	}
	// Fallback: Content between ``` and ```
	if start := strings.Index(s, "```"); start != -1 {
		inner := s[start+3:]
		if end := strings.Index(inner, "```"); end != -1 {
			return strings.TrimSpace(inner[:end])
		}
	}

	// Final fallback: Look for something that looks like a JSON object or array
	startObj := strings.Index(s, "{")
	endObj := strings.LastIndex(s, "}")
	startArr := strings.Index(s, "[")
	endArr := strings.LastIndex(s, "]")

	// If it looks like an array and either starts before an object or there is no object
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
