package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var version = "dev"

var rootCmd = &cobra.Command{
    Use:   "pbt",
    Short: "PromptBucket CLI",
    Run: func(cmd *cobra.Command, args []string) {
        cmd.Help()
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    viper.AutomaticEnv()
    rootCmd.PersistentFlags().BoolP("version", "v", false, "print version")
    viper.BindPFlag("version", rootCmd.PersistentFlags().Lookup("version"))
    rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
        if viper.GetBool("version") {
            fmt.Println(version)
            os.Exit(0)
        }
    }
}
