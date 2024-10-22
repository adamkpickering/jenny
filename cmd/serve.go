package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

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

var buildMutex = sync.Mutex{}

func exclusiveBuild(op string, filePath string) error {
	if !buildMutex.TryLock() {
		return nil
	}
	defer buildMutex.Unlock()
	log.Printf("build triggered by %s on %s", op, filePath)
	if err := build(); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to construct watcher: %w", err)
	}
	defer watcher.Close()

	for _, dirPath := range []string{templatesPath, contentPath, staticPath} {
		if err := watcher.Add(dirPath); err != nil {
			return fmt.Errorf("failed to watch %s: %w", dirPath, err)
		}
	}

	go func() {
		op := "INITIAL_BUILD"
		filePath := "N/A"
	forloop:
		for {
			if err := exclusiveBuild(op, filePath); err != nil {
				log.Println(err)
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
				op = event.Op.String()
				filePath = event.Name
			case err, ok := <-watcher.Errors:
				if !ok {
					break forloop
				}
				log.Printf("error from watcher: %s", err)
				break forloop
			}
		}
		stop()
	}()

	server := http.Server{
		Addr:    args[0],
		Handler: http.FileServer(http.Dir(outputPath)),
	}

	go func() {
		log.Printf("listening on %s", args[0])
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server failed: %s", err)
			stop()
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("error while shutting down server: %w", err)
	}

	return nil
}
