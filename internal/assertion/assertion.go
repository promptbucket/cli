package assertion

import (
	"fmt"

	"github.com/promptbucket/cli/internal/config"
	"github.com/promptbucket/cli/internal/provider"
)

// Result holds the outcome of a single assertion check.
type Result struct {
	Passed  bool
	Type    string
	Message string
}

// Asserter is the interface for all assertion types.
type Asserter interface {
	Assert(response string, cost float64) Result
}

// FromConfig creates an Asserter from a config Assertion definition.
// judgeProvider is used only for llm-judge assertions and may be nil otherwise.
func FromConfig(a config.Assertion, judgeProvider provider.Provider) (Asserter, error) {
	switch a.Type {
	case "contains":
		if a.Value == "" {
			return nil, fmt.Errorf("contains assertion requires a value")
		}
		return &Contains{Value: a.Value}, nil

	case "not-contains":
		if a.Value == "" {
			return nil, fmt.Errorf("not-contains assertion requires a value")
		}
		return &NotContains{Value: a.Value}, nil

	case "regex":
		if a.Pattern == "" {
			return nil, fmt.Errorf("regex assertion requires a pattern")
		}
		return NewRegex(a.Pattern)

	case "cost-below":
		if a.Max <= 0 {
			return nil, fmt.Errorf("cost-below assertion requires a positive max value")
		}
		return &CostBelow{Max: a.Max}, nil

	case "llm-judge":
		if a.Criteria == "" {
			return nil, fmt.Errorf("llm-judge assertion requires criteria")
		}
		if judgeProvider == nil {
			return nil, fmt.Errorf("llm-judge assertion requires a judge provider")
		}
		return &LLMJudge{
			Criteria: a.Criteria,
			Provider: judgeProvider,
		}, nil

	default:
		return nil, fmt.Errorf("unknown assertion type: %q", a.Type)
	}
}
