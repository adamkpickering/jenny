package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

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
