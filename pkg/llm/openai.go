package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kazuto/Weave/pkg/config"
)

type OpenAIClient struct {
	config config.OpenAIConfig
	client *http.Client
}

type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	TopP        float64         `json:"top_p"`
	Stream      bool            `json:"stream"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatResponse struct {
	Choices []struct {
		Message openaiMessage `json:"message"`
	} `json:"choices"`
}

type openaiModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func NewOpenAIClient(cfg config.OpenAIConfig) *OpenAIClient {
	return &OpenAIClient{
		config: cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAIClient) CheckConnection() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/models", c.config.Host), nil)
	if err != nil {
		return false
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

func (c *OpenAIClient) IsModelAvailable() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/models", c.config.Host), nil)
	if err != nil {
		return false
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	var modelsResp openaiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return false
	}

	for _, m := range modelsResp.Data {
		if strings.Contains(m.ID, c.config.Model) {
			return true
		}
	}
	return false
}

func (c *OpenAIClient) Generate(prompt string) (string, error) {
	reqBody := openaiChatRequest{
		Model: c.config.Model,
		Messages: []openaiMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.config.Temperature,
		TopP:        c.config.TopP,
		Stream:      false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/chat/completions", c.config.Host), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
