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

// Google implements Provider for Google Gemini models.
type Google struct {
	apiKey string
	client *http.Client
}

// NewGoogle creates a new Google Gemini provider.
func NewGoogle() *Google {
	return &Google{
		apiKey: os.Getenv("GOOGLE_API_KEY"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (g *Google) Name() string { return "google" }

// google cost table: (input $/1M tokens, output $/1M tokens)
var googleCosts = map[string][2]float64{
	"gemini-2.5-pro":       {1.25, 10.00},
	"gemini-2.5-flash":     {0.15, 0.60},
	"gemini-2.0-flash":     {0.10, 0.40},
	"gemini-2.0-flash-lite": {0.075, 0.30},
	"gemini-1.5-pro":       {1.25, 5.00},
	"gemini-1.5-flash":     {0.075, 0.30},
}

type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (g *Google) Call(ctx context.Context, req Request) (*Response, error) {
	if g.apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is not set")
	}

	body := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: req.Prompt}},
				Role:  "user",
			},
		},
	}

	if req.System != "" {
		body.SystemInstruction = &geminiSystemInstruction{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", req.Model, g.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	httpResp, err := g.client.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("google request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google returned %d: %s", httpResp.StatusCode, string(respBody))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if gemResp.Error != nil {
		return nil, fmt.Errorf("google error: %s", gemResp.Error.Message)
	}

	if len(gemResp.Candidates) == 0 {
		return nil, fmt.Errorf("google returned no candidates")
	}

	var content string
	for _, part := range gemResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	cost := computeGoogleCost(req.Model, gemResp.UsageMetadata.PromptTokenCount, gemResp.UsageMetadata.CandidatesTokenCount)

	return &Response{
		Content:      content,
		Model:        req.Model,
		InputTokens:  gemResp.UsageMetadata.PromptTokenCount,
		OutputTokens: gemResp.UsageMetadata.CandidatesTokenCount,
		LatencyMs:    latency,
		Cost:         cost,
	}, nil
}

func computeGoogleCost(model string, inputTokens, outputTokens int) float64 {
	rates, ok := googleCosts[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
