package assertion

import "fmt"

// CostBelow asserts that the response cost is below a maximum threshold.
type CostBelow struct {
	Max float64
}

func (c *CostBelow) Assert(response string, cost float64) Result {
	if cost <= c.Max {
		return Result{
			Passed:  true,
			Type:    "cost-below",
			Message: fmt.Sprintf("cost $%.6f is below max $%.4f", cost, c.Max),
		}
	}

	return Result{
		Passed:  false,
		Type:    "cost-below",
		Message: fmt.Sprintf("cost $%.6f exceeds max $%.4f", cost, c.Max),
	}
}
