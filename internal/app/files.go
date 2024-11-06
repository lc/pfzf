package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/lc/pfzf/pkg/types"
	"github.com/sahilm/fuzzy"
)

const (
	previewChunkSize = 16 * 1024 // 16KB chunks
	previewMaxLines  = 1000      // Maximum lines to show
	previewContext   = 5         // Context lines around search
)

func (a *App) startScanning() error {
	scanOpts := types.ScanOptions{
		RootDir:       ".",
		IgnorePattern: a.config.Scanner.IgnorePatterns,
		MaxFileSize:   a.config.Scanner.MaxFileSize,
		MaxFiles:      a.config.Scanner.MaxFiles,
	}

	filesChan, errChan := a.scanner.Scan(scanOpts)

	// Handle incoming files
	go func() {
		for {
			select {
			case entry, ok := <-filesChan:
				if !ok {
					return
				}
				a.addEntry(entry)
			case err := <-errChan:
				if err != nil {
					a.updateStatus(fmt.Sprintf("Error scanning: %v", err))
				}
			case <-a.ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (a *App) addEntry(entry types.FileEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.entries = append(a.entries, entry)
	a.QueueUpdateDraw(func() {
		a.updateFileList()
	})
}

func (a *App) toggleSelection(idx int) {
	if idx < 0 || idx >= len(a.entries) {
		return
	}

	a.mu.Lock()
	currentItem := a.fileList.GetCurrentItem()
	entry := a.entries[idx]
	entry.IsSelected = !entry.IsSelected
	a.entries[idx] = entry
	a.mu.Unlock()

	if entry.IsSelected {
		go a.processAndWriteEntry(entry)
	} else {
		// Remove from writer when deselected
		a.writer.Remove(entry.Path)
	}

	a.updateFileListPreserveSelection(currentItem)
}

// updateFileListPreserveSelection updates the list while preserving selection
func (a *App) updateFileListPreserveSelection(currentItem int) {
	a.fileList.Clear()
	a.filteredIdx = make([]int, 0)

	if a.searchString == "" {
		// Show all entries
		for i, entry := range a.entries {
			a.filteredIdx = append(a.filteredIdx, i)
			a.fileList.AddItem(a.formatListItem(entry), "", 0, nil)
		}
	} else {
		// Perform fuzzy search
		patterns := make([]string, len(a.entries))
		for i, entry := range a.entries {
			patterns[i] = entry.Path
		}

		matches := fuzzy.Find(a.searchString, patterns)
		for _, match := range matches {
			a.filteredIdx = append(a.filteredIdx, match.Index)
			a.fileList.AddItem(a.formatListItem(a.entries[match.Index]), "", 0, nil)
		}
	}

	// Restore the selection
	if currentItem >= 0 && currentItem < a.fileList.GetItemCount() {
		a.fileList.SetCurrentItem(currentItem)
	}
}

func (a *App) processAndWriteEntry(entry types.FileEntry) {
	processed, err := a.processor.Process(entry)
	if err != nil {
		a.updateStatus(fmt.Sprintf("Error processing %s: %v", entry.Path, err))
		return
	}

	if err := a.writer.Write(processed); err != nil {
		a.updateStatus(fmt.Sprintf("Error writing %s: %v", entry.Path, err))
		return
	}

	a.updateStatus(fmt.Sprintf("Added %s to context", entry.Path))
}

func (a *App) updateStatus(msg string) {
	a.QueueUpdateDraw(func() {
		a.status.SetText(msg)
	})
}

// handleSearch processes search input and updates the UI accordingly
func (a *App) handleSearch(text string) {
	a.mu.Lock()
	a.searchString = text
	a.mu.Unlock()

	// Clear filtered indices
	a.filteredIdx = a.filteredIdx[:0]

	if text == "" {
		// If search is empty, show all files
		a.filteredIdx = make([]int, len(a.entries))
		for i := range a.entries {
			a.filteredIdx[i] = i
		}
	} else {
		// Filter files based on search
		for i, entry := range a.entries {
			if strings.Contains(strings.ToLower(entry.Path), strings.ToLower(text)) {
				a.filteredIdx = append(a.filteredIdx, i)
			}
		}
	}

	// Update UI
	a.updateFileList()

	// Clear preview if no matches
	if len(a.filteredIdx) == 0 {
		a.preview.Clear()
		a.status.SetText("No matches found")
		return
	}

	// Update preview for first match if any exist
	if len(a.filteredIdx) > 0 {
		a.handleSelection(0)
	}
}

func (a *App) updateFileList() {
	a.fileList.Clear()
	a.filteredIdx = make([]int, 0)

	if a.searchString == "" {
		// Show all entries
		for i, entry := range a.entries {
			a.filteredIdx = append(a.filteredIdx, i)
			a.fileList.AddItem(a.formatListItem(entry), "", 0, nil)
		}
		return
	}

	// Perform fuzzy search
	patterns := make([]string, len(a.entries))
	for i, entry := range a.entries {
		patterns[i] = entry.Path
	}

	matches := fuzzy.Find(a.searchString, patterns)
	for _, match := range matches {
		a.filteredIdx = append(a.filteredIdx, match.Index)
		a.fileList.AddItem(a.formatListItem(a.entries[match.Index]), "", 0, nil)
	}
}

func (a *App) formatListItem(entry types.FileEntry) string {
	prefix := map[bool]string{true: "[x]", false: "[ ]"}[entry.IsSelected]
	return fmt.Sprintf("%s %s", prefix, entry.Path)
}

func (a *App) handleSelection(index int) {
	if index >= 0 && index < len(a.filteredIdx) {
		entry := a.entries[a.filteredIdx[index]]
		a.showPreview(entry)
	}
}

// PreviewState tracks preview pane state
type PreviewState struct {
	filename    string
	offset      int64
	lines       []string
	currentLine int
	totalLines  int
	searchMatch []int
	isDirty     bool
}

// previewBuffer manages the preview content
type previewBuffer struct {
	mu      sync.RWMutex
	content []string
	size    int
}

func newPreviewBuffer() *previewBuffer {
	return &previewBuffer{
		content: make([]string, 0, previewMaxLines),
	}
}

func (pb *previewBuffer) append(lines []string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// If we would exceed max lines, remove oldest lines
	if len(pb.content)+len(lines) > previewMaxLines {
		excess := len(pb.content) + len(lines) - previewMaxLines
		pb.content = pb.content[excess:]
	}

	pb.content = append(pb.content, lines...)
	pb.size += len(lines)
}

func (pb *previewBuffer) get() []string {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.content
}

func (pb *previewBuffer) clear() {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.content = pb.content[:0]
	pb.size = 0
}

func (a *App) showPreview(entry types.FileEntry) {
	if entry.IsBinary {
		a.preview.SetText("Binary file - preview not available")
		return
	}

	// Create new preview state
	state := &PreviewState{
		filename: entry.Path,
		isDirty:  true,
	}

	// Start preview in background
	go a.loadPreview(state)
}

func (a *App) loadPreview(state *PreviewState) {
	f, err := os.Open(state.filename)
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.preview.SetText(fmt.Sprintf("Error opening file: %v", err))
		})
		return
	}
	defer f.Close()

	buffer := newPreviewBuffer()
	reader := bufio.NewReader(f)
	lineCount := 0

	// Read file in chunks
	for lineCount < previewMaxLines {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			a.QueueUpdateDraw(func() {
				a.preview.SetText(fmt.Sprintf("Error reading file: %v", err))
			})
			return
		}

		buffer.append([]string{strings.TrimRight(line, "\n")})
		lineCount++

		// Update preview periodically
		if lineCount%100 == 0 {
			a.updatePreviewContent(buffer.get(), state)
		}
	}

	// Final update
	a.updatePreviewContent(buffer.get(), state)
}

func (a *App) updatePreviewContent(lines []string, state *PreviewState) {
	state.lines = lines
	state.totalLines = len(lines)

	// Find search matches if search is active
	if a.searchString != "" {
		state.searchMatch = a.findSearchMatches(lines, a.searchString)
		if len(state.searchMatch) > 0 && state.currentLine == 0 {
			state.currentLine = state.searchMatch[0]
		}
	}

	a.QueueUpdateDraw(func() {
		a.renderPreview(state)
		a.updatePreviewStatus(state)
	})
}

func (a *App) renderPreview(state *PreviewState) {
	var preview strings.Builder

	// Calculate visible range
	visibleLines := min(len(state.lines), previewMaxLines)
	start := max(0, state.currentLine-previewContext)
	end := min(visibleLines, start+previewMaxLines)

	// Add file info header
	fmt.Fprintf(&preview, "[yellow]%s (%d/%d lines)[white]\n",
		state.filename, visibleLines, state.totalLines)

	// Render visible lines
	for i := start; i < end; i++ {
		line := state.lines[i]

		// Highlight current line
		prefix := "  "
		if i == state.currentLine {
			prefix = "> "
		}

		// Highlight search matches
		if a.searchString != "" && strings.Contains(
			strings.ToLower(line),
			strings.ToLower(a.searchString)) {
			line = fmt.Sprintf("[red]%s[white]", line)
		}

		fmt.Fprintf(&preview, "%s[dimgray]%4d[white] %s\n",
			prefix, i+1, line)
	}

	a.preview.SetText(preview.String())
}

func (a *App) updatePreviewStatus(state *PreviewState) {
	if state == nil {
		a.status.SetText("No preview available")
		return
	}

	status := fmt.Sprintf(
		"Preview: Line %d/%d | %d matches",
		state.currentLine+1,
		state.totalLines,
		len(state.searchMatch),
	)
	a.status.SetText(status)
}

func (a *App) findSearchMatches(lines []string, search string) []int {
	var matches []int
	searchLower := strings.ToLower(search)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), searchLower) {
			matches = append(matches, i)
		}
	}
	return matches
}

func (a *App) scrollToTop() {
	a.preview.ScrollTo(0, 0)
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
