package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "strings"

    "github.com/promptbucket/cli/internal/packager"
    "github.com/spf13/cobra"
)

type ToolAdapter struct {
    Name        string
    CmdTemplate string
}

var Adapters = map[string]ToolAdapter{
    "codex": {Name: "codex", CmdTemplate: "codex run --stdin"},
}

var (
    varFlags    []string
    toolFlag    string
    contextFlag string
)

var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Build prompt with variable substitution and generate final_prompt.md",
    RunE: func(cmd *cobra.Command, args []string) error {
        // If no flags provided, use legacy build
        if len(varFlags) == 0 && toolFlag == "" {
            art, err := packager.Build()
            if err != nil {
                return err
            }
            fmt.Printf("%s\t%d KB\t%s\n", art.Path, art.Size/1024, art.Digest)
            return nil
        }
        
        // New build with variables
        if err := packager.BuildWithVariables(varFlags); err != nil {
            return err
        }
        
        // If tool is specified, pipe the prompt to it
        if toolFlag != "" {
            return runWithTool(toolFlag)
        }
        
        return nil
    },
}

func runWithTool(toolName string) error {
    adapter, exists := Adapters[toolName]
    if !exists {
        return fmt.Errorf("unsupported tool: %s", toolName)
    }
    
    // Get the expected prompt filename
    filename, err := packager.GetPromptFilename(packager.ManifestFile)
    if err != nil {
        return fmt.Errorf("failed to get prompt filename: %w", err)
    }
    
    // Read the prompt file
    content, err := os.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("failed to read %s: %w", filename, err)
    }
    
    // Parse command template
    cmdParts := strings.Fields(adapter.CmdTemplate)
    if len(cmdParts) == 0 {
        return fmt.Errorf("invalid command template for tool %s", toolName)
    }
    
    // Execute tool with stdin
    cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
    cmd.Stdin = strings.NewReader(string(content))
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}

func init() {
    buildCmd.Flags().StringArrayVar(&varFlags, "var", []string{}, "Variable substitution (key=value, repeatable)")
    buildCmd.Flags().StringVar(&toolFlag, "tool", "", "Tool to pipe the rendered prompt to (e.g., codex)")
    buildCmd.Flags().StringVar(&contextFlag, "context", "", "Context file for future injection (stub)")
    rootCmd.AddCommand(buildCmd)
}
