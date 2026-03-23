package assertion

import (
	"fmt"
	"strings"
)

// Contains asserts that the response contains a given substring (case-insensitive).
type Contains struct {
	Value string
}

func (c *Contains) Assert(response string, cost float64) Result {
	lower := strings.ToLower(response)
	target := strings.ToLower(c.Value)

	if strings.Contains(lower, target) {
		return Result{
			Passed:  true,
			Type:    "contains",
			Message: fmt.Sprintf("response contains %q", c.Value),
		}
	}

	return Result{
		Passed:  false,
		Type:    "contains",
		Message: fmt.Sprintf("response does not contain %q", c.Value),
	}
}

// NotContains asserts that the response does NOT contain a given substring (case-insensitive).
type NotContains struct {
	Value string
}

func (n *NotContains) Assert(response string, cost float64) Result {
	lower := strings.ToLower(response)
	target := strings.ToLower(n.Value)

	if !strings.Contains(lower, target) {
		return Result{
			Passed:  true,
			Type:    "not-contains",
			Message: fmt.Sprintf("response does not contain %q", n.Value),
		}
	}

	return Result{
		Passed:  false,
		Type:    "not-contains",
		Message: fmt.Sprintf("response contains %q (should not)", n.Value),
	}
}
