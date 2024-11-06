// internal/writer/writer_test.go
package writer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lc/pfzf/pkg/types"
)

func TestWriter(t *testing.T) {
	testCases := []struct {
		name   string
		format types.OutputFormat
	}{
		{"XML", types.OutputFormatXML},
		{"JSON", types.OutputFormatJSON},
		{"YAML", types.OutputFormatYAML},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary output file
			tmpFile := filepath.Join(t.TempDir(), "test_output")

			opts := types.WriterOptions{
				OutputPath:  tmpFile,
				Format:      tc.format,
				PrettyPrint: true,
			}

			writer, err := New(opts)
			if err != nil {
				t.Fatalf("Failed to create writer: %v", err)
			}

			// Test writing content
			content := types.ProcessedContent{
				Entry: types.FileEntry{
					Path:     "test.txt",
					Size:     100,
					ModTime:  time.Now(),
					IsBinary: false,
				},
				Content: []byte("test content"),
			}

			if err := writer.Write(content); err != nil {
				t.Errorf("Failed to write content: %v", err)
			}

			if err := writer.Close(); err != nil {
				t.Errorf("Failed to close writer: %v", err)
			}

			// Verify file exists and has content
			if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
				t.Error("Output file was not created")
			}

			data, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Errorf("Failed to read output file: %v", err)
			}

			if len(data) == 0 {
				t.Error("Output file is empty")
			}
		})
	}
}
