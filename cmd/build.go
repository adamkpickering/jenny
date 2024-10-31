package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"

	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	RunE:  runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
	return build()
}

func build() error {
	if err := os.RemoveAll(configJson.Output); err != nil {
		return fmt.Errorf("failed to wipe output dir: %w", err)
	}
	if err := buildContent(); err != nil {
		return fmt.Errorf("failed to build content: %w", err)
	}
	if err := copyStatic(); err != nil {
		return fmt.Errorf("failed to copy static directory: %w", err)
	}
	return nil
}

func copyStatic() error {
	outputStaticPath := filepath.Join(configJson.Output, "static")
	if err := os.RemoveAll(outputStaticPath); err != nil {
		return fmt.Errorf("failed to remove existing static output directory: %w", err)
	}
	staticFs := os.DirFS(configJson.Static)
	if err := os.CopyFS(outputStaticPath, staticFs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to copy filesystem: %w", err)
	}
	return nil
}

func buildContent() error {
	templatesGlob := filepath.Join(configJson.Templates, "*.gotmpl")
	templates, err := template.ParseGlob(templatesGlob)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	dirEntries, err := os.ReadDir(configJson.Content)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configJson.Content, err)
	}

	if err := os.MkdirAll(configJson.Output, 0o755); err != nil {
		return fmt.Errorf("failed to ensure output path exists: %w", err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		contentPath := filepath.Join(configJson.Content, dirEntry.Name())
		parts := strings.Split(dirEntry.Name(), ".")
		if len(parts) != 2 {
			return fmt.Errorf("failed to split %q into name and extension", dirEntry.Name())
		}
		outputPath := filepath.Join(configJson.Output, parts[0]+".html")

		contentFile, err := ParseContentFile(contentPath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", contentPath, err)
		}

		// convert markdown in content file to html
		content := &bytes.Buffer{}
		if err := goldmark.Convert([]byte(contentFile.Content), content); err != nil {
			return fmt.Errorf("failed to convert %s to html: %w", contentPath, err)
		}

		// fill template
		fd, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", outputPath, err)
		}
		defer fd.Close()
		data := map[string]string{
			"Content": content.String(),
		}
		if err := templates.ExecuteTemplate(fd, contentFile.Metadata.TemplateName, data); err != nil {
			return fmt.Errorf("failed to execute %s for %s: %w", contentFile.Metadata.TemplateName, outputPath, err)
		}
	}

	return nil
}

type ContentFile struct {
	Metadata ContentMetadata
	Content  string
}

type ContentMetadata struct {
	Title        string `yaml:"title"`
	TemplateName string `yaml:"templateName"`
}

func ParseContentFile(filePath string) (ContentFile, error) {
	rawContentFile, err := os.ReadFile(filePath)
	if err != nil {
		return ContentFile{}, fmt.Errorf("failed to read file: %w", err)
	}

	parts := strings.Split(string(rawContentFile), "---")
	if len(parts) != 3 {
		return ContentFile{}, errors.New(`file not split with \"---\" correctly`)
	}
	rawMetadata := strings.TrimSpace(parts[1])
	content := strings.TrimSpace(parts[2])

	contentMetadata := ContentMetadata{}
	if err := yaml.Unmarshal([]byte(rawMetadata), &contentMetadata); err != nil {
		return ContentFile{}, fmt.Errorf("failed to parse metadata as yaml: %w", err)
	}

	contentFile := ContentFile{
		Metadata: contentMetadata,
		Content:  content,
	}

	return contentFile, nil
}
