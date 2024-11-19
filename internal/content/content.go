package content

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Represents a markdown file with a YAML header containing metadata about that
// file.
type ContentFile struct {
	// The built (i.e. HTML) content.
	Content string `yaml:"Content"`
	// The contents of the yaml header.
	Metadata ContentMetadata `yaml:"Metadata"`
	// The path to the built content file relative to the output directory.
	Path string `yaml:"Path"`
	// The markdown content of the file from below the yaml header.
	RawContent string `yaml:"RawContent"`
	// The path to the file the Content struct was built from.
	SourcePath string `yaml:"SourcePath"`
}

type ContentMetadata struct {
	LastModified time.Time `yaml:"LastModified,omitempty"`
	Published    time.Time `yaml:"Published,omitempty"`
	TemplateName string    `yaml:"TemplateName,omitempty"`
	Title        string    `yaml:"Title,omitempty"`
}

func ReadFile(filePath string) (*ContentFile, error) {
	rawContentFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	parts := strings.Split(string(rawContentFile), "---")
	if len(parts) != 3 {
		return nil, errors.New(`file not split with \"---\" correctly`)
	}
	rawMetadata := strings.TrimSpace(parts[1])
	content := strings.TrimSpace(parts[2])

	contentMetadata := ContentMetadata{}
	if err := yaml.Unmarshal([]byte(rawMetadata), &contentMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata as yaml: %w", err)
	}

	contentFile := &ContentFile{
		Metadata:   contentMetadata,
		RawContent: content,
		SourcePath: filePath,
	}

	if err := contentFile.Validate(); err != nil {
		return nil, fmt.Errorf("invalid content file: %w", err)
	}

	return contentFile, nil
}

func (contentFile ContentFile) Validate() error {
	if contentFile.Metadata.TemplateName == "" {
		return fmt.Errorf("must define TemplateName")
	}
	return nil
}
