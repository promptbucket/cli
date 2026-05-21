package cmd

import (
	"fmt"
	"os"

	"github.com/promptbucket/cli/internal/mcp"
	"github.com/spf13/cobra"
)

var servePersonaDir string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an MCP server for the persona in this project",
	Long: `Start a Model Context Protocol (MCP) server over stdio.

Claude Code, Cursor, Cline, and other MCP clients can connect to this server
to get the persona's system prompt and memory context injected automatically.

To connect Claude Code, add this to your .claude/mcp.json:
  {
    "mcpServers": {
      "persona": {
        "command": "promptbucket",
        "args": ["serve"]
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := servePersonaDir

		// If no --persona flag, look for .promptbucket/ in current directory
		if dir == "" {
			if _, err := os.Stat(".promptbucket"); err == nil {
				dir = ".promptbucket"
			} else {
				return fmt.Errorf("no .promptbucket/ directory found in current directory\nRun: promptbucket init")
			}
		}

		srv, err := mcp.New(dir)
		if err != nil {
			return fmt.Errorf("failed to load persona: %w", err)
		}

		return srv.Serve()
	},
}

func init() {
	serveCmd.Flags().StringVar(&servePersonaDir, "persona", "", "Path to persona directory (default: .promptbucket/)")
	rootCmd.AddCommand(serveCmd)
}
