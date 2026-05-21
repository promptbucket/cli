package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"
)

// Persistent flag values accessible by subcommands.
var (
	ciMode      bool
	concurrency int
	apiKey      string
)

var rootCmd = &cobra.Command{
	Use:   "promptbucket",
	Short: "Stop re-explaining your project to AI every day",
	Long: `PromptBucket — persistent AI personas for developers.

Run 'promptbucket init' to scaffold a persona in your project.
Run 'promptbucket serve' to start an MCP server for Claude Code, Cursor, or Cline.

Your persona's context stays with your project. Your AI remembers.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Load .env files before anything else.
	loadEnvFiles()

	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "CI mode — exit 1 on any failure")
	rootCmd.PersistentFlags().IntVar(&concurrency, "concurrency", 4, "max concurrent provider calls")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "PromptBucket API key (env PROMPTBUCKET_API_KEY)")
}

// loadEnvFiles loads environment variables from .env files in order of precedence.
func loadEnvFiles() {
	envFiles := []string{
		".env",
		".env.local",
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		envFiles = append(envFiles, fmt.Sprintf(".env.%s", env))
		envFiles = append(envFiles, fmt.Sprintf(".env.%s.local", env))
	}

	// Load files in reverse order so later files can override earlier ones.
	for i := len(envFiles) - 1; i >= 0; i-- {
		envFile := envFiles[i]
		if _, err := os.Stat(envFile); err == nil {
			if err := gotenv.OverLoad(envFile); err == nil {
				if os.Getenv("DEBUG") != "" {
					fmt.Fprintf(os.Stderr, "Loaded environment from %s\n", envFile)
				}
			}
		}
	}
}
