package cmd

import (
	"fmt"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from PromptBucket",
	Long:  `Logout from PromptBucket by clearing stored authentication credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create auth config and token manager
		config := auth.NewConfig()
		tokenManager := auth.NewTokenManager(config)

		// Check if user is logged in
		if !tokenManager.IsAuthenticated() {
			fmt.Printf("⚠️  Not currently logged in\n")
			return nil
		}

		// Get current user info before clearing
		token, err := tokenManager.GetToken()
		if err != nil {
			return fmt.Errorf("error reading current token: %w", err)
		}

		// Clear the token
		if err := tokenManager.ClearToken(); err != nil {
			return fmt.Errorf("failed to clear auth token: %w", err)
		}

		fmt.Printf("✅ Successfully logged out\n")
		if token != nil && token.Email != "" {
			fmt.Printf("   Previously logged in as: %s\n", token.Email)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}