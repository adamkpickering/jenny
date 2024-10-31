package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	templatesGlob := filepath.Join(configJson.Templates, "*.gotmpl")
	templates, err := template.ParseGlob(templatesGlob)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	// wipe output directory
	if err := os.RemoveAll(configJson.Output); err != nil {
		return fmt.Errorf("failed to wipe output dir: %w", err)
	}
	if err := os.MkdirAll(configJson.Output, 0o755); err != nil {
		return fmt.Errorf("failed to ensure output dir exists: %w", err)
	}

	buildFunc := func(contentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(configJson.Content, contentPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path of %s: %w", contentPath, err)
		}
		outputPath := filepath.Join(configJson.Output, relativePath)
		if dirEntry.IsDir() {
			if err := os.MkdirAll(outputPath, 0o755); err != nil {
				return fmt.Errorf("failed to create %s: %w", outputPath, err)
			}
			return nil
		}

		if ext := filepath.Ext(contentPath); ext != ".md" {
			if err := copyFile(outputPath, contentPath); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", contentPath, outputPath, err)
			}
			return nil
		}

		// get output path
		parentDir, fileName := filepath.Split(contentPath)
		parts := strings.Split(fileName, ".")
		if len(parts) != 2 {
			return fmt.Errorf("failed to split %q into name and extension", dirEntry.Name())
		}
		relativeParentDir, err := filepath.Rel(configJson.Content, parentDir)
		if err != nil {
			return fmt.Errorf("failed to get path of parent dir %s relative to %s: %w", parentDir, configJson.Content, err)
		}
		outputPath = filepath.Join(configJson.Output, relativeParentDir, parts[0]+".html")

		contentFile, err := ParseContentFile(contentPath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", contentPath, err)
		}

		content := &bytes.Buffer{}
		if err := goldmark.Convert([]byte(contentFile.Content), content); err != nil {
			return fmt.Errorf("failed to convert %s to html: %w", contentPath, err)
		}

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

		return nil
	}

	if err := filepath.WalkDir(configJson.Content, buildFunc); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	return nil
}

func copyFile(dst, src string) error {
	srcFd, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFd.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create destination parent dir: %w", err)
	}
	dstFd, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer dstFd.Close()
	if _, err := io.Copy(dstFd, srcFd); err != nil {
		return fmt.Errorf("failed to copy contents of src to dst: %w", err)
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
