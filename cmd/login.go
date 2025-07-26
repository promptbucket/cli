package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/browser"
	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to PromptBucket using your browser",
	Long: `Login to PromptBucket using OAuth authentication through your browser.
This will open your default browser to authenticate with Google or GitHub,
then save your credentials locally for future CLI use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		port, _ := cmd.Flags().GetInt("port")
		
		// Validate provider
		if provider != "google" && provider != "github" {
			return fmt.Errorf("invalid provider '%s'. Use 'google' or 'github'", provider)
		}

		// Create auth config
		config := auth.NewConfig()
		if port != 0 {
			config.CallbackPort = port
		}

		// Create token manager
		tokenManager := auth.NewTokenManager(config)

		// Check if already logged in
		if tokenManager.IsAuthenticated() {
			token, err := tokenManager.GetToken()
			if err != nil {
				return fmt.Errorf("error reading existing token: %w", err)
			}
			fmt.Printf("✅ Already logged in as %s\n", token.Email)
			if token.Name != "" {
				fmt.Printf("   Name: %s\n", token.Name)
			}
			if token.Provider != "" {
				fmt.Printf("   Provider: %s\n", token.Provider)
			}
			return nil
		}

		fmt.Printf("🔐 Starting PromptBucket authentication...\n")
		fmt.Printf("   Provider: %s\n", provider)
		fmt.Printf("   Callback port: %d\n", config.CallbackPort)

		// Create OAuth server
		oauthServer := auth.NewOAuthServer(config)
		
		// Check if port is available
		if !oauthServer.IsPortAvailable() {
			return fmt.Errorf("callback port %d is not available. Try a different port with --port flag", config.CallbackPort)
		}

		// Start OAuth server with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		fmt.Printf("⏳ Starting local callback server...\n")
		
		// Start server in goroutine
		resultCh := make(chan *auth.OAuthResult, 1)
		go func() {
			result, err := oauthServer.Start(ctx)
			if err != nil {
				resultCh <- &auth.OAuthResult{Success: false, Error: err.Error()}
			} else {
				resultCh <- result
			}
		}()

		// Small delay to ensure server is started
		time.Sleep(100 * time.Millisecond)

		// Open browser to OAuth URL
		oauthURL := config.GetOAuthURL(provider)
		fmt.Printf("🌐 Opening browser to %s login...\n", provider)
		fmt.Printf("🔗 OAuth URL: %s\n", oauthURL)
		
		if err := browser.OpenURL(oauthURL); err != nil {
			fmt.Printf("⚠️  Could not open browser automatically. Please visit:\n")
			fmt.Printf("   %s\n", oauthURL)
		}

		fmt.Printf("⏳ Waiting for authentication in browser...\n")

		// Wait for result
		select {
		case result := <-resultCh:
			if result.Success && result.Token != nil {
				// Save token
				if err := tokenManager.SaveToken(result.Token); err != nil {
					return fmt.Errorf("failed to save auth token: %w", err)
				}

				fmt.Printf("✅ Authentication successful!\n")
				fmt.Printf("   Logged in as: %s\n", result.Token.Email)
				if result.Token.Name != "" {
					fmt.Printf("   Name: %s\n", result.Token.Name)
				}
				if result.Token.Provider != "" {
					fmt.Printf("   Provider: %s\n", result.Token.Provider)
				}
				return nil
			} else {
				return fmt.Errorf("authentication failed: %s", result.Error)
			}
		case <-ctx.Done():
			oauthServer.Stop()
			return fmt.Errorf("authentication timeout - no response received within 5 minutes")
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringP("provider", "p", "google", "OAuth provider (google|github)")
	loginCmd.Flags().IntP("port", "", 3456, "Local callback server port")
}