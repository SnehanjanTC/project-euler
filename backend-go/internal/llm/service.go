package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	BaseURL string
	Model   string
}

type Service struct {
	config Config
	client *http.Client
}

func NewService(baseURL, model string) *Service {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen3-vl:2b" // Default model matches Python config
	}
	return &Service{
		config: Config{
			BaseURL: baseURL,
			Model:   model,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
}

// CallOllama calls the Ollama API
func (s *Service) CallOllama(prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  s.config.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Post(s.config.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", err
	}

	return genResp.Response, nil
}

type Match struct {
	ColA       string  `json:"col_a"`
	ColB       string  `json:"col_b"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

type MatchesResponse struct {
	Matches []Match `json:"matches"`
}

// GetSemanticMatches asks the LLM to match columns
func (s *Service) GetSemanticMatches(cols1, cols2 []string) ([]Match, error) {
	prompt := fmt.Sprintf(`
You are an expert data integration specialist. Match columns from List A to List B based on semantic meaning.

List A: %s
List B: %s

Return a JSON object where keys are columns from List A and values are the best matching column from List B.
Only include matches where you are confident (score > 0.5).

Format:
{
	"matches": [
		{"col_a": "column_from_list_a", "col_b": "column_from_list_b", "confidence": 0.9, "reason": "Both refer to..."}
	]
}

Return ONLY the JSON.
`, strings.Join(cols1, ", "), strings.Join(cols2, ", "))

	response, err := s.CallOllama(prompt)
	if err != nil {
		return nil, err
	}

	// Extract JSON
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonStr := jsonRegex.FindString(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var matchesResp MatchesResponse
	if err := json.Unmarshal([]byte(jsonStr), &matchesResp); err != nil {
		return nil, err
	}

	return matchesResp.Matches, nil
}
