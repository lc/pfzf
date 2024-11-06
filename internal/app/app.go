package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/lc/pfzf/internal/config"
	"github.com/lc/pfzf/pkg/types"
	"github.com/rivo/tview"
)

// App represents the main application.
type App struct {
	*tview.Application
	config       *config.Config
	scanner      types.Scanner
	processor    types.Processor
	writer       types.Writer
	themeManager *ThemeManager

	// UI components
	fileList *tview.List
	preview  *tview.TextView
	status   *tview.TextView
	search   *tview.InputField

	// State
	entries      []types.FileEntry
	filteredIdx  []int
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	searchString string
}

// New creates a new App instance.
func New(cfg *config.Config, scanner types.Scanner, processor types.Processor, writer types.Writer) *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Application: tview.NewApplication(),
		config:      cfg,
		scanner:     scanner,
		processor:   processor,
		writer:      writer,
		fileList:    tview.NewList(),
		preview:     tview.NewTextView(),
		status:      tview.NewTextView(),
		search:      tview.NewInputField(),
		ctx:         ctx,
		cancel:      cancel,
		filteredIdx: make([]int, 0),
	}

	// initialize theme manager
	app.themeManager = newThemeManager(app)
	app.themeManager.applyTheme(config.DefaultTheme())

	// if theme is not default, apply it
	if cfg.UI.Theme != "default" {
		app.themeManager.applyTheme(config.DefaultTheme())
	}

	app.setupUI()
	return app
}

// Run starts the application.
func (a *App) Run() error {
	// Start file scanning
	if err := a.startScanning(); err != nil {
		return fmt.Errorf("scanning files: %w", err)
	}

	// Run the application
	if err := a.Application.Run(); err != nil {
		return fmt.Errorf("running application: %w", err)
	}

	// Cleanup
	a.cancel()

	if err := a.writer.Flush(); err != nil {
		return fmt.Errorf("flushing writer: %w", err)
	}

	return a.writer.Close()
}

// Stop stops the application.
func (a *App) Stop() {
	a.cancel()
	a.Application.Stop()
}
