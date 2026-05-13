package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AnthropicClient struct {
	APIKey  string
	BaseURL string
}

// NewAnthropicClient creates a client using the default Anthropic API endpoint.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		APIKey:  apiKey,
		BaseURL: "https://api.anthropic.com",
	}
}

// NewAnthropicClientWithBaseURL creates a client with a custom base URL,
// enabling alternative providers or local proxies that implement the
// Anthropic Messages API (e.g. LiteLLM, OpenRouter, local Claude proxies).
// Falls back to the default endpoint if baseURL is empty.
func NewAnthropicClientWithBaseURL(apiKey, baseURL string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

type MessageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *AnthropicClient) GenerateScript(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := MessageRequest{
		Model:     "claude-3-5-sonnet-20240620",
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic api error: %s", string(body))
	}

	var msgResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return "", err
	}

	if len(msgResp.Content) == 0 {
		return "", fmt.Errorf("empty response from anthropic")
	}

	return msgResp.Content[0].Text, nil
}
