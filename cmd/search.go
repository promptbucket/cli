package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for packages in the registry",
	Long: `Search for packages in the PromptBucket registry using keywords.

The search looks through package names, descriptions, tags, and authors.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		limit, _ := cmd.Flags().GetInt("limit")

		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Build endpoint
		endpoint := fmt.Sprintf("/packages/search?q=%s", query)
		if limit > 0 {
			endpoint = fmt.Sprintf("%s&limit=%d", endpoint, limit)
		}

		// Search packages
		fmt.Printf("🔍 Searching for \"%s\"...\n", query)
		resp, err := apiClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
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
					return fmt.Errorf("search failed: %s", msg)
				}
				if errMsg, ok := errorResp["error"].(string); ok {
					return fmt.Errorf("search failed: %s", errMsg)
				}
			}
			return fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Parse results - the API returns an object with results array
		var searchResp struct {
			Query   string                   `json:"query"`
			Results []map[string]interface{} `json:"results"`
			Total   int                      `json:"total"`
			Limit   int                      `json:"limit"`
			Offset  int                      `json:"offset"`
		}
		
		if err := json.Unmarshal(body, &searchResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Display results
		if searchResp.Total == 0 {
			fmt.Printf("No packages found matching \"%s\".\n", query)
			fmt.Println("\nTry:")
			fmt.Println("  • Using different keywords")
			fmt.Println("  • Browsing all packages with 'promptbucket list'")
			return nil
		}

		fmt.Printf("\nFound %d result(s):\n\n", searchResp.Total)
		
		for i, result := range searchResp.Results {
			// Extract fields
			name, _ := result["name"].(string)
			orgID, _ := result["org_id"].(string)
			org, _ := result["org"].(string)
			description, _ := result["description"].(string)
			
			// Display result number and name
			fmt.Printf("%d. 📦 ", i+1)
			if org != "" && name != "" {
				fmt.Printf("%s/%s", org, name)
			} else if orgID != "" && name != "" {
				fmt.Printf("%s/%s", orgID, name)
			} else if name != "" {
				fmt.Printf("%s", name)
			}
			
			// Show stats if available
			if starCount, ok := getFloat(result["star_count"]); ok && starCount > 0 {
				fmt.Printf(" ⭐ %.0f", starCount)
			}
			if pullCount, ok := getFloat(result["pull_count"]); ok && pullCount > 0 {
				fmt.Printf(" 📥 %.0f", pullCount)
			}
			
			fmt.Println()
			
			// Show description
			if description != "" {
				fmt.Printf("   %s\n", description)
			}
			
			// Show tags if available
			if tags, ok := result["tags"].([]interface{}); ok && len(tags) > 0 {
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
			if org != "" && name != "" {
				fmt.Printf("   Info: promptbucket info %s/%s\n", org, name)
			} else if name != "" {
				fmt.Printf("   Info: promptbucket info %s\n", name)
			}
			
			fmt.Println()
		}

		return nil
	},
}

// getFloat is defined in list.go, so we don't need to redefine it here

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().Int("limit", 10, "Maximum number of results to display")
}