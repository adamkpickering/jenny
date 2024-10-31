package cmd

import (
	"fmt"
	"os"

	"github.com/adamkpickering/jenny/internal/config"
	"github.com/spf13/cobra"
)

const configPath = "configuration.json"

var configJson config.ConfigJson

var rootCmd = &cobra.Command{
	Use:   "jenny",
	Short: "jenny is a simple static site generator",
}

func Execute() {
	newConfig, err := config.Read(configPath)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	configJson = newConfig

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
