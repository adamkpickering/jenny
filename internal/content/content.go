package content

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Represents a markdown file with a YAML header containing metadata about that
// file.
type Content struct {
	Metadata ContentMetadata
	Content  string
}

type ContentMetadata struct {
	Title        string
	TemplateName string
}

func ReadFile(filePath string) (Content, error) {
	rawContentFile, err := os.ReadFile(filePath)
	if err != nil {
		return Content{}, fmt.Errorf("failed to read file: %w", err)
	}

	parts := strings.Split(string(rawContentFile), "---")
	if len(parts) != 3 {
		return Content{}, errors.New(`file not split with \"---\" correctly`)
	}
	rawMetadata := strings.TrimSpace(parts[1])
	content := strings.TrimSpace(parts[2])

	contentMetadata := ContentMetadata{}
	if err := yaml.Unmarshal([]byte(rawMetadata), &contentMetadata); err != nil {
		return Content{}, fmt.Errorf("failed to parse metadata as yaml: %w", err)
	}

	contentFile := Content{
		Metadata: contentMetadata,
		Content:  content,
	}

	return contentFile, nil
}
