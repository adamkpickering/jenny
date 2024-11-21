package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectReloadScript(t *testing.T) {
	t.Run("should inject script when head element is present", func(t *testing.T) {
		testDir := t.TempDir()
		testFilePath := filepath.Join(testDir, "testFile.html")
		contentTemplate := `<!DOCTYPE html><html><head><link rel="stylesheet" href="/static/style.css"/>%s</head></html>`
		testContents := fmt.Sprintf(contentTemplate, "")
		if err := os.WriteFile(testFilePath, []byte(testContents), 0o644); err != nil {
			t.Fatalf("failed to write test file: %s", err)
		}
		websocketUrl := &url.URL{
			Path: "/websocket",
			Host: "localhost:1234",
		}
		if err := injectReloadScript(testFilePath, websocketUrl); err != nil {
			t.Fatalf("unexpected error in injectReloadScript(): %s", err)
		}
		newByteContents, err := os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("failed to read modified test file: %s", err)
		}
		newContents := string(newByteContents)
		expectedContents := fmt.Sprintf(contentTemplate, getReloadScript(websocketUrl))
		if newContents != expectedContents {
			t.Errorf("got contents %q but expected contents %q", newContents, expectedContents)
		}
	})

	t.Run("should inject script when head element is not present but html element is present", func(t *testing.T) {
		testDir := t.TempDir()
		testFilePath := filepath.Join(testDir, "testFile.html")
		contentTemplate := `<!DOCTYPE html><html>%s</html>`
		testContents := fmt.Sprintf(contentTemplate, "")
		if err := os.WriteFile(testFilePath, []byte(testContents), 0o644); err != nil {
			t.Fatalf("failed to write test file: %s", err)
		}
		websocketUrl := &url.URL{
			Path: "/websocket",
			Host: "localhost:1234",
		}
		if err := injectReloadScript(testFilePath, websocketUrl); err != nil {
			t.Fatalf("unexpected error in injectReloadScript(): %s", err)
		}
		newByteContents, err := os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("failed to read modified test file: %s", err)
		}
		newContents := string(newByteContents)
		expectedContents := fmt.Sprintf(contentTemplate, `<head>`+getReloadScript(websocketUrl)+`</head>`)
		if newContents != expectedContents {
			t.Errorf("got contents %q but expected contents %q", newContents, expectedContents)
		}
	})

	t.Run("should return error when neither html element nor head element is present", func(t *testing.T) {
		testDir := t.TempDir()
		testFilePath := filepath.Join(testDir, "testFile.html")
		testContents := `<!DOCTYPE html><body><p>here is the body but not the tags we are looking for</p></body>`
		if err := os.WriteFile(testFilePath, []byte(testContents), 0o644); err != nil {
			t.Fatalf("failed to write test file: %s", err)
		}
		websocketUrl := &url.URL{
			Path: "/websocket",
			Host: "localhost:1234",
		}

		err := injectReloadScript(testFilePath, websocketUrl)
		if err == nil {
			t.Fatalf("did not get error from injectReloadScript when we should have")
		}
		expectedError := "failed to find index of <html> or </head>"
		if err.Error() != expectedError {
			t.Fatalf("returned error %q did not match expected error %q", err.Error(), expectedError)
		}
	})
}
