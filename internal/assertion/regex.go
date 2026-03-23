package assertion

import (
	"fmt"
	"regexp"
)

// Regex asserts that the response matches a regular expression pattern.
type Regex struct {
	Pattern string
	re      *regexp.Regexp
}

// NewRegex creates a Regex asserter, compiling the pattern.
func NewRegex(pattern string) (*Regex, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	return &Regex{Pattern: pattern, re: re}, nil
}

func (r *Regex) Assert(response string, cost float64) Result {
	if r.re.MatchString(response) {
		return Result{
			Passed:  true,
			Type:    "regex",
			Message: fmt.Sprintf("response matches pattern %q", r.Pattern),
		}
	}

	return Result{
		Passed:  false,
		Type:    "regex",
		Message: fmt.Sprintf("response does not match pattern %q", r.Pattern),
	}
}
