package content

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
