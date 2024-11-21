package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"time"

	"github.com/coder/websocket"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

const reloadScriptTemplate = `<script>
  let ws = new WebSocket("http://%s%s");
  ws.onmessage = (event) => {
    if (event.data === "%s") {
      window.location.reload();
    }
  }
</script>`
const reloadMsg = "reload"
const websocketPath = "/websocket"

var host string

func init() {
	serveCmd.PersistentFlags().StringVar(&host, "host", "localhost:9023", "host and port to listen on in host:port format")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve static site",
	Args:  cobra.NoArgs,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	websocketUrl, err := url.Parse("http://" + host + websocketPath)
	if err != nil {
		return fmt.Errorf("failed to parse websocket URL: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	reloadNotificationChan := make(chan struct{})
	defer close(reloadNotificationChan)

	go watchAndBuild(ctx, stop, reloadNotificationChan, websocketUrl)

	mux := http.NewServeMux()
	mux.HandleFunc("/", addLogging(http.FileServerFS(os.DirFS(configYaml.Output))))
	mux.HandleFunc(websocketPath, handleWebsocket(reloadNotificationChan))
	server := http.Server{
		Addr:    host,
		Handler: mux,
	}
	go func() {
		log.Printf("listening on http://%s", host)
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

func addLogging(handler http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s", req.Method, req.URL.Path)
		handler.ServeHTTP(rw, req)
	}
}

func handleWebsocket(reloadNotifcationChan <-chan struct{}) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(rw, req, nil)
		if err != nil {
			log.Printf("failed to accept: %s", err)
			return
		}
		defer conn.CloseNow()
		readCtx := conn.CloseRead(context.Background())

		for {
			select {
			case <-readCtx.Done():
				if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
					log.Printf("failed to close: %s", err)
				}
				return
			case <-reloadNotifcationChan:
				if err := conn.Write(context.Background(), websocket.MessageText, []byte("reload")); err != nil {
					log.Printf("failed to write: %s", err)
					return
				}
			}
		}
	}
}

func watchAndBuild(ctx context.Context, stop func(), reloadNotificationChan chan<- struct{}, websocketUrl *url.URL) {
	var watcher *fsnotify.Watcher
	var err error
	filePath := ""

	// initial build
	if err := build(); err != nil {
		log.Printf("failed to build: %s", err)
		return
	}
	if err := modifyHtmlFiles(websocketUrl); err != nil {
		log.Printf("failed to modify HTML files: %s", err)
		return
	}

forloop:
	for {
		// construct new directory watcher
		if watcher != nil {
			watcher.Close()
		}
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Printf("failed to construct watcher: %s", err)
			break forloop
		}
		if err := watcher.Add(configYaml.Templates); err != nil {
			log.Printf("failed to watch %s: %s", configYaml.Templates, err)
		}
		err = filepath.WalkDir(configYaml.Input, func(walkPath string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !dirEntry.IsDir() {
				return nil
			}
			if err := watcher.Add(walkPath); err != nil {
				return fmt.Errorf("failed to watch %s: %s", walkPath, err)
			}
			return nil
		})
		if err != nil {
			log.Println(err)
			break forloop
		}

		// wait for something to happen
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

		// rebuild
		log.Printf("build triggered by change to %s", filePath)
		if err := build(); err != nil {
			log.Printf("failed to build: %s", err)
			break forloop
		}
		if err := modifyHtmlFiles(websocketUrl); err != nil {
			log.Printf("failed to modify HTML files: %s", err)
			break forloop
		}
		reloadNotificationChan <- struct{}{}

		// avoid unnecessary rebuilds
		time.Sleep(100 * time.Millisecond)
	}

	// clean up
	if watcher != nil {
		watcher.Close()
	}
	stop()
}

func modifyHtmlFiles(websocketUrl *url.URL) error {
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

		if err := injectReloadScript(outputPath, websocketUrl); err != nil {
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
// Does not use golang.org/x/net/html because that package converts
// escaped HTML to the thing it represents (but only sometimes), and
// our needs are simple.
func injectReloadScript(filePath string, websocketUrl *url.URL) error {
	headEndRegex := regexp.MustCompile(`\<\/head\>`)
	htmlOpenRegex := regexp.MustCompile(`\<html\>`)
	reloadScript := getReloadScript(websocketUrl)

	byteContents, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}
	contents := string(byteContents)

	// Try to find the closing head element (i.e. </head>) and insert
	// the script immediately before it.
	loc := headEndRegex.FindStringIndex(contents)
	if loc != nil {
		before := contents[0:loc[0]]
		after := contents[loc[0]:]
		newContents := before + reloadScript + after
		if err := os.WriteFile(filePath, []byte(newContents), 0o644); err != nil {
			return fmt.Errorf("failed to write injected file: %w", err)
		}
		return nil
	}

	// Assume that <head>...</head> does not exist in the document. Try to
	// find the opening html element (i.e. <html>) and insert the script,
	// wrapped in <head>...</head>, immediately after it.
	loc = htmlOpenRegex.FindStringIndex(contents)
	if loc != nil {
		before := contents[0:loc[1]]
		after := contents[loc[1]:]
		newContents := before + `<head>` + reloadScript + `</head>` + after
		if err := os.WriteFile(filePath, []byte(newContents), 0o644); err != nil {
			return fmt.Errorf("failed to write injected file: %w", err)
		}
		return nil
	}

	return errors.New("failed to find index of <html> or </head>")

	// fd, err := os.OpenFile(filePath, os.O_RDWR, 0o644)
	// if err != nil {
	// 	return fmt.Errorf("failed to open: %w", err)
	// }
	// defer fd.Close()

	// document, err := html.Parse(fd)
	// if err != nil {
	// 	return fmt.Errorf("failed to parse: %w", err)
	// }
	// found := false
	// for node := range document.Descendants() {
	// 	if node.Type == html.ElementNode && node.DataAtom == atom.Head {
	// 		reloadScript := fmt.Sprintf(reloadScriptTemplate, websocketUrl.Host, websocketUrl.Path, reloadMsg)
	// 		scriptNode := &html.Node{
	// 			Type: html.RawNode,
	// 			Data: reloadScript,
	// 		}
	// 		node.AppendChild(scriptNode)
	// 		found = true
	// 		break
	// 	}
	// }
	// if !found {
	// 	return errors.New("failed to find head element")
	// }

	// if err := fd.Truncate(0); err != nil {
	// 	return fmt.Errorf("failed to truncate: %w", err)
	// }
	// if _, err := fd.Seek(0, io.SeekStart); err != nil {
	// 	return fmt.Errorf("failed to seek: %w", err)
	// }

	// if err := html.Render(fd, document); err != nil {
	// 	return fmt.Errorf("failed to render modified document: %w", err)
	// }

	// return nil
}

func getReloadScript(websocketUrl *url.URL) string {
	return fmt.Sprintf(reloadScriptTemplate, websocketUrl.Host, websocketUrl.Path, reloadMsg)
}
