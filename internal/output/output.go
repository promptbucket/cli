package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/promptbucket/cli/internal/runner"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func useColor(ci bool) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if ci {
		return false
	}
	return true
}

// PrintResults formats and prints the test suite results.
func PrintResults(results *runner.SuiteResult, ci bool) {
	color := useColor(ci)

	fmt.Println()
	if color {
		fmt.Printf("%s%s PromptBucket Test Results %s\n", colorBold, colorCyan, colorReset)
	} else {
		fmt.Println("PromptBucket Test Results")
	}
	fmt.Println(strings.Repeat("-", 50))

	for _, r := range results.Results {
		printTestResult(r, color)
	}

	// Summary line.
	fmt.Println(strings.Repeat("-", 50))

	passedStr := fmt.Sprintf("%d/%d passed", results.Passed, results.Total)
	failedStr := ""
	errorStr := ""

	if results.Failed > 0 {
		failedStr = fmt.Sprintf(" | %d failed", results.Failed)
	}
	if results.Errors > 0 {
		errorStr = fmt.Sprintf(" | %d errors", results.Errors)
	}

	costStr := fmt.Sprintf("$%.4f total cost", results.TotalCost)
	durationStr := fmt.Sprintf("%.1fs", results.Duration.Seconds())

	if color {
		statusColor := colorGreen
		if results.Failed > 0 || results.Errors > 0 {
			statusColor = colorRed
		}
		fmt.Printf("%s%s%s%s%s | %s | %s\n",
			statusColor, passedStr, colorReset,
			failedStr, errorStr,
			costStr, durationStr,
		)
	} else {
		fmt.Printf("%s%s%s | %s | %s\n",
			passedStr, failedStr, errorStr,
			costStr, durationStr,
		)
	}
	fmt.Println()
}

func printTestResult(r runner.TestResult, color bool) {
	if r.Error != nil {
		if color {
			fmt.Printf("  %s\u26a0 %s [%s]%s\n", colorYellow, r.TestName, r.Model, colorReset)
			fmt.Printf("    %serror: %v%s\n", colorDim, r.Error, colorReset)
		} else {
			fmt.Printf("  ! %s [%s]\n", r.TestName, r.Model)
			fmt.Printf("    error: %v\n", r.Error)
		}
		return
	}

	if r.Passed {
		if color {
			fmt.Printf("  %s\u2713 %s [%s]%s", colorGreen, r.TestName, r.Model, colorReset)
		} else {
			fmt.Printf("  PASS %s [%s]", r.TestName, r.Model)
		}
	} else {
		if color {
			fmt.Printf("  %s\u2717 %s [%s]%s", colorRed, r.TestName, r.Model, colorReset)
		} else {
			fmt.Printf("  FAIL %s [%s]", r.TestName, r.Model)
		}
	}

	// Print latency and cost inline.
	if r.Response != nil {
		if color {
			fmt.Printf(" %s(%dms, $%.6f)%s", colorDim, r.Response.LatencyMs, r.Response.Cost, colorReset)
		} else {
			fmt.Printf(" (%dms, $%.6f)", r.Response.LatencyMs, r.Response.Cost)
		}
	}
	fmt.Println()

	// Print individual assertion results.
	for _, a := range r.Assertions {
		if a.Passed {
			if color {
				fmt.Printf("    %s\u2713 [%s] %s%s\n", colorGreen, a.Type, a.Message, colorReset)
			} else {
				fmt.Printf("    PASS [%s] %s\n", a.Type, a.Message)
			}
		} else {
			if color {
				fmt.Printf("    %s\u2717 [%s] %s%s\n", colorRed, a.Type, a.Message, colorReset)
			} else {
				fmt.Printf("    FAIL [%s] %s\n", a.Type, a.Message)
			}
		}
	}
}
