// internal/app/app_test.go
package app

import (
	"testing"
	"time"

	"github.com/lc/pfzf/internal/config"
	"github.com/lc/pfzf/pkg/types"
)

type mockScanner struct {
	files []types.FileEntry
}

func (m *mockScanner) Scan(opts types.ScanOptions) (<-chan types.FileEntry, <-chan error) {
	filesChan := make(chan types.FileEntry)
	errChan := make(chan error)

	go func() {
		defer close(filesChan)
		defer close(errChan)

		for _, f := range m.files {
			filesChan <- f
		}
	}()

	return filesChan, errChan
}

func (m *mockScanner) Stop() {}

type mockProcessor struct{}

func (m *mockProcessor) Process(entry types.FileEntry) (types.ProcessedContent, error) {
	return types.ProcessedContent{
		Entry:   entry,
		Content: []byte("test content"),
	}, nil
}

func (m *mockProcessor) ShouldProcess(entry types.FileEntry) bool {
	return !entry.IsBinary
}

type mockWriter struct {
	written []types.ProcessedContent
}

func (m *mockWriter) Write(content types.ProcessedContent) error {
	m.written = append(m.written, content)
	return nil
}

func (m *mockWriter) Close() error {
	return nil
}

func TestApp(t *testing.T) {
	// Create test files
	testFiles := []types.FileEntry{
		{
			Path:     "test1.txt",
			Size:     100,
			ModTime:  time.Now(),
			IsBinary: false,
		},
		{
			Path:     "test2.txt",
			Size:     200,
			ModTime:  time.Now(),
			IsBinary: false,
		},
	}

	scanner := &mockScanner{files: testFiles}
	processor := &mockProcessor{}
	writer := &mockWriter{}

	app := New(config.DefaultConfig(), scanner, processor, writer)

	// Test file scanning
	if err := app.startScanning(); err != nil {
		t.Fatalf("Failed to start scanning: %v", err)
	}

	// Wait for scanning to complete
	time.Sleep(100 * time.Millisecond)

	// Verify files were added
	if len(app.entries) != len(testFiles) {
		t.Errorf("Expected %d entries, got %d", len(testFiles), len(app.entries))
	}

	// Test file selection
	app.toggleSelection(0)

	// Verify file was processed and written
	time.Sleep(100 * time.Millisecond)
	if len(writer.written) != 1 {
		t.Errorf("Expected 1 written file, got %d", len(writer.written))
	}
}
