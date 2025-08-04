package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "strings"

    "github.com/promptbucket/cli/internal/packager"
    "github.com/spf13/cobra"
)

var (
    runVarFlags    []string
    runToolFlag    string
    runContextFlag string
)

var runCmd = &cobra.Command{
    Use:   "run",
    Short: "Build prompt with variable substitution and optionally pipe to tool",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Build with variables
        if err := packager.BuildWithVariables(runVarFlags); err != nil {
            return err
        }
        
        // If tool is specified, pipe the prompt to it
        if runToolFlag != "" {
            return runWithToolIntegration(runToolFlag)
        }
        
        return nil
    },
}

func runWithToolIntegration(toolName string) error {
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
    runCmd.Flags().StringArrayVar(&runVarFlags, "var", []string{}, "Variable substitution (key=value, repeatable)")
    runCmd.Flags().StringVar(&runToolFlag, "tool", "", "Tool to pipe the rendered prompt to (e.g., codex)")
    runCmd.Flags().StringVar(&runContextFlag, "context", "", "Context file for future injection (stub)")
    rootCmd.AddCommand(runCmd)
}