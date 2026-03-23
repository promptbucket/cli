package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/promptbucket/cli/internal/assertion"
	"github.com/promptbucket/cli/internal/config"
	"github.com/promptbucket/cli/internal/provider"
)

// TestResult holds the outcome for one test case + model combination.
type TestResult struct {
	TestName   string
	Model      string
	Response   *provider.Response
	Assertions []assertion.Result
	Passed     bool
	Error      error
}

// SuiteResult holds the aggregate results for all tests.
type SuiteResult struct {
	Results   []TestResult
	Total     int
	Passed    int
	Failed    int
	Errors    int
	TotalCost float64
	Duration  time.Duration
}

// Run executes all tests in the suite with the given concurrency limit.
func Run(ctx context.Context, suite *config.TestSuite, concurrency int) *SuiteResult {
	start := time.Now()

	// Set up a judge provider for llm-judge assertions.
	judgeModel := assertion.DefaultJudgeModel()
	judgeProvider, judgeErr := provider.ResolveProvider(judgeModel)

	type job struct {
		tc    config.TestCase
		model string
	}

	// Build the list of jobs (test x model).
	var jobs []job
	for _, tc := range suite.Tests {
		for _, model := range tc.Models {
			jobs = append(jobs, job{tc: tc, model: model})
		}
	}

	results := make([]TestResult, len(jobs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, j job) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			results[idx] = runSingle(ctx, j.tc, j.model, judgeProvider, judgeErr)
		}(i, j)
	}

	wg.Wait()

	// Aggregate results.
	sr := &SuiteResult{
		Results:  results,
		Total:    len(results),
		Duration: time.Since(start),
	}

	for _, r := range results {
		if r.Error != nil {
			sr.Errors++
			continue
		}
		if r.Response != nil {
			sr.TotalCost += r.Response.Cost
		}
		if r.Passed {
			sr.Passed++
		} else {
			sr.Failed++
		}
	}

	return sr
}

func runSingle(ctx context.Context, tc config.TestCase, model string, judgeProvider provider.Provider, judgeErr error) TestResult {
	result := TestResult{
		TestName: tc.Name,
		Model:    model,
	}

	// Resolve system prompt (may be a file path).
	system, err := tc.ResolveSystem()
	if err != nil {
		result.Error = fmt.Errorf("resolve system prompt: %w", err)
		return result
	}

	// Resolve the provider for this model.
	p, err := provider.ResolveProvider(model)
	if err != nil {
		result.Error = err
		return result
	}

	// Call the provider.
	resp, err := p.Call(ctx, provider.Request{
		Model:  model,
		System: system,
		Prompt: tc.Prompt,
	})
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = resp
	result.Passed = true

	// Run each assertion.
	for _, assertCfg := range tc.Assert {
		// Determine the judge provider for this assertion.
		var jp provider.Provider
		if assertCfg.Type == "llm-judge" {
			if judgeErr != nil {
				result.Assertions = append(result.Assertions, assertion.Result{
					Passed:  false,
					Type:    "llm-judge",
					Message: fmt.Sprintf("cannot resolve judge provider: %v", judgeErr),
				})
				result.Passed = false
				continue
			}
			jp = judgeProvider
		}

		asserter, err := assertion.FromConfig(assertCfg, jp)
		if err != nil {
			result.Assertions = append(result.Assertions, assertion.Result{
				Passed:  false,
				Type:    assertCfg.Type,
				Message: fmt.Sprintf("config error: %v", err),
			})
			result.Passed = false
			continue
		}

		ar := asserter.Assert(resp.Content, resp.Cost)
		result.Assertions = append(result.Assertions, ar)
		if !ar.Passed {
			result.Passed = false
		}
	}

	return result
}
