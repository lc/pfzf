// Package fs provides a utility responsible for generating a string representation of the directory tree.
// It ignores common patterns such as .git, .DS_Store, node_modules, and .idea.
package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TreeOptions configures the directory tree generation
type TreeOptions struct {
	IgnorePatterns []string
}

// shouldIgnore checks if a path should be ignored based on patterns
func shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		// Handle glob patterns
		if strings.Contains(pattern, "*") {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				return true
			}
			continue
		}

		// Handle direct matches and path components
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// GetDirectoryTree returns a string representation of the directory tree
func GetDirectoryTree(root string, opts TreeOptions) (string, error) {
	var tree strings.Builder
	tree.WriteString(".\n")

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		// Use configured ignore patterns
		if shouldIgnore(path, opts.IgnorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("  ", depth)
		tree.WriteString(fmt.Sprintf("%s├── %s\n", indent, filepath.Base(path)))
		return nil
	})

	return tree.String(), err
}
