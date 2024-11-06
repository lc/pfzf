// Package types provides the core types used throughout the pfzf utility.
package types

import (
	"io"
	"time"
)

// FileEntry represents a file in the workspace with its metadata.
type FileEntry struct {
	Path       string
	Size       int64
	ModTime    time.Time
	IsSelected bool
	IsBinary   bool
	Language   string
}

// ProcessedContent represents processed file content ready for output.
type ProcessedContent struct {
	Entry   FileEntry
	Content []byte
	Chunks  []Chunk
}

// Chunk represents a segment of file content.
type Chunk struct {
	Content    []byte
	StartLine  int
	EndLine    int
	TokenCount int
}

// Scanner defines the interface for file scanning operations.
type Scanner interface {
	// Scan starts scanning the workspace and returns a channel of FileEntry.
	// The channel is closed when scanning is complete or an error occurs.
	Scan(opts ScanOptions) (<-chan FileEntry, <-chan error)

	// Stop terminates the current scanning operation.
	Stop()
}

// ScanOptions configures the scanning behavior.
type ScanOptions struct {
	RootDir       string
	IgnorePattern []string
	MaxFileSize   int64
	MaxFiles      int
}

// Processor defines the interface for content processing operations.
type Processor interface {
	// Process processes a file entry and returns the processed content.
	Process(entry FileEntry) (ProcessedContent, error)

	// ShouldProcess determines if a file should be processed based on its metadata.
	ShouldProcess(entry FileEntry) bool
}

// ProcessorOptions configures the processing behavior.
type ProcessorOptions struct {
	MaxChunkSize  int64
	ChunkOverlap  int
	MaxTokens     int
	StripComments bool
}

// Writer defines the interface for output writing operations.
type Writer interface {
	// Write writes processed content to the output destination.
	Write(content ProcessedContent) error

	// WriteDirectoryContext writes the directory context information.
	WriteDirectoryContext(cwd string, tree string) error

	// Flush flushes any buffered data to the output.
	Flush() error

	// Remove removes the file at the specified path.
	Remove(path string)
	// Close finalizes the output and closes any open resources.
	Close() error
}

// WriterOptions configures the output writing behavior.
type WriterOptions struct {
	OutputPath  string
	Format      OutputFormat
	PrettyPrint bool
}

// OutputFormat represents the supported output formats.
type OutputFormat string

const (
	// OutputFormatXML represents XML output format.
	OutputFormatXML OutputFormat = "xml"
	// OutputFormatJSON represents JSON output format.
	OutputFormatJSON OutputFormat = "json"
	// OutputFormatYAML represents YAML output format.
	OutputFormatYAML OutputFormat = "yaml"
)

// LanguageProcessor defines the interface for language-specific processing.
type LanguageProcessor interface {
	// DetectLanguage attempts to detect the programming language of a file.
	DetectLanguage(filename string, reader io.Reader) (string, error)

	// ExtractSymbols extracts language-specific symbols (functions, classes, etc.).
	ExtractSymbols(content []byte) ([]Symbol, error)

	// StripComments removes comments from the source code.
	StripComments(content []byte) ([]byte, error)
}

// Symbol represents a programming language symbol (function, class, etc.).
type Symbol struct {
	Name      string
	Type      string
	StartLine int
	EndLine   int
	Content   string
}
