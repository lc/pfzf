// Package processor provides content processing capabilities for pfzf.
package processor

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lc/pfzf/pkg/types"
)

// DefaultChunkSize is the default size for content chunks.
const DefaultChunkSize = 4096

// Processor implements the types.Processor interface.
type Processor struct {
	opts     types.ProcessorOptions
	language *LanguageDetector
}

// New creates a new Processor with the given options.
func New(opts types.ProcessorOptions) (*Processor, error) {
	if opts.MaxChunkSize <= 0 {
		opts.MaxChunkSize = DefaultChunkSize
	}

	detector, err := NewLanguageDetector()
	if err != nil {
		return nil, fmt.Errorf("creating language detector: %w", err)
	}

	return &Processor{
		opts:     opts,
		language: detector,
	}, nil
}

// Process implements types.Processor.Process.
func (p *Processor) Process(entry types.FileEntry) (types.ProcessedContent, error) {
	if !p.ShouldProcess(entry) {
		return types.ProcessedContent{Entry: entry}, nil
	}

	// Read file content
	content, err := os.ReadFile(entry.Path)
	if err != nil {
		return types.ProcessedContent{}, fmt.Errorf("reading file: %w", err)
	}

	// Detect language if not already set
	if entry.Language == "" {
		lang, err := p.language.DetectLanguage(entry.Path, bytes.NewReader(content))
		if err != nil {
			// Don't fail on language detection errors
			entry.Language = "unknown"
		} else {
			entry.Language = lang
		}
	}

	// Process content based on options
	processed := types.ProcessedContent{
		Entry:   entry,
		Content: content,
	}

	// Strip comments if requested and language is supported
	if p.opts.StripComments {
		stripped, err := p.stripComments(content, entry.Language)
		if err == nil { // Only use stripped content if successful
			processed.Content = stripped
		}
	}

	// Create chunks if content exceeds chunk size
	if int64(len(content)) > p.opts.MaxChunkSize {
		chunks, err := p.createChunks(processed.Content)
		if err != nil {
			return types.ProcessedContent{}, fmt.Errorf("creating chunks: %w", err)
		}
		processed.Chunks = chunks
	}

	return processed, nil
}

// ShouldProcess implements types.Processor.ShouldProcess.
func (p *Processor) ShouldProcess(entry types.FileEntry) bool {
	// Don't process binary files
	if entry.IsBinary {
		return false
	}

	// Don't process empty files
	if entry.Size == 0 {
		return false
	}

	// Don't process files larger than max tokens (rough estimate)
	if p.opts.MaxTokens > 0 && entry.Size > int64(p.opts.MaxTokens*4) {
		return false
	}

	return true
}

// stripComments removes comments from the content based on the language.
func (p *Processor) stripComments(content []byte, language string) ([]byte, error) {
	stripper, err := p.language.GetCommentStripper(language)
	if err != nil {
		return nil, err
	}
	return stripper.StripComments(content)
}

// createChunks splits content into overlapping chunks.
func (p *Processor) createChunks(content []byte) ([]types.Chunk, error) {
	chunker := NewChunker(ChunkerOptions{
		MaxSize:    p.opts.MaxChunkSize,
		Overlap:    p.opts.ChunkOverlap,
		MaxTokens:  p.opts.MaxTokens,
		PreserveML: true, // Preserve markup language tags
	})

	return chunker.Chunk(content)
}

// Configure updates the processor options.
func (p *Processor) Configure(opts types.ProcessorOptions) {
	if opts.MaxChunkSize > 0 {
		p.opts.MaxChunkSize = opts.MaxChunkSize
	}
	if opts.ChunkOverlap >= 0 {
		p.opts.ChunkOverlap = opts.ChunkOverlap
	}
	if opts.MaxTokens > 0 {
		p.opts.MaxTokens = opts.MaxTokens
	}
	p.opts.StripComments = opts.StripComments
}
