package config

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"

	"github.com/lc/pfzf/pkg/types"
)

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Scanner: ScannerConfig{
			IgnorePatterns: []string{
				".next",
				"webpack",
				".contentlayer",
				".git",
				"node_modules",
				".idea",
				"vendor",
				"*.exe",
				"*.dll",
				"*.so",
				"*.dylib",
				"*.bin",
				"*.dat",
			},
			MaxFileSize: 1 << 20, // 1MB
			MaxFiles:    1000,
		},
		Processor: ProcessorConfig{
			MaxChunkSize:   4096,
			ChunkOverlap:   200,
			MaxTokens:      2000,
			StripComments:  false,
			DetectLanguage: true,
		},
		Writer: WriterConfig{
			OutputPath:  generateRandomFilename(".xml"),
			Format:      types.OutputFormatXML,
			PrettyPrint: true,
		},
		UI: UIConfig{
			PreviewWidth: 50,
			Theme:        "default",
			KeyBindings: map[string]string{
				"quit":           "q",
				"select":         "space",
				"toggle_preview": "p",
				"help":           "?",
				"focus_search":   "/",
				"clear_search":   "esc",
			},
		},
	}
}

// DefaultTheme returns the default UI theme configuration.
func DefaultTheme() map[string]string {
	return map[string]string{
		"background":       "black",
		"foreground":       "white",
		"selection":        "blue",
		"preview":          "default",
		"status":           "green",
		"error":            "red",
		"search_highlight": "yellow",
	}
}

// generateRandomFilename generates a random filename with the given extension
func generateRandomFilename(extension string) string {
	// Generate 8 random bytes (16 hex chars)
	b := make([]byte, 8)
	rand.Read(b)
	return filepath.Join(".", "pfzf_"+hex.EncodeToString(b)+extension)
}
