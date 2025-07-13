package cmd

import (
    "fmt"

    "github.com/promptbucket/cli/internal/packager"
    "github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Build .pbt package",
    RunE: func(cmd *cobra.Command, args []string) error {
        art, err := packager.Build()
        if err != nil {
            return err
        }
        fmt.Printf("%s\t%d KB\t%s\n", art.Path, art.Size/1024, art.Digest)
        return nil
    },
}

func init() { rootCmd.AddCommand(buildCmd) }
