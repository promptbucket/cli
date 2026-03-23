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

// OpenAI implements Provider for OpenAI models.
type OpenAI struct {
	apiKey string
	client *http.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI() *OpenAI {
	return &OpenAI{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (o *OpenAI) Name() string { return "openai" }

// openAI cost table: (input $/1M tokens, output $/1M tokens)
var openAICosts = map[string][2]float64{
	"gpt-4o":        {2.50, 10.00},
	"gpt-4o-mini":   {0.15, 0.60},
	"gpt-4.1":       {2.00, 8.00},
	"gpt-4.1-mini":  {0.40, 1.60},
	"gpt-4.1-nano":  {0.10, 0.40},
	"gpt-4-turbo":   {10.00, 30.00},
	"gpt-4":         {30.00, 60.00},
	"gpt-3.5-turbo": {0.50, 1.50},
	"o1":            {15.00, 60.00},
	"o1-mini":       {3.00, 12.00},
	"o3":            {10.00, 40.00},
	"o3-mini":       {1.10, 4.40},
	"o4-mini":       {1.10, 4.40},
}

type openAIRequest struct {
	Model    string             `json:"model"`
	Messages []openAIMessage    `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAI) Call(ctx context.Context, req Request) (*Response, error) {
	if o.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	messages := make([]openAIMessage, 0, 2)
	if req.System != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: req.Prompt})

	body := openAIRequest{
		Model:    req.Model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	start := time.Now()
	httpResp, err := o.client.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned %d: %s", httpResp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if oaiResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", oaiResp.Error.Message)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	cost := computeOpenAICost(req.Model, oaiResp.Usage.PromptTokens, oaiResp.Usage.CompletionTokens)

	return &Response{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        req.Model,
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
		LatencyMs:    latency,
		Cost:         cost,
	}, nil
}

func computeOpenAICost(model string, inputTokens, outputTokens int) float64 {
	rates, ok := openAICosts[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
