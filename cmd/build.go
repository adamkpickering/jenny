package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/adamkpickering/jenny/internal/config"
	"github.com/adamkpickering/jenny/internal/content"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
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
	templatesGlob := filepath.Join(configYaml.Templates, "*.gotmpl")
	templates, err := template.ParseGlob(templatesGlob)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	// wipe output directory
	if err := os.RemoveAll(configYaml.Output); err != nil {
		return fmt.Errorf("failed to wipe output dir: %w", err)
	}
	if err := os.MkdirAll(configYaml.Output, 0o755); err != nil {
		return fmt.Errorf("failed to ensure output dir exists: %w", err)
	}

	buildFunc := func(contentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(configYaml.Content, contentPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path of %s: %w", contentPath, err)
		}
		outputPath := filepath.Join(configYaml.Output, relativePath)
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
		relativeParentDir, err := filepath.Rel(configYaml.Content, parentDir)
		if err != nil {
			return fmt.Errorf("failed to get path of parent dir %s relative to %s: %w", parentDir, configYaml.Content, err)
		}
		outputPath = filepath.Join(configYaml.Output, relativeParentDir, parts[0]+".html")

		contentFile, err := content.ReadFile(contentPath)
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

	if err := filepath.WalkDir(configYaml.Content, buildFunc); err != nil {
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

// TemplateData is the data that gets passed when building a template.
type TemplateData struct {
	// The specific page that is being rendered in this template execution.
	Page content.Content
	// A slice of all content pages in this website.
	Pages []content.Content
	// Any extra data that doesn't have anything to do with pages that we want
	// to make available in templates.
	Context TemplateContext
}

type TemplateContext struct {
	Now    time.Time
	Config config.ConfigYaml
}
