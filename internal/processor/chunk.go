package processor

import (
	"bufio"
	"bytes"
	"unicode"

	"github.com/lc/pfzf/pkg/types"
)

// ChunkerOptions configures the behavior of content chunking.
type ChunkerOptions struct {
	// MaxSize is the maximum size of each chunk in bytes
	MaxSize int64
	// Overlap is the number of bytes to overlap between chunks
	Overlap int
	// MaxTokens is the maximum number of tokens per chunk (approximate)
	MaxTokens int
	// PreserveML determines if markup language tags should be preserved
	PreserveML bool
}

// Chunker handles content chunking operations.
type Chunker struct {
	opts ChunkerOptions
}

// NewChunker creates a new chunker with the given options.
func NewChunker(opts ChunkerOptions) *Chunker {
	return &Chunker{opts: opts}
}

// Chunk splits content into overlapping chunks while trying to maintain
// semantic boundaries (line breaks, sentences, paragraphs).
func (c *Chunker) Chunk(content []byte) ([]types.Chunk, error) {
	if len(content) == 0 {
		return nil, nil
	}

	if c.opts.MaxSize > 0 && int64(len(content)) <= c.opts.MaxSize {
		return []types.Chunk{{
			Content:    append(bytes.TrimSpace(content), '\n'),
			StartLine:  1,
			EndLine:    1,
			TokenCount: c.countTokens(string(content)),
		}}, nil
	}

	var chunks []types.Chunk
	pos := int64(0)
	contentLen := int64(len(content))

	for pos < contentLen {
		chunkSize := c.opts.MaxSize
		if pos+chunkSize > contentLen {
			chunkSize = contentLen - pos
		}

		// Create chunk
		chunkContent := content[pos : pos+chunkSize]
		chunk := types.Chunk{
			Content:    append(bytes.TrimSpace(chunkContent), '\n'),
			StartLine:  1,
			EndLine:    1,
			TokenCount: c.countTokens(string(chunkContent)),
		}
		chunks = append(chunks, chunk)

		// Move position forward
		advance := chunkSize - int64(c.opts.Overlap)
		if advance < 1 {
			advance = 1
		}
		pos += advance

		// Handle remaining content
		if pos < contentLen && contentLen-pos <= int64(c.opts.Overlap) {
			finalContent := content[pos:]
			// Always add the final chunk if there's content remaining
			if len(finalContent) > 0 {
				chunks = append(chunks, types.Chunk{
					Content:    append(bytes.TrimSpace(finalContent), '\n'),
					StartLine:  1,
					EndLine:    1,
					TokenCount: c.countTokens(string(finalContent)),
				})
			}
			break
		}
	}

	return chunks, nil
}

// chunkSingleLine handles chunking of a single line of content
func (c *Chunker) chunkSingleLine(content []byte) ([]types.Chunk, error) {
	chunks := make([]types.Chunk, 0)
	length := len(content)

	// If content is smaller than max size, return single chunk
	if int64(length) <= c.opts.MaxSize {
		return []types.Chunk{{
			Content:    append(content, '\n'),
			StartLine:  1,
			EndLine:    1,
			TokenCount: c.countTokens(string(content)),
		}}, nil
	}

	pos := 0
	for pos < length {
		// Calculate end position for this chunk
		end := pos + int(c.opts.MaxSize)
		if end > length {
			end = length
		}

		// Create chunk with single newline at end
		chunk := types.Chunk{
			Content:    append(bytes.TrimSpace(content[pos:end]), '\n'),
			StartLine:  1,
			EndLine:    1,
			TokenCount: c.countTokens(string(content[pos:end])),
		}
		chunks = append(chunks, chunk)

		// Move position forward by (maxSize - overlap)
		pos += int(c.opts.MaxSize) - c.opts.Overlap

		// Ensure we make progress
		if pos <= 0 {
			pos = 1
		}

		// If remaining content is smaller than overlap, we're done
		if length-pos <= c.opts.Overlap {
			if pos < length {
				// Add final chunk if there's remaining content
				chunks = append(chunks, types.Chunk{
					Content:    append(bytes.TrimSpace(content[pos:]), '\n'),
					StartLine:  1,
					EndLine:    1,
					TokenCount: c.countTokens(string(content[pos:])),
				})
			}
			break
		}
	}

	return chunks, nil
}

// getSplitFunc returns a split function based on content type.
func (c *Chunker) getSplitFunc() bufio.SplitFunc {
	if c.opts.PreserveML {
		return c.splitMarkup
	}
	return bufio.ScanLines
}

// splitMarkup is a custom split function that preserves markup tags.
func (c *Chunker) splitMarkup(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Handle tags specially
	if data[0] == '<' {
		if i := bytes.IndexByte(data, '>'); i >= 0 {
			// Include newline if it follows the tag
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[0 : i+1], nil
			}
			return i + 1, data[0 : i+1], nil
		}
	}

	// Otherwise split on newlines
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// shouldStartNewChunk determines if a new chunk should be started.
func (c *Chunker) shouldStartNewChunk(currentSize, newTokens, currentTokens int) bool {
	// Always start a new chunk if we exceed MaxSize
	if c.opts.MaxSize > 0 && int64(currentSize) >= c.opts.MaxSize {
		return true
	}

	// Always start a new chunk if we exceed MaxTokens
	if c.opts.MaxTokens > 0 && currentTokens+newTokens > c.opts.MaxTokens {
		return true
	}

	return false
}

// countTokens provides a rough estimate of token count.
func (c *Chunker) countTokens(text string) int {
	var count int
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else {
			if !inWord {
				count++
				inWord = true
			}
		}
	}

	return count
}

// countLines counts the number of lines in the text.
func (c *Chunker) countLines(text []byte) int {
	count := 0
	for _, b := range text {
		if b == '\n' {
			count++
		}
	}
	return count + 1 // Add one for the last line if it doesn't end with newline
}

// Rename the existing getOverlap method to getOverlapContent
func (c *Chunker) getOverlapContent(content []byte) []byte {
	if c.opts.Overlap == 0 || len(content) == 0 {
		return nil
	}

	// Find a suitable boundary for overlap
	start := len(content) - c.opts.Overlap
	if start < 0 {
		start = 0
	}

	// Try to find a line break
	for i := start; i < len(content); i++ {
		if content[i] == '\n' {
			start = i + 1
			break
		}
	}

	return content[start:]
}
