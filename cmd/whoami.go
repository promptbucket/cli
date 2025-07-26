package cmd

import (
	"fmt"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	Long:  `Display information about the currently authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create auth config and token manager
		config := auth.NewConfig()
		tokenManager := auth.NewTokenManager(config)

		// Check if user is logged in
		if !tokenManager.IsAuthenticated() {
			fmt.Printf("⚠️  Not logged in\n")
			fmt.Printf("   Run 'promptbucket login' to authenticate\n")
			return nil
		}

		// Get current user info
		token, err := tokenManager.GetToken()
		if err != nil {
			return fmt.Errorf("error reading auth token: %w", err)
		}

		if token == nil {
			fmt.Printf("❌ Authentication token not found\n")
			return nil
		}

		// Display user information
		fmt.Printf("👤 Current User:\n")
		fmt.Printf("   Email: %s\n", token.Email)
		fmt.Printf("   User ID: %s\n", token.UserID)
		
		if token.Name != "" {
			fmt.Printf("   Name: %s\n", token.Name)
		}
		
		if token.Provider != "" {
			fmt.Printf("   Provider: %s\n", token.Provider)
		}

		if token.ExpiresAt != nil {
			fmt.Printf("   Token expires: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05 MST"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}