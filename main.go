package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lc/pfzf/internal/fs"

	"github.com/lc/pfzf/internal/app"
	"github.com/lc/pfzf/internal/config"
	"github.com/lc/pfzf/internal/processor"
	"github.com/lc/pfzf/internal/scanner"
	"github.com/lc/pfzf/internal/writer"
	"github.com/lc/pfzf/pkg/types"
)

var (
	configPath = flag.String("config", "", "path to config file (default: $XDG_CONFIG_HOME/pfzf/config.json)")
	outputPath = flag.String("output", "", "path to output file (default: pfzf_*.xml)")
	format     = flag.String("format", "xml", "output format: xml, json, yaml (default: xml)")
)

func validateFlags() error {
	if *format != "" {
		switch strings.ToLower(*format) {
		case "xml", "json", "yaml":
			// Valid format
		default:
			return fmt.Errorf("invalid format: %s (must be xml, json, or yaml)", *format)
		}
	}
	return nil
}

func main() {
	flag.Parse()

	if err := validateFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags if provided
	if *outputPath != "" {
		cfg.Writer.OutputPath = *outputPath
	}
	if *format != "" {
		cfg.Writer.Format = types.OutputFormat(strings.ToLower(*format))
	}

	// Initialize scanner
	s, err := scanner.New(
		scanner.WithRootDir("."),
		scanner.WithMaxFileSize(cfg.Scanner.MaxFileSize),
		scanner.WithIgnorePattern(cfg.Scanner.IgnorePatterns...),
		scanner.WithMaxFiles(cfg.Scanner.MaxFiles),
	)
	if err != nil {
		log.Fatalf("failed to create scanner: %v", err)
	}

	// Initialize processor with converted options
	procOpts := types.ProcessorOptions{
		MaxChunkSize:  cfg.Processor.MaxChunkSize,
		ChunkOverlap:  cfg.Processor.ChunkOverlap,
		MaxTokens:     cfg.Processor.MaxTokens,
		StripComments: cfg.Processor.StripComments,
	}

	proc, err := processor.New(procOpts)
	if err != nil {
		log.Fatalf("failed to create processor: %v", err)
	}

	// Initialize writer with converted options
	writerOpts := types.WriterOptions{
		OutputPath:  cfg.Writer.OutputPath,
		Format:      cfg.Writer.Format,
		PrettyPrint: cfg.Writer.PrettyPrint,
	}

	w, err := writer.New(writerOpts)
	if err != nil {
		log.Fatalf("failed to create writer: %v", err)
	}
	defer w.Close()

	// Write directory context before starting UI
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get the current directory: %v", err)
	}

	tree, err := fs.GetDirectoryTree(".", fs.TreeOptions{IgnorePatterns: cfg.Scanner.IgnorePatterns})
	if err != nil {
		log.Fatalf("failed to generate directory tree: %v", err)
	}

	if err := w.WriteDirectoryContext(cwd, tree); err != nil {
		log.Fatalf("failed to write directory context: %v\n", err)
	}

	// Create and run application
	app := app.New(cfg, s, proc, w)
	if err := app.Run(); err != nil {
		log.Fatalf("failed to run: %v\n", err)
	}

	fmt.Printf("context written to %s\n", cfg.Writer.OutputPath)
}

// loadConfig loads the configuration from the specified path or uses defaults
func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		// Use default config if no config file exists
		return config.DefaultConfig(), nil
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		// If config file doesn't exist, use defaults
		if os.IsNotExist(err) {
			return config.DefaultConfig(), nil
		}
		return nil, fmt.Errorf("loading config from %s: %w", path, err)
	}

	return cfg, nil
}
