package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "promptbucket",
	Short: "PromptBucket CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("version", "v", false, "print version")
	viper.BindPFlag("version", rootCmd.PersistentFlags().Lookup("version"))
	
	// Load .env files before setting up viper
	loadEnvFiles()
	
	// Setup viper after .env files are loaded
	viper.AutomaticEnv()
	
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if viper.GetBool("version") {
			fmt.Println(version)
			os.Exit(0)
		}
	}
}

// loadEnvFiles loads environment variables from .env files in order of precedence
func loadEnvFiles() {
	envFiles := []string{
		".env",           // Base environment file
		".env.local",     // Local overrides (should be gitignored)
	}
	
	// Add environment-specific files if ENVIRONMENT is set
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		envFiles = append(envFiles, fmt.Sprintf(".env.%s", env))
		envFiles = append(envFiles, fmt.Sprintf(".env.%s.local", env))
	}
	
	// Load files in reverse order so later files can override earlier ones
	for i := len(envFiles) - 1; i >= 0; i-- {
		envFile := envFiles[i]
		if _, err := os.Stat(envFile); err == nil {
			if err := gotenv.OverLoad(envFile); err == nil {
				// Only log successful loads in debug mode to avoid noise
				if os.Getenv("DEBUG") != "" {
					fmt.Fprintf(os.Stderr, "Loaded environment from %s\n", envFile)
				}
			}
		}
	}
}
