package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:   "star [package]",
	Short: "Star a package",
	Long: `Star a package to show your appreciation and bookmark it.

Package can be specified as:
  - org/name (e.g., rawte.mayur/Ui-Artist)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleStar(args[0], true)
	},
}

var unstarCmd = &cobra.Command{
	Use:   "unstar [package]",
	Short: "Unstar a package",
	Long: `Remove a star from a package.

Package can be specified as:
  - org/name (e.g., rawte.mayur/Ui-Artist)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleStar(args[0], false)
	},
}

func handleStar(packageSpec string, star bool) error {
	// Parse package specification
	parts := strings.Split(packageSpec, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid package format: %s (expected org/name)", packageSpec)
	}
	org := parts[0]
	packageName := parts[1]

	// Create auth config and API client
	config := auth.NewConfig()
	apiClient := auth.NewAPIClient(config)

	// Check if authenticated
	if !apiClient.IsAuthenticated() {
		return fmt.Errorf("authentication required. Run 'promptbucket login' first")
	}

	// Build endpoint
	endpoint := fmt.Sprintf("/packages/%s/%s/star", org, packageName)

	// Make request
	var resp *http.Response
	var err error
	
	if star {
		fmt.Printf("⭐ Starring %s/%s...\n", org, packageName)
		resp, err = apiClient.Post(endpoint, nil)
	} else {
		fmt.Printf("💫 Unstarring %s/%s...\n", org, packageName)
		resp, err = apiClient.Delete(endpoint)
	}
	
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode == 404 {
		return fmt.Errorf("package not found: %s/%s", org, packageName)
	}
	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication required. Run 'promptbucket login' first")
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if msg, ok := errorResp["message"].(string); ok {
				return fmt.Errorf("%s", msg)
			}
			if errMsg, ok := errorResp["error"].(string); ok {
				return fmt.Errorf("%s", errMsg)
			}
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Success
	if star {
		fmt.Printf("✅ Successfully starred %s/%s\n", org, packageName)
	} else {
		fmt.Printf("✅ Successfully unstarred %s/%s\n", org, packageName)
	}

	return nil
}

var starsCmd = &cobra.Command{
	Use:   "stars",
	Short: "List your starred packages",
	Long:  `List all packages you have starred.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Check if authenticated
		if !apiClient.IsAuthenticated() {
			return fmt.Errorf("authentication required. Run 'promptbucket login' first")
		}

		// Get starred packages
		fmt.Println("⭐ Fetching your starred packages...")
		resp, err := apiClient.Get("/user/starred")
		if err != nil {
			return fmt.Errorf("failed to fetch starred packages: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status
		if resp.StatusCode == 401 {
			return fmt.Errorf("authentication required. Run 'promptbucket login' first")
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response - API returns object with results array
		var response struct {
			Results []map[string]interface{} `json:"results"`
			Total   int                      `json:"total"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		packages := response.Results

		// Display results
		if len(packages) == 0 {
			fmt.Println("\nYou haven't starred any packages yet.")
			fmt.Println("\nStar a package with:")
			fmt.Println("  promptbucket star org/package")
			return nil
		}

		fmt.Printf("\n⭐ Your starred packages (%d):\n\n", len(packages))
		
		for i, pkg := range packages {
			// Extract package info
			name, _ := pkg["name"].(string)
			org, _ := pkg["org"].(string)
			description, _ := pkg["description"].(string)
			
			// Display package
			fmt.Printf("%d. 📦 ", i+1)
			if org != "" && name != "" {
				fmt.Printf("%s/%s", org, name)
			} else if name != "" {
				fmt.Printf("%s", name)
			}
			
			// Show pull count if available
			if pullCount, ok := getFloatValue(pkg["pull_count"]); ok && pullCount > 0 {
				fmt.Printf(" 📥 %.0f", pullCount)
			}
			
			fmt.Println()
			
			// Show description
			if description != "" {
				fmt.Printf("   %s\n", description)
			}
			
			// Show how to unstar
			if org != "" && name != "" {
				fmt.Printf("   Unstar: promptbucket unstar %s/%s\n", org, name)
			}
			
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(starCmd)
	rootCmd.AddCommand(unstarCmd)
	rootCmd.AddCommand(starsCmd)
}