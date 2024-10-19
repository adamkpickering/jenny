package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yuin/goldmark"

	"gopkg.in/yaml.v3"
)

const (
	contentPath   = "content"
	outputPath    = "output"
	templatesPath = "templates"
)

var templates *template.Template

func init() {
	templates = template.Must(template.ParseGlob("templates/*.gotmpl"))
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	dirEntries, err := os.ReadDir(contentPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", contentPath, err)
	}

	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to ensure output path exists: %w", err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		contentPath := filepath.Join(contentPath, dirEntry.Name())
		parts := strings.Split(dirEntry.Name(), ".")
		if len(parts) != 2 {
			return fmt.Errorf("failed to split %q into name and extension", dirEntry.Name())
		}
		outputPath := filepath.Join(outputPath, parts[0]+".html")

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
	fmt.Printf("%+v\n", contentMetadata)

	contentFile := ContentFile{
		Metadata: contentMetadata,
		Content:  content,
	}

	return contentFile, nil
}

func parseContentMetadata(rawMetadata string) (ContentMetadata, error) {
	metadata := ContentMetadata{}
	if err := yaml.Unmarshal([]byte(rawMetadata), &metadata); err != nil {
		return ContentMetadata{}, fmt.Errorf("failed to parse metadata as yaml: %w", err)
	}
	return metadata, nil
}
