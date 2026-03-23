package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const starterYAML = `# PromptBucket Test Suite
# Docs: https://promptbucket.co/docs

tests:
  - name: "basic greeting"
    prompt: "Say hello in a friendly way"
    models:
      - gpt-4o-mini
      - claude-sonnet-4-20250514
    assert:
      - type: contains
        value: "hello"
      - type: cost-below
        max: 0.01

  - name: "code generation"
    prompt: "Write a Python function that checks if a number is prime"
    system: "You are a helpful coding assistant. Write clean, efficient code."
    models:
      - gpt-4o
    assert:
      - type: contains
        value: "def"
      - type: not-contains
        value: "TODO"
      - type: llm-judge
        criteria: "The code correctly implements a prime number checker"
`

var forceInit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter promptbucket.yaml config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := configFile

		if !forceInit {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			}
		}

		if err := os.WriteFile(path, []byte(starterYAML), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		fmt.Printf("Created %s\n", path)
		fmt.Println("Next: edit the file, then run `promptbucket test`")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&forceInit, "force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
}
