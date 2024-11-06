package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) setupUI() {
	// Configure search field
	a.search.SetLabel("Search: ").
		SetChangedFunc(a.handleSearch)

	// Configure file list
	a.fileList.ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("Files (↑/↓ to move, Space to select, q to quit)")

		// Configure preview pane
	a.preview.SetBorder(true)
	a.preview.SetTitle("Preview")
	a.preview.SetDynamicColors(true) // This method exists on TextView directly
	a.preview.SetWrap(true)

	// Configure status bar
	a.status.SetBorder(true).
		SetTitle("Status")

	// Create layout
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.search, 1, 0, true).
		AddItem(tview.NewFlex().
			AddItem(a.fileList, 0, 2, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(a.preview, 0, 3, false).
				AddItem(a.status, 3, 1, false), 0, 3, false),
			0, 1, false)

	// Set up key handlers
	a.fileList.SetInputCapture(a.handleInput)
	a.search.SetInputCapture(a.handleSearchInput)

	// Set up selection handler
	a.fileList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.handleSelection(index)
	})

	a.SetRoot(mainFlex, true)
}

func (a *App) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q':
			a.Stop()
			return nil
		case ' ':
			if idx := a.fileList.GetCurrentItem(); idx >= 0 && idx < len(a.filteredIdx) {
				a.toggleSelection(a.filteredIdx[idx])
			}
			return nil
		}
	case tcell.KeyEscape:
		a.SetFocus(a.search)
		return nil
	}
	return event
}

func (a *App) handleSearchInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyDown:
		a.SetFocus(a.fileList)
		return nil
	case tcell.KeyEnter:
		if len(a.filteredIdx) > 0 {
			a.SetFocus(a.fileList)
			return nil
		}
	}
	return event
}
