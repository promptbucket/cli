package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var testAuthCmd = &cobra.Command{
	Use:   "test-auth",
	Short: "Test authentication with PromptBucket API",
	Long:  `Test authentication by making a health check request to the PromptBucket API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Check if authenticated
		if !apiClient.IsAuthenticated() {
			fmt.Printf("❌ Not authenticated. Run 'promptbucket login' first.\n")
			return nil
		}

		// Get current user info
		user, err := apiClient.GetCurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		fmt.Printf("🔐 Testing authentication...\n")
		fmt.Printf("   User: %s\n", user.Email)
		fmt.Printf("   API: %s\n", config.GetAPIURL(""))

		// Make health check request
		resp, err := apiClient.Get("/healthz")
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status
		if resp.StatusCode != 200 {
			fmt.Printf("❌ API request failed with status %d\n", resp.StatusCode)
			fmt.Printf("   Response: %s\n", string(body))
			return nil
		}

		// Parse JSON response
		var healthCheck map[string]interface{}
		if err := json.Unmarshal(body, &healthCheck); err != nil {
			fmt.Printf("⚠️  API responded but couldn't parse JSON: %v\n", err)
			fmt.Printf("   Raw response: %s\n", string(body))
			return nil
		}

		fmt.Printf("✅ Authentication working!\n")
		fmt.Printf("   API Health Check:\n")
		
		// Pretty print health check response
		for key, value := range healthCheck {
			fmt.Printf("     %s: %v\n", key, value)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testAuthCmd)
}