package cmd

import (
	"fmt"
	"os"

	"github.com/adamkpickering/jenny/internal/config"
	"github.com/spf13/cobra"
)

const configPath = "configuration.yaml"

var configYaml config.ConfigYaml

var rootCmd = &cobra.Command{
	Use: "jenny",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	newConfig, err := config.ReadFile(configPath)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	configYaml = newConfig

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
