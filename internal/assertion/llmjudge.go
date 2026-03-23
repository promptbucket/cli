package assertion

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/promptbucket/cli/internal/provider"
)

// LLMJudge uses an LLM to evaluate whether a response meets given criteria.
type LLMJudge struct {
	Criteria string
	Provider provider.Provider
}

// DefaultJudgeModel is the model used for LLM-as-judge evaluations.
func DefaultJudgeModel() string {
	if m := os.Getenv("PROMPTBUCKET_JUDGE_MODEL"); m != "" {
		return m
	}
	return "gpt-4o-mini"
}

func (j *LLMJudge) Assert(response string, cost float64) Result {
	prompt := fmt.Sprintf(
		"You are an AI evaluation judge. Given the following AI response and evaluation criteria, "+
			"answer PASS or FAIL on the first line, then explain why on subsequent lines.\n\n"+
			"--- AI RESPONSE ---\n%s\n\n--- CRITERIA ---\n%s\n\n"+
			"Your verdict (first line must be exactly PASS or FAIL):",
		response,
		j.Criteria,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	judgeResp, err := j.Provider.Call(ctx, provider.Request{
		Model:  DefaultJudgeModel(),
		Prompt: prompt,
	})
	if err != nil {
		return Result{
			Passed:  false,
			Type:    "llm-judge",
			Message: fmt.Sprintf("judge call failed: %v", err),
		}
	}

	// Parse the first line for PASS/FAIL.
	firstLine := strings.TrimSpace(strings.SplitN(judgeResp.Content, "\n", 2)[0])
	firstLine = strings.ToUpper(firstLine)

	passed := strings.Contains(firstLine, "PASS")

	explanation := judgeResp.Content
	if parts := strings.SplitN(judgeResp.Content, "\n", 2); len(parts) > 1 {
		explanation = strings.TrimSpace(parts[1])
	}

	if passed {
		return Result{
			Passed:  true,
			Type:    "llm-judge",
			Message: fmt.Sprintf("judge: PASS — %s", truncate(explanation, 120)),
		}
	}

	return Result{
		Passed:  false,
		Type:    "llm-judge",
		Message: fmt.Sprintf("judge: FAIL — %s", truncate(explanation, 120)),
	}
}

func truncate(s string, max int) string {
	// Replace newlines with spaces for compact display.
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
