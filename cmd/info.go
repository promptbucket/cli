package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [package]",
	Short: "Show detailed information about a package",
	Long: `Show detailed information about a specific package.

Package can be specified as:
  - org/name (e.g., rawte.mayur/Ui-Artist)
  - name (will search for it)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageSpec := args[0]

		// Parse package specification
		var orgName, packageName string
		if strings.Contains(packageSpec, "/") {
			parts := strings.SplitN(packageSpec, "/", 2)
			orgName = parts[0]
			packageName = parts[1]
		} else {
			// If no org specified, we'll need to search for it
			packageName = packageSpec
		}

		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// If we have both org and name, get directly
		if orgName != "" && packageName != "" {
			return showPackageInfo(apiClient, orgName, packageName)
		}

		// Otherwise, search for the package first
		fmt.Printf("🔍 Searching for package '%s'...\n", packageName)
		
		endpoint := fmt.Sprintf("/packages/search?q=%s&limit=5", packageName)
		resp, err := apiClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to search for package: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("search failed with status %d", resp.StatusCode)
		}

		var searchResp struct {
			Results []struct {
				Name string `json:"name"`
				Org  string `json:"org"`
				OrgID string `json:"org_id"`
			} `json:"results"`
			Total int `json:"total"`
		}
		
		if err := json.Unmarshal(body, &searchResp); err != nil {
			return fmt.Errorf("failed to parse search results: %w", err)
		}

		if searchResp.Total == 0 {
			return fmt.Errorf("package '%s' not found", packageName)
		}

		// If multiple matches, ask user to be more specific
		if searchResp.Total > 1 {
			fmt.Printf("\nFound %d packages matching '%s':\n", searchResp.Total, packageName)
			for _, pkg := range searchResp.Results {
				if pkg.Org != "" {
					fmt.Printf("  - %s/%s\n", pkg.Org, pkg.Name)
				} else {
					fmt.Printf("  - %s\n", pkg.Name)
				}
			}
			fmt.Println("\nPlease specify the full package name (org/name)")
			return nil
		}

		// Found exactly one match
		result := searchResp.Results[0]
		if result.Org == "" {
			// Try to extract org from org_id if org field is empty
			result.Org = result.OrgID
		}
		
		return showPackageInfo(apiClient, result.Org, result.Name)
	},
}

func showPackageInfo(apiClient *auth.APIClient, orgName, packageName string) error {
	if orgName == "" || packageName == "" {
		return fmt.Errorf("invalid package specification")
	}

	fmt.Printf("📦 Fetching info for %s/%s...\n\n", orgName, packageName)

	// Get package details
	endpoint := fmt.Sprintf("/packages/%s/%s", orgName, packageName)
	resp, err := apiClient.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to get package info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return fmt.Errorf("package %s/%s not found", orgName, packageName)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse package info
	var pkgInfo map[string]interface{}
	if err := json.Unmarshal(body, &pkgInfo); err != nil {
		return fmt.Errorf("failed to parse package info: %w", err)
	}

	// Display package information
	fmt.Printf("📦 Package: %s/%s\n", orgName, packageName)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Description
	if desc, ok := pkgInfo["description"].(string); ok && desc != "" {
		fmt.Printf("📝 Description:\n   %s\n\n", desc)
	}

	// Stats
	fmt.Printf("📊 Statistics:\n")
	if starCount, ok := getFloatValue(pkgInfo["star_count"]); ok {
		fmt.Printf("   ⭐ Stars: %.0f\n", starCount)
	}
	if pullCount, ok := getFloatValue(pkgInfo["pull_count"]); ok {
		fmt.Printf("   📥 Pulls: %.0f\n", pullCount)
	}
	if hasStarred, ok := pkgInfo["has_starred"].(bool); ok && hasStarred {
		fmt.Printf("   ✨ You've starred this package\n")
	}
	fmt.Println()

	// Versions
	if versions, ok := pkgInfo["versions"].([]interface{}); ok && len(versions) > 0 {
		fmt.Printf("📋 Versions (%d):\n", len(versions))
		for i, v := range versions {
			if i >= 5 {
				fmt.Printf("   ... and %d more\n", len(versions)-5)
				break
			}
			if version, ok := v.(map[string]interface{}); ok {
				tag, _ := version["tag"].(string)
				createdAt, _ := version["created_at"].(string)
				digest, _ := version["digest"].(string)
				
				fmt.Printf("   • %s", tag)
				if createdAt != "" {
					if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
						fmt.Printf(" (published %s)", t.Format("2006-01-02"))
					}
				}
				if digest != "" && len(digest) > 16 {
					fmt.Printf("\n     Digest: %s...%s", digest[:8], digest[len(digest)-8:])
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Dates
	if createdAt, ok := pkgInfo["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			fmt.Printf("📅 Created: %s\n", t.Format("Jan 2, 2006"))
		}
	}
	if updatedAt, ok := pkgInfo["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			fmt.Printf("🔄 Updated: %s\n", t.Format("Jan 2, 2006"))
		}
	}

	// Usage instructions
	fmt.Printf("\n📖 Usage:\n")
	fmt.Printf("   Pull latest:  promptbucket pull %s/%s\n", orgName, packageName)
	if versions, ok := pkgInfo["versions"].([]interface{}); ok && len(versions) > 0 {
		if v, ok := versions[0].(map[string]interface{}); ok {
			if tag, ok := v["tag"].(string); ok {
				fmt.Printf("   Pull version: promptbucket pull %s/%s:%s\n", orgName, packageName, tag)
				fmt.Printf("   Fetch YAML:   promptbucket fetch https://api.promptbucket.co/v1/manifests/%s/%s/%s\n", orgName, packageName, tag)
			}
		}
	}

	return nil
}

// Helper function to get float value from interface{}
func getFloatValue(v interface{}) (float64, bool) {
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
	rootCmd.AddCommand(infoCmd)
}