package cmd

import (
	"fmt"
	"os"

	"github.com/adamkpickering/jenny/internal/config"
	"github.com/spf13/cobra"
)

var configYaml config.ConfigYaml

var rootCmd = &cobra.Command{
	Use: "jenny",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	PersistentPreRunE: populateConfigYaml,
}

func populateConfigYaml(cmd *cobra.Command, args []string) error {
	var err error
	configYaml, err = config.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}

func SetVersionInfo(version string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{ .Version }}\n")
}
