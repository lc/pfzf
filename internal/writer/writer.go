// Package writer handles output file generation in various formats.
package writer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/lc/pfzf/pkg/types"
	"gopkg.in/yaml.v3"
)

// FileWriter manages writing processed content to a file in various formats.
type FileWriter struct {
	opts      types.WriterOptions
	file      io.WriteCloser
	mu        sync.Mutex
	initOnce  sync.Once
	initError error
	buffer    map[string]types.ProcessedContent
}

// New creates a new FileWriter without immediately creating the output file.
func New(opts types.WriterOptions) (*FileWriter, error) {
	if opts.OutputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}

	return &FileWriter{
		opts:   opts,
		buffer: make(map[string]types.ProcessedContent),
	}, nil
}

// initialize creates the output file and writes initial format headers.
func (w *FileWriter) initialize() error {
	var err error
	w.initOnce.Do(func() {
		var f *os.File
		f, err = os.Create(w.opts.OutputPath)
		if err != nil {
			err = fmt.Errorf("creating output file: %w", err)
			return
		}
		w.file = f

		// Write format-specific headers
		switch w.opts.Format {
		case types.OutputFormatXML:
			_, err = io.WriteString(f, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<files>\n")
		case types.OutputFormatJSON:
			_, err = io.WriteString(f, "{\n")
		case types.OutputFormatYAML:
			_, err = io.WriteString(f, "---\n")
		default:
			err = fmt.Errorf("unsupported format: %s", w.opts.Format)
		}

		if err != nil {
			f.Close()
			err = fmt.Errorf("writing format header: %w", err)
			return
		}
	})

	if err != nil {
		w.initError = err
		return err
	}

	return w.initError
}

// Write buffers content instead of writing immediately.
func (w *FileWriter) Write(content types.ProcessedContent) error {
	if content.Entry.Path == "" {
		return fmt.Errorf("content path cannot be empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer[content.Entry.Path] = content
	return nil
}

// Remove removes content from the buffer.
func (w *FileWriter) Remove(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.buffer, path)
}

// Flush writes all buffered content to file.
func (w *FileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Don't create file if nothing to write
	if len(w.buffer) == 0 {
		return nil
	}

	if err := w.initialize(); err != nil {
		return fmt.Errorf("initializing writer: %w", err)
	}

	// Write buffered content based on format
	switch w.opts.Format {
	case types.OutputFormatXML:
		return w.flushXML()
	case types.OutputFormatJSON:
		return w.flushJSON()
	case types.OutputFormatYAML:
		return w.flushYAML()
	default:
		return fmt.Errorf("unsupported format: %s", w.opts.Format)
	}
}

func (w *FileWriter) flushXML() error {
	for _, content := range w.buffer {
		if _, err := fmt.Fprintf(w.file,
			"<file>\n  <path>%s</path>\n  <content><![CDATA[\n%s\n]]></content>\n</file>\n",
			content.Entry.Path,
			content.Content); err != nil {
			return fmt.Errorf("writing XML content: %w", err)
		}
	}
	return nil
}

func (w *FileWriter) flushJSON() error {
	encoder := json.NewEncoder(w.file)
	if w.opts.PrettyPrint {
		encoder.SetIndent("", "  ")
	}

	// Write files array opening
	if _, err := io.WriteString(w.file, "\"files\": [\n"); err != nil {
		return fmt.Errorf("writing JSON array opening: %w", err)
	}

	first := true
	for _, content := range w.buffer {
		if !first {
			if _, err := io.WriteString(w.file, ",\n"); err != nil {
				return fmt.Errorf("writing JSON separator: %w", err)
			}
		}
		first = false

		if err := encoder.Encode(struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}{
			Path:    content.Entry.Path,
			Content: string(content.Content),
		}); err != nil {
			return fmt.Errorf("encoding JSON content: %w", err)
		}
	}

	return nil
}

func (w *FileWriter) flushYAML() error {
	encoder := yaml.NewEncoder(w.file)
	for _, content := range w.buffer {
		if err := encoder.Encode(struct {
			Path    string `yaml:"path"`
			Content string `yaml:"content"`
		}{
			Path:    content.Entry.Path,
			Content: string(content.Content),
		}); err != nil {
			return fmt.Errorf("encoding YAML content: %w", err)
		}
	}
	return nil
}

// WriteDirectoryContext writes the directory context information.
func (w *FileWriter) WriteDirectoryContext(cwd, tree string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.initialize(); err != nil {
		return fmt.Errorf("initializing writer: %w", err)
	}

	switch w.opts.Format {
	case types.OutputFormatXML:
		_, err := fmt.Fprintf(w.file,
			"<directory-context>\n  <cwd>%s</cwd>\n  <tree><![CDATA[\n%s\n]]></tree>\n</directory-context>\n",
			cwd, tree)
		if err != nil {
			return fmt.Errorf("writing XML directory context: %w", err)
		}

	case types.OutputFormatJSON:
		if _, err := io.WriteString(w.file, "\"directory_context\": {\n"); err != nil {
			return fmt.Errorf("writing JSON context opening: %w", err)
		}

		encoder := json.NewEncoder(w.file)
		if w.opts.PrettyPrint {
			encoder.SetIndent("  ", "  ")
		}

		if err := encoder.Encode(struct {
			CWD  string `json:"cwd"`
			Tree string `json:"tree"`
		}{
			CWD:  cwd,
			Tree: tree,
		}); err != nil {
			return fmt.Errorf("encoding JSON directory context: %w", err)
		}

		if _, err := io.WriteString(w.file, "},\n"); err != nil {
			return fmt.Errorf("writing JSON context closing: %w", err)
		}

	case types.OutputFormatYAML:
		encoder := yaml.NewEncoder(w.file)
		if err := encoder.Encode(map[string]interface{}{
			"directory_context": struct {
				CWD  string `yaml:"cwd"`
				Tree string `yaml:"tree"`
			}{
				CWD:  cwd,
				Tree: tree,
			},
		}); err != nil {
			return fmt.Errorf("encoding YAML directory context: %w", err)
		}

	default:
		return fmt.Errorf("unsupported format: %s", w.opts.Format)
	}

	return nil
}

// Close properly closes the file if it was created.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	var err error
	switch w.opts.Format {
	case types.OutputFormatXML:
		_, err = io.WriteString(w.file, "</files>")
	case types.OutputFormatJSON:
		_, err = io.WriteString(w.file, "\n]}")
	}

	if err != nil {
		w.file.Close()
		return fmt.Errorf("writing closing tags: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	return nil
}
