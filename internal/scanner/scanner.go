// Package scanner provides efficient file scanning functionality for pfzf.
package scanner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/lc/pfzf/pkg/types"
)

const (
	binaryCheckSize = 512
	binaryThreshold = 0.3
	workerCount     = 4
)

type Scanner struct {
	opts    types.ScanOptions
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	results chan types.FileEntry
	errors  chan error
}

func New(opts ...Option) (*Scanner, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scanner{
		ctx:     ctx,
		cancel:  cancel,
		results: make(chan types.FileEntry),
		errors:  make(chan error),
		opts: types.ScanOptions{
			RootDir:     ".",
			MaxFileSize: 1 << 20, // 1MB default
		},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Scanner) Scan(opts types.ScanOptions) (<-chan types.FileEntry, <-chan error) {
	if opts.RootDir != "" {
		s.opts.RootDir = opts.RootDir
	}
	if opts.MaxFileSize > 0 {
		s.opts.MaxFileSize = opts.MaxFileSize
	}
	if len(opts.IgnorePattern) > 0 {
		s.opts.IgnorePattern = opts.IgnorePattern
	}
	if opts.MaxFiles > 0 {
		s.opts.MaxFiles = opts.MaxFiles
	}

	go s.startScan()
	return s.results, s.errors
}

func (s *Scanner) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *Scanner) startScan() {
	defer close(s.results)
	defer close(s.errors)

	paths := make(chan string)

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go s.worker(paths)
	}

	// Walk directory tree
	go func() {
		defer close(paths)
		err := filepath.Walk(s.opts.RootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				select {
				case s.errors <- fmt.Errorf("walk error at %s: %w", path, err):
				case <-s.ctx.Done():
				}
				return nil
			}

			skip, skipDir := s.shouldSkip(path, info)
			if skip {
				if info.IsDir() && skipDir {
					return filepath.SkipDir
				}
				return nil
			}

			if !info.IsDir() {
				select {
				case paths <- path:
				case <-s.ctx.Done():
					return filepath.SkipDir
				}
			}

			return nil
		})
		if err != nil {
			select {
			case s.errors <- fmt.Errorf("walk error: %w", err):
			case <-s.ctx.Done():
			}
		}
	}()

	s.wg.Wait()
}

func (s *Scanner) worker(paths <-chan string) {
	defer s.wg.Done()

	for {
		select {
		case path, ok := <-paths:
			if !ok {
				return
			}
			if entry, err := s.processFile(path); err != nil {
				select {
				case s.errors <- fmt.Errorf("processing file %s: %w", path, err):
				case <-s.ctx.Done():
					return
				}
			} else {
				select {
				case s.results <- entry:
				case <-s.ctx.Done():
					return
				}
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scanner) shouldSkip(path string, info os.FileInfo) (bool, bool) {
	// Skip files larger than MaxFileSize
	if !info.IsDir() && info.Size() > s.opts.MaxFileSize {
		return true, false
	}

	// Get the relative path for pattern matching
	relPath, err := filepath.Rel(s.opts.RootDir, path)
	if err != nil {
		// If we can't get relative path, use full path
		relPath = path
	}

	// Check patterns against the relative path
	for _, pattern := range s.opts.IgnorePattern {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true, info.IsDir()
		}

		// Handle directory wildcard patterns (e.g., "ignored/*")
		if strings.HasSuffix(pattern, "/*") {
			dirPattern := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(relPath, dirPattern+string(filepath.Separator)) {
				return true, info.IsDir()
			}
		}
	}

	return false, false
}

func (s *Scanner) processFile(path string) (types.FileEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return types.FileEntry{}, fmt.Errorf("stat error: %w", err)
	}

	isBinary, err := s.isBinaryFile(path)
	if err != nil {
		return types.FileEntry{}, fmt.Errorf("binary check error: %w", err)
	}

	// Get relative path
	relPath, err := filepath.Rel(s.opts.RootDir, path)
	if err != nil {
		return types.FileEntry{}, fmt.Errorf("relative path error: %w", err)
	}

	return types.FileEntry{
		Path:     relPath,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		IsBinary: isBinary,
	}, nil
}

func (s *Scanner) isBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, binaryCheckSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	buf = buf[:n]

	if len(buf) == 0 {
		return false, nil
	}

	nonPrintable := 0
	for _, b := range buf {
		if b == 0 || (!unicode.IsGraphic(rune(b)) && !unicode.IsSpace(rune(b))) {
			nonPrintable++
		}
	}

	ratio := float64(nonPrintable) / float64(len(buf))
	return ratio > binaryThreshold, nil
}
