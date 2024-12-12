# pfzf

Interactively generate structured context files from your codebase, designed for LLM pair programming workflows. 

Kind of inspired by `fzf`. I put 0 effort in, not responsible for anything.

> ⚠️ **Alpha Status**: This project is in early development, written mostly with Claude because I was lazy and needed this quickly. APIs and features may change significantly.
> 


## Features (if you want to even call them that)

- Interactive file preview and selection with fuzzy search
- Fast and memory-efficient processing
- Multiple output formats (XML, JSON, YAML)
- Terminal UI with customizable themes (sort of works lol)

## Installation

```bash
go install github.com/lc/pfzf@latest
```

## Quick Start

```bash
# Basic usage (outputs to pfzf_*.xml by default)
pfzf

# Specify output format
pfzf -format json

# Use custom config file
pfzf -config ~/.config/pfzf/config.json
```

## Configuration
pfzf can be configured via a JSON configuration file located at `$HOME/.pfzf/config.json`

You can specify a custom config location using the `-config` flag:


**Note**: a lot of the keys are not implemented yet, but they are there for future use.

```bash
pfzf -config /path/to/config.json
```

```json
{
  "scanner": {
    "ignorePatterns": [".git", "node_modules"],
    "maxFileSize": 1048576,
    "maxFiles": 1000
  },
  "processor": {
    "maxChunkSize": 4096,
    "chunkOverlap": 200,
    "maxTokens": 2000,
    "stripComments": false,
    "detectLanguage": true
  },
  "writer": {
    "outputPath": "",
    "format": "xml",
    "prettyPrint": true
  },
  "ui": {
    "previewWidth": 50,
    "theme": "default",
    "keyBindings": {
      "quit": "q",
      "select": "space",
      "toggle_preview": "p",
      "help": "?",
      "focus_search": "/",
      "clear_search": "esc"
    }
  }
}
```

## Key Bindings

- `Space`: Select/deselect file
- `↑/↓`: Navigate files
- `/`: Focus search
- `ESC`: Clear search
- `p`: Toggle preview
- `q`: Quit
- `?`: Show help

## Output Formats

pfzf supports three output formats:

- XML (default)
- JSON
- YAML

Each format includes:
- Directory context (current working directory and tree structure)
- Selected file contents with metadata
- Language-specific processing results (when enabled)

## Development Status

This is an alpha release. While the core functionality is working, you may encounter:
- API changes
- Feature additions/removals
- Performance optimizations
- Bug fixes

Please report issues and feature requests via GitHub issues.

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting pull requests.

## License

[MIT License](LICENSE)
