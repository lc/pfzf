// Package config provides configuration management for pfzf.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lc/pfzf/pkg/types"
)

// Config represents the complete configuration for pfzf.
type Config struct {
	// Scanner configuration
	Scanner ScannerConfig `json:"scanner"`

	// Processor configuration
	Processor ProcessorConfig `json:"processor"`

	// Writer configuration
	Writer WriterConfig `json:"writer"`

	// UI configuration
	UI UIConfig `json:"ui"`
}

// ScannerConfig configures the file scanner behavior.
type ScannerConfig struct {
	IgnorePatterns []string `json:"ignorePatterns"`
	MaxFileSize    int64    `json:"maxFileSize"`
	MaxFiles       int      `json:"maxFiles"`
}

// ProcessorConfig configures content processing behavior.
type ProcessorConfig struct {
	MaxChunkSize   int64 `json:"maxChunkSize"`
	ChunkOverlap   int   `json:"chunkOverlap"`
	MaxTokens      int   `json:"maxTokens"`
	StripComments  bool  `json:"stripComments"`
	DetectLanguage bool  `json:"detectLanguage"`
}

// WriterConfig configures output writing behavior.
type WriterConfig struct {
	OutputPath  string             `json:"outputPath"`
	Format      types.OutputFormat `json:"format"`
	PrettyPrint bool               `json:"prettyPrint"`
}

// UIConfig configures the user interface behavior.
type UIConfig struct {
	PreviewWidth int               `json:"previewWidth"`
	Theme        string            `json:"theme"`
	KeyBindings  map[string]string `json:"keyBindings"`
	CustomTheme  map[string]string `json:"customTheme,omitempty"`
}

// LoadConfig loads configuration from the specified path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	var extension string
	switch config.Writer.Format {
	case types.OutputFormatJSON:
		extension = ".json"
	case types.OutputFormatYAML:
		extension = ".yaml"
	default:
		extension = ".xml"
	}
	config.Writer.OutputPath = generateRandomFilename(extension)
	return &config, nil
}

// SaveConfig saves the configuration to the specified path.
func SaveConfig(config *Config, path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default configuration file path.
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pfzf/config.json"
	}
	return filepath.Join(home, ".pfzf", "config.json")
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Scanner.MaxFileSize < 0 {
		return fmt.Errorf("maxFileSize must be non-negative")
	}
	if c.Scanner.MaxFiles < 0 {
		return fmt.Errorf("maxFiles must be non-negative")
	}
	if c.Processor.MaxChunkSize < 0 {
		return fmt.Errorf("maxChunkSize must be non-negative")
	}
	if c.Processor.ChunkOverlap < 0 {
		return fmt.Errorf("chunkOverlap must be non-negative")
	}
	if c.Processor.MaxTokens < 0 {
		return fmt.Errorf("maxTokens must be non-negative")
	}
	return nil
}

// ValidateTheme checks if the theme configuration is valid.
func (c *UIConfig) ValidateTheme() error {
	if c.Theme == "" {
		c.Theme = "default"
	}

	// If using a custom theme, validate the required colors are present
	if c.Theme != "default" && len(c.CustomTheme) > 0 {
		required := []string{"background", "foreground", "selection", "status"}
		for _, color := range required {
			if _, ok := c.CustomTheme[color]; !ok {
				return fmt.Errorf("custom theme missing required color: %s", color)
			}
		}
	}
	return nil
}
