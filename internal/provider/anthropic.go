package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Anthropic implements Provider for Anthropic Claude models.
type Anthropic struct {
	apiKey string
	client *http.Client
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic() *Anthropic {
	return &Anthropic{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

// anthropic cost table: (input $/1M tokens, output $/1M tokens)
var anthropicCosts = map[string][2]float64{
	"claude-sonnet-4-20250514":      {3.00, 15.00},
	"claude-haiku-4-5-20251001":     {1.00, 5.00},
	"claude-3-5-sonnet-20241022":    {3.00, 15.00},
	"claude-3-5-haiku-20241022":     {0.80, 4.00},
	"claude-3-opus-20240229":        {15.00, 75.00},
	"claude-3-sonnet-20240229":      {3.00, 15.00},
	"claude-3-haiku-20240307":       {0.25, 1.25},
	"claude-opus-4-20250514":        {15.00, 75.00},
}

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) Call(ctx context.Context, req Request) (*Response, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
	}

	body := anthropicRequest{
		Model:     req.Model,
		MaxTokens: 4096,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.Prompt},
		},
	}

	// System prompt goes as top-level field, not in messages array.
	if req.System != "" {
		body.System = req.System
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	httpResp, err := a.client.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned %d: %s", httpResp.StatusCode, string(respBody))
	}

	var antResp anthropicResponse
	if err := json.Unmarshal(respBody, &antResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if antResp.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", antResp.Error.Message)
	}

	// Concatenate all text content blocks.
	var content string
	for _, block := range antResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	cost := computeAnthropicCost(req.Model, antResp.Usage.InputTokens, antResp.Usage.OutputTokens)

	return &Response{
		Content:      content,
		Model:        req.Model,
		InputTokens:  antResp.Usage.InputTokens,
		OutputTokens: antResp.Usage.OutputTokens,
		LatencyMs:    latency,
		Cost:         cost,
	}, nil
}

func computeAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	rates, ok := anthropicCosts[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
