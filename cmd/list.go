package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages from the registry",
	Long: `List packages available in the PromptBucket registry.

You can list popular packages, trending packages, or search for specific packages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		search, _ := cmd.Flags().GetString("search")
		trending, _ := cmd.Flags().GetBool("trending")
		popular, _ := cmd.Flags().GetBool("popular")
		limit, _ := cmd.Flags().GetInt("limit")

		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Determine which endpoint to use
		var endpoint string
		if search != "" {
			endpoint = fmt.Sprintf("/packages/search?q=%s", search)
			if limit > 0 {
				endpoint = fmt.Sprintf("%s&limit=%d", endpoint, limit)
			}
		} else if trending {
			endpoint = "/packages/trending"
		} else if popular {
			endpoint = "/packages/popular"
		} else {
			// Default to popular if no specific option
			endpoint = "/packages/popular"
		}

		// Make request
		fmt.Println("📋 Fetching packages from registry...")
		resp, err := apiClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to fetch packages: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status
		if resp.StatusCode != 200 {
			var errorResp map[string]interface{}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if msg, ok := errorResp["message"].(string); ok {
					return fmt.Errorf("request failed: %s", msg)
				}
				if errMsg, ok := errorResp["error"].(string); ok {
					return fmt.Errorf("request failed: %s", errMsg)
				}
			}
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response based on endpoint type
		if search != "" {
			// Search endpoint returns a different structure
			var searchResp struct {
				Query   string                   `json:"query"`
				Results []map[string]interface{} `json:"results"`
				Total   int                      `json:"total"`
			}
			if err := json.Unmarshal(body, &searchResp); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if searchResp.Total == 0 {
				fmt.Printf("No packages found matching \"%s\".\n", search)
				return nil
			}

			fmt.Printf("\nFound %d package(s) matching \"%s\":\n\n", searchResp.Total, search)
			displayPackages(searchResp.Results)
		} else {
			// Popular/trending endpoints might return object with results or array
			var packages []map[string]interface{}
			
			// Try to parse as object with results first
			var objResp struct {
				Results []map[string]interface{} `json:"results"`
				Total   int                      `json:"total"`
				Limit   int                      `json:"limit"`
			}
			
			if err := json.Unmarshal(body, &objResp); err == nil && objResp.Results != nil {
				packages = objResp.Results
			} else {
				// Try parsing as direct array
				if err := json.Unmarshal(body, &packages); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
			}

			if len(packages) == 0 {
				fmt.Println("No packages found.")
				return nil
			}

			if trending {
				fmt.Printf("\n📈 Trending packages (%d):\n\n", len(packages))
			} else {
				fmt.Printf("\n⭐ Popular packages (%d):\n\n", len(packages))
			}
			displayPackages(packages)
		}

		return nil
	},
}

func displayPackages(packages []map[string]interface{}) {
	for i, pkg := range packages {
		// Extract package info
		name, _ := pkg["name"].(string)
		orgName, _ := pkg["org_name"].(string)
		if orgName == "" {
			orgName, _ = pkg["org"].(string)
		}
		description, _ := pkg["description"].(string)
		
		// Display package number
		fmt.Printf("%d. ", i+1)
		
		// Display package name with org
		if orgName != "" && name != "" {
			fmt.Printf("📦 %s/%s", orgName, name)
		} else if name != "" {
			fmt.Printf("📦 %s", name)
		} else if id, ok := pkg["id"].(string); ok {
			fmt.Printf("📦 %s", id)
		}

		// Show stats if available
		if starCount, ok := getFloat(pkg["star_count"]); ok && starCount > 0 {
			fmt.Printf(" ⭐ %.0f", starCount)
		}
		if pullCount, ok := getFloat(pkg["pull_count"]); ok && pullCount > 0 {
			fmt.Printf(" 📥 %.0f", pullCount)
		}
		
		fmt.Println()
		
		// Show description
		if description != "" {
			fmt.Printf("   %s\n", description)
		}
		
		// Show tags if available
		if tags, ok := pkg["tags"].([]interface{}); ok && len(tags) > 0 {
			tagStrings := make([]string, 0, len(tags))
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					tagStrings = append(tagStrings, tagStr)
				}
			}
			if len(tagStrings) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(tagStrings, ", "))
			}
		}
		
		// Show how to get more info
		if orgName != "" && name != "" {
			fmt.Printf("   Info: promptbucket info %s/%s\n", orgName, name)
		}
		
		fmt.Println()
	}
}

// Helper function to get float value from interface{}
func getFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("search", "s", "", "Search packages by keyword")
	listCmd.Flags().BoolP("trending", "t", false, "Show trending packages")
	listCmd.Flags().BoolP("popular", "p", false, "Show popular packages")
	listCmd.Flags().Int("limit", 20, "Maximum number of packages to display")
}