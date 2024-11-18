package cmd

import (
	"fmt"
	"os"

	"github.com/adamkpickering/jenny/internal/content"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(templateDataCmd)
}

var templateDataCmd = &cobra.Command{
	Use:   "template-data",
	Short: "Print the data available to a template for a given content file",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplateData,
}

func runTemplateData(cmd *cobra.Command, args []string) error {
	_, templateData, err := gatherFileInfo(configYaml)
	if err != nil {
		return fmt.Errorf("failed to gather info on input files: %w", err)
	}

	// look for the specified content file, and redact content
	// so that the output is legible
	foundContentFile := &content.ContentFile{}
	for _, contentFile := range templateData.Pages {
		contentFile.Content = "redacted"
		contentFile.RawContent = "redacted"
		if contentFile.SourcePath == args[0] {
			foundContentFile = contentFile
		}
	}
	if foundContentFile.Path == "" {
		return fmt.Errorf("failed to find content file %q", args[0])
	}
	templateData.Page = foundContentFile

	encoder := yaml.NewEncoder(os.Stdout)
	if err := encoder.Encode(templateData); err != nil {
		return fmt.Errorf("failed to encode template data to yaml: %w", err)
	}

	return nil
}
