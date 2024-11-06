package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lc/pfzf/pkg/types"
)

func TestScanner(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "pfzf-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string][]byte{
		"test.txt":     []byte("Hello, World!"),
		"test.bin":     {0x00, 0x01, 0x02, 0x03},
		".gitignore":   []byte("*.log\n*.tmp"),
		"ignored/test": []byte("ignored"),
		"src/main.go":  []byte("package main\n\nfunc main() {}\n"),
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name        string
		opts        []Option
		scanOpts    types.ScanOptions
		wantFiles   []string
		wantErrors  int
		checkBinary bool
	}{
		{
			name: "basic scan",
			scanOpts: types.ScanOptions{
				RootDir:     tmpDir,
				MaxFileSize: 1 << 20,
			},
			wantFiles: []string{
				"test.txt",
				"test.bin",
				".gitignore",
				"ignored/test",
				"src/main.go",
			},
			wantErrors: 0,
		},
		{
			name: "with ignore pattern",
			scanOpts: types.ScanOptions{
				RootDir:       tmpDir,
				IgnorePattern: []string{"*.bin", "ignored/*"},
			},
			wantFiles: []string{
				"test.txt",
				".gitignore",
				"src/main.go",
			},
			wantErrors: 0,
		},
		{
			name: "with small max size",
			scanOpts: types.ScanOptions{
				RootDir:     tmpDir,
				MaxFileSize: 5,
			},
			wantFiles: []string{
				"test.bin",
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.opts...)
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}

			results, errs := s.Scan(tt.scanOpts)

			var files []types.FileEntry
			var errors []error

			done := make(chan struct{})
			go func() {
				defer close(done)
				for entry := range results {
					files = append(files, entry)
				}
				for err := range errs {
					errors = append(errors, err)
				}
			}()

			// Wait for scan to complete with timeout
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Scanner timed out")
			}

			if len(errors) != tt.wantErrors {
				t.Errorf("Got %d errors, want %d", len(errors), tt.wantErrors)
			}

			foundFiles := make(map[string]bool)
			for _, f := range files {
				foundFiles[f.Path] = true
			}

			for _, want := range tt.wantFiles {
				if !foundFiles[want] {
					t.Errorf("Missing expected file: %s", want)
				}
			}

			t.Log(tt.name)
			if len(files) != len(tt.wantFiles) {
				for _, f := range tt.wantFiles {
					t.Logf("Want file: %s", f)
				}
				for _, f := range files {
					t.Logf("Found file: %s", f.Path)
				}
				t.Errorf("Got %d files, want %d", len(files), len(tt.wantFiles))
			}

			if tt.checkBinary {
				for _, f := range files {
					if filepath.Base(f.Path) == "test.bin" && !f.IsBinary {
						t.Error("Binary file not detected")
					}
					if filepath.Base(f.Path) == "test.txt" && f.IsBinary {
						t.Error("Text file incorrectly marked as binary")
					}
				}
			}
		})
	}
}

func TestScannerStop(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	results, errs := s.Scan(types.ScanOptions{RootDir: "."})

	// Start consuming results
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range results {
		}
		for range errs {
		}
	}()

	// Stop scanner immediately
	s.Stop()

	// Wait for channels to close
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Scanner did not stop in time")
	}
}
