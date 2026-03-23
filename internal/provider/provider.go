package provider

import (
	"context"
	"fmt"
	"strings"
)

// Request is the input to a model provider call.
type Request struct {
	Model  string
	System string
	Prompt string
}

// Response is the output from a model provider call.
type Response struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
	Cost         float64
}

// Provider is the interface each LLM backend implements.
type Provider interface {
	Name() string
	Call(ctx context.Context, req Request) (*Response, error)
}

// ResolveProvider returns the appropriate provider for a given model name.
func ResolveProvider(model string) (Provider, error) {
	switch {
	case strings.HasPrefix(model, "gpt-") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4"):
		return NewOpenAI(), nil

	case strings.HasPrefix(model, "claude-"):
		return NewAnthropic(), nil

	case strings.HasPrefix(model, "gemini-"):
		return NewGoogle(), nil

	default:
		return nil, fmt.Errorf("unknown model prefix: %q — supported prefixes: gpt-, o1, o3, o4, claude-, gemini-", model)
	}
}
