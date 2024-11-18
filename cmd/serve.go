package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const reloadScript = `<script>console.log("placeholder")</script>`

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve <addr>",
	Short: "Serve static site",
	Args:  cobra.ExactArgs(1),
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	go watchAndBuild(ctx, stop)

	server := http.Server{
		Addr:    args[0],
		Handler: http.FileServer(http.Dir(configYaml.Output)),
	}

	go func() {
		log.Printf("listening on %s", args[0])
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server error: %s", err)
			stop()
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("error while shutting down server: %w", err)
	}

	return nil
}

func watchAndBuild(ctx context.Context, stop func()) {
	var watcher *fsnotify.Watcher
	var err error
	filePath := "INITIAL_BUILD"

forloop:
	for {
		log.Printf("build triggered by %s", filePath)
		if err := build(); err != nil {
			log.Printf("failed to build: %s", err)
			break forloop
		}
		if err := modifyHtmlFiles(); err != nil {
			log.Printf("failed to modify HTML files: %s", err)
			break forloop
		}

		if watcher != nil {
			watcher.Close()
		}
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Printf("failed to construct watcher: %s", err)
			break forloop
		}
		defer watcher.Close()
		if err := watcher.Add(configYaml.Content); err != nil {
			log.Printf("failed to watch %s: %s", configYaml.Content, err)
			break forloop
		}

		select {
		case _, ok := <-ctx.Done():
			if !ok {
				break forloop
			}
		case event, ok := <-watcher.Events:
			if !ok {
				break forloop
			}
			filePath = event.Name
		case err, ok := <-watcher.Errors:
			if !ok {
				break forloop
			}
			log.Printf("error from watcher: %s", err)
			break forloop
		}

		// avoid unnecessary rebuilds
		time.Sleep(100 * time.Millisecond)
	}

	if watcher != nil {
		watcher.Close()
	}
	stop()
}

func modifyHtmlFiles() error {
	walkDirFunc := func(outputPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return nil
		}
		if filepath.Ext(outputPath) != ".html" {
			return nil
		}

		if err := injectReloadScript(outputPath); err != nil {
			return fmt.Errorf("failed to inject reload script into %s: %w", outputPath, err)
		}

		return nil
	}

	if err := filepath.WalkDir(configYaml.Output, walkDirFunc); err != nil {
		return err
	}

	return nil
}

// injectScript injects the reloading script into a given .html file.
func injectReloadScript(filePath string) error {
	fd, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer fd.Close()

	document, err := html.Parse(fd)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}
	found := false
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && node.DataAtom == atom.Head {
			scriptNode := &html.Node{
				Type: html.RawNode,
				Data: reloadScript,
			}
			node.AppendChild(scriptNode)
			found = true
			break
		}
	}
	if !found {
		return errors.New("failed to find head element")
	}

	if err := fd.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate: %w", err)
	}
	if _, err := fd.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	if err := html.Render(fd, document); err != nil {
		return fmt.Errorf("failed to render modified document: %w", err)
	}

	return nil
}
