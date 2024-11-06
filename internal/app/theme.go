package app

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ThemeManager handles the application's visual styling
type ThemeManager struct {
	app    *App
	colors map[string]tcell.Color
}

// newThemeManager creates a new theme manager
func newThemeManager(app *App) *ThemeManager {
	tm := &ThemeManager{
		app:    app,
		colors: make(map[string]tcell.Color),
	}
	return tm
}

// applyTheme applies the current theme to all UI components
func (tm *ThemeManager) applyTheme(theme map[string]string) {
	// Convert theme string colors to tcell.Color using tcell's GetColor
	for key, colorName := range theme {
		tm.colors[key] = tcell.GetColor(colorName)
	}

	// Apply colors to components
	tm.applyListColors()
	tm.applyPreviewColors()
	tm.applyStatusColors()
	tm.applySearchColors()
}

// applyListColors applies theme colors to the file list
func (tm *ThemeManager) applyListColors() {
	tm.app.fileList.SetBackgroundColor(tm.getColor("background", tcell.ColorDefault))
	tm.app.fileList.SetMainTextColor(tm.getColor("foreground", tcell.ColorWhite))
	tm.app.fileList.SetSelectedBackgroundColor(tm.getColor("selection", tcell.ColorBlue))
	tm.app.fileList.SetSelectedTextColor(tm.getColor("foreground", tcell.ColorWhite))
}

// applyPreviewColors applies theme colors to the preview pane
func (tm *ThemeManager) applyPreviewColors() {
	tm.app.preview.SetBackgroundColor(tm.getColor("background", tcell.ColorDefault))
	tm.app.preview.SetTextColor(tm.getColor("foreground", tcell.ColorWhite))
}

// applyStatusColors applies theme colors to the status bar
func (tm *ThemeManager) applyStatusColors() {
	tm.app.status.SetBackgroundColor(tm.getColor("background", tcell.ColorDefault))
	tm.app.status.SetTextColor(tm.getColor("status", tcell.ColorGreen))
}

// applySearchColors applies theme colors to the search field
func (tm *ThemeManager) applySearchColors() {
	tm.app.search.SetBackgroundColor(tm.getColor("background", tcell.ColorDefault))
	tm.app.search.SetFieldBackgroundColor(tm.getColor("background", tcell.ColorDefault))
	tm.app.search.SetFieldTextColor(tm.getColor("foreground", tcell.ColorWhite))
	tm.app.search.SetLabelColor(tm.getColor("foreground", tcell.ColorWhite))
}

// getColor safely retrieves a color from the theme map with a fallback
func (tm *ThemeManager) getColor(key string, fallback tcell.Color) tcell.Color {
	if color, ok := tm.colors[key]; ok {
		return color
	}
	return fallback
}

// highlightText applies highlighting to matched text in the preview
func (tm *ThemeManager) highlightText(text string) {
	if text == "" {
		tm.app.preview.SetText(tm.app.preview.GetText(false))
		return
	}

	content := tm.app.preview.GetText(false)
	highlightColor := tm.getColor("search_highlight", tcell.ColorYellow)

	// Convert the color to a tview-compatible format
	// tview uses region tags for coloring: "[<color>]text[-]"
	colorName := highlightColor.String()

	// Create the highlighted content using tview's region tags
	styledContent := replaceWithHighlight(content, text, colorName)
	tm.app.preview.SetText(styledContent)
}

// replaceWithHighlight replaces occurrences of searchText with highlighted version
func replaceWithHighlight(content, searchText, colorName string) string {
	if searchText == "" {
		return content
	}

	if content == "" {
		return content
	}

	// Create the color tags for tview
	openTag := "[" + colorName + "]"
	closeTag := "[-]"

	// Escape any existing tags in the content
	content = tview.Escape(content)

	// Replace the search text with the highlighted version
	return strings.ReplaceAll(content, searchText, openTag+searchText+closeTag)
}
