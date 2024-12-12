package scanner

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Option represents a scanner configuration option.
type Option func(*Scanner) error

// WithRootDir sets the root directory for scanning.
func WithRootDir(dir string) Option {
	return func(s *Scanner) error {
		if dir == "" {
			dir = "."
		}
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("invalid root directory: %w", err)
		}
		s.opts.RootDir = absPath
		return nil
	}
}

// WithIgnorePattern adds ignore patterns for file scanning.
func WithIgnorePattern(patterns ...string) Option {
	return func(s *Scanner) error {
		for _, pattern := range patterns {
			if strings.TrimSpace(pattern) != "" {
				s.opts.IgnorePattern = append(s.opts.IgnorePattern, pattern)
			}
		}
		return nil
	}
}

// WithMaxFileSize sets the maximum file size for scanning.
func WithMaxFileSize(size int64) Option {
	return func(s *Scanner) error {
		if size < 0 {
			return fmt.Errorf("max file size must be non-negative")
		}
		s.opts.MaxFileSize = size
		return nil
	}
}

// WithMaxFiles sets the maximum number of files to scan.
func WithMaxFiles(count int) Option {
	return func(s *Scanner) error {
		if count < 0 {
			return fmt.Errorf("max files must be non-negative")
		}
		s.opts.MaxFiles = count
		return nil
	}
}

// Configure applies the given options to the scanner.
func (s *Scanner) Configure(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return fmt.Errorf("configuring scanner: %w", err)
		}
	}
	return nil
}

// DefaultOptions returns the default scanner options.
func DefaultOptions() []Option {
	return []Option{
		WithRootDir("."),
		WithMaxFileSize(5 << 20), // 5MB
		WithMaxFiles(1000),
		WithIgnorePattern(
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
		),
	}
}
