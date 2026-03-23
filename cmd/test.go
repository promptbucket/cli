package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/promptbucket/cli/internal/cloud"
	"github.com/promptbucket/cli/internal/config"
	"github.com/promptbucket/cli/internal/output"
	"github.com/promptbucket/cli/internal/runner"
	"github.com/spf13/cobra"
)

var (
	filterName string
	cloudMode  bool
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run prompt tests defined in your config file",
	Long:  "Execute all prompt tests, call LLM providers, run assertions, and report results.",
	RunE: func(cmd *cobra.Command, args []string) error {
		suite, err := config.Load(configFile)
		if err != nil {
			return err
		}

		// Apply name filter if set.
		if filterName != "" {
			var filtered []config.TestCase
			for _, tc := range suite.Tests {
				if strings.Contains(strings.ToLower(tc.Name), strings.ToLower(filterName)) {
					filtered = append(filtered, tc)
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("no tests match filter %q", filterName)
			}
			suite.Tests = filtered
		}

		fmt.Printf("Running %d test(s) with concurrency %d...\n", len(suite.Tests), concurrency)

		ctx := context.Background()
		results := runner.Run(ctx, suite, concurrency)

		output.PrintResults(results, ciMode)

		if cloudMode {
			key := apiKey
			if key == "" {
				key = os.Getenv("PROMPTBUCKET_API_KEY")
			}
			cloud.UploadResults(key, results)
		}

		if ciMode && (results.Failed > 0 || results.Errors > 0) {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	testCmd.Flags().StringVar(&filterName, "filter", "", "filter tests by name substring")
	testCmd.Flags().BoolVar(&cloudMode, "cloud", false, "upload test results to PromptBucket cloud")
	rootCmd.AddCommand(testCmd)
}
