package cmd

import (
    "errors"
    "os"

    "github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Scaffold promptbucket.yaml",
    RunE: func(cmd *cobra.Command, args []string) error {
        if _, err := os.Stat("promptbucket.yaml"); err == nil {
            return errors.New("promptbucket.yaml already exists")
        }
        example := `name: example
version: 0.1.0
licence: Apache-2.0
prompt: |-
  You are a helpful assistant.`
        return os.WriteFile("promptbucket.yaml", []byte(example+"\n"), 0644)
    },
}

func init() { rootCmd.AddCommand(initCmd) }
