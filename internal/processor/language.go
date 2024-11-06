package processor

import (
	"bufio"
	"bytes"
	"io"
	"path/filepath"
	"strings"
)

// LanguageDetector handles programming language detection and processing.
type LanguageDetector struct {
	// extensionMap maps file extensions to language names
	extensionMap map[string]string
	// shebangMap maps shebang patterns to language names
	shebangMap map[string]string
	// commentMap maps languages to their comment strippers
	commentMap map[string]CommentStripper
}

// CommentStripper defines the interface for language-specific comment stripping.
type CommentStripper interface {
	StripComments(content []byte) ([]byte, error)
}

// NewLanguageDetector creates a new language detector with predefined mappings.
func NewLanguageDetector() (*LanguageDetector, error) {
	ld := &LanguageDetector{
		extensionMap: make(map[string]string),
		shebangMap:   make(map[string]string),
		commentMap:   make(map[string]CommentStripper),
	}

	// Initialize extension mappings
	ld.initExtensionMap()
	// Initialize shebang mappings
	ld.initShebangMap()
	// Initialize comment strippers
	ld.initCommentStrippers()

	return ld, nil
}

// DetectLanguage attempts to identify the programming language of a file.
func (ld *LanguageDetector) DetectLanguage(filename string, reader io.Reader) (string, error) {
	// Try extension-based detection first
	if lang := ld.detectByExtension(filename); lang != "" {
		return lang, nil
	}

	// Try shebang-based detection for scripts
	if lang := ld.detectByShebang(reader); lang != "" {
		return lang, nil
	}

	return "unknown", nil
}

// GetCommentStripper returns a comment stripper for the given language.
func (ld *LanguageDetector) GetCommentStripper(language string) (CommentStripper, error) {
	stripper, ok := ld.commentMap[language]
	if !ok {
		return &GenericCommentStripper{}, nil
	}
	return stripper, nil
}

func (ld *LanguageDetector) detectByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	return ld.extensionMap[ext]
}

func (ld *LanguageDetector) detectByShebang(reader io.Reader) string {
	// Reset reader if it's a seeker
	if seeker, ok := reader.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return ""
	}

	firstLine := scanner.Text()
	if !strings.HasPrefix(firstLine, "#!") {
		return ""
	}

	for pattern, lang := range ld.shebangMap {
		if strings.Contains(firstLine, pattern) {
			return lang
		}
	}

	return ""
}

func (ld *LanguageDetector) initExtensionMap() {
	extensions := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".rb":    "ruby",
		".php":   "php",
		".java":  "java",
		".cpp":   "cpp",
		".cc":    "cpp",
		".c":     "c",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rs":    "rust",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".r":     "r",
		".sh":    "shell",
		".bash":  "shell",
		".zsh":   "shell",
		".fish":  "shell",
		".pl":    "perl",
		".pm":    "perl",
		".t":     "perl",
		".html":  "html",
		".htm":   "html",
		".css":   "css",
		".scss":  "scss",
		".sass":  "scss",
		".less":  "less",
		".xml":   "xml",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".md":    "markdown",
		".sql":   "sql",
		".lua":   "lua",
		".vim":   "vim",
		".el":    "elisp",
		".clj":   "clojure",
		".ex":    "elixir",
		".exs":   "elixir",
		".erl":   "erlang",
		".hs":    "haskell",
		".ml":    "ocaml",
		".mli":   "ocaml",
	}

	for ext, lang := range extensions {
		ld.extensionMap[ext] = lang
	}
}

func (ld *LanguageDetector) initShebangMap() {
	shebangs := map[string]string{
		"python": "python",
		"ruby":   "ruby",
		"node":   "javascript",
		"php":    "php",
		"perl":   "perl",
		"bash":   "shell",
		"sh":     "shell",
		"zsh":    "shell",
		"fish":   "shell",
		"lua":    "lua",
		"R":      "r",
	}

	for pattern, lang := range shebangs {
		ld.shebangMap[pattern] = lang
	}
}

func (ld *LanguageDetector) initCommentStrippers() {
	ld.commentMap = map[string]CommentStripper{
		"go":         &GoCommentStripper{},
		"python":     &PythonCommentStripper{},
		"javascript": &JavaScriptCommentStripper{},
		"typescript": &JavaScriptCommentStripper{},
		"java":       &JavaCommentStripper{},
		"cpp":        &CppCommentStripper{},
		"c":          &CCommentStripper{},
		"rust":       &RustCommentStripper{},
		"shell":      &ShellCommentStripper{},
	}
}

// Generic comment stripper that handles common comment styles
type GenericCommentStripper struct{}

func (s *GenericCommentStripper) StripComments(content []byte) ([]byte, error) {
	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var (
		inMultiLineComment bool
		lastLineWasEmpty   bool
	)

	for scanner.Scan() {
		line := scanner.Text()
		originalIndent := getIndentation(line)
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines if the last line was empty
		if trimmedLine == "" {
			if !lastLineWasEmpty {
				result.WriteString("\n")
				lastLineWasEmpty = true
			}
			continue
		}

		// Handle multi-line comments
		if inMultiLineComment {
			if idx := strings.Index(line, "*/"); idx >= 0 {
				inMultiLineComment = false
				line = originalIndent + strings.TrimSpace(line[idx+2:])
				if strings.TrimSpace(line) == "" {
					continue
				}
			} else {
				continue
			}
		}

		// Check for start of multi-line comment
		if idx := strings.Index(line, "/*"); idx >= 0 {
			inMultiLineComment = true
			beforeComment := strings.TrimSpace(line[:idx])
			if beforeComment == "" {
				continue
			}
			line = originalIndent + beforeComment
		}

		// Handle single-line comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			beforeComment := strings.TrimSpace(line[:idx])
			if beforeComment == "" {
				continue
			}
			line = originalIndent + beforeComment
		}

		// Preserve structural empty lines between major blocks
		if strings.TrimSpace(line) == "" {
			if !lastLineWasEmpty {
				result.WriteString("\n")
				lastLineWasEmpty = true
			}
			continue
		}

		// Write the processed line
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(line)
		lastLineWasEmpty = false
	}

	// Ensure content ends with a single newline
	if result.Len() > 0 {
		return bytes.TrimRight(result.Bytes(), "\n"), scanner.Err()
	}
	return result.Bytes(), scanner.Err()
}

// getIndentation returns the leading whitespace of a line
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// Language-specific comment strippers
type (
	GoCommentStripper         struct{ GenericCommentStripper }
	PythonCommentStripper     struct{ GenericCommentStripper }
	JavaScriptCommentStripper struct{ GenericCommentStripper }
	JavaCommentStripper       struct{ GenericCommentStripper }
	CppCommentStripper        struct{ GenericCommentStripper }
	CCommentStripper          struct{ GenericCommentStripper }
	RustCommentStripper       struct{ GenericCommentStripper }
	ShellCommentStripper      struct{ GenericCommentStripper }
)
