package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve <addr>",
	Short: "A brief description of your command",
	Args:  cobra.ExactArgs(1),
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	if err := build(); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	handler := http.FileServer(http.Dir(outputPath))
	fmt.Printf("Listening on %s\n", args[0])
	return http.ListenAndServe(args[0], handler)
}
