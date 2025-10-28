// Package cfgx generates type-safe Go code from TOML configuration files.
//
// This package provides a clean API for generating strongly-typed configuration code
// from TOML files, with optional environment variable override support and file embedding.
//
// Example usage:
//
//	// Basic generation
//	opts := &cfgx.GenerateOptions{
//		InputFile:   "config.toml",
//		OutputFile:  "config/config.go",
//		PackageName: "config",
//		EnableEnv:   true,
//	}
//	if err := cfgx.GenerateFromFile(opts); err != nil {
//		log.Fatal(err)
//	}
//
//	// With file embedding
//	opts := &cfgx.GenerateOptions{
//		InputFile:   "config.toml",
//		OutputFile:  "config/config.go",
//		PackageName: "config",
//		MaxFileSize: 5 * cfgx.DefaultMaxFileSize, // 5MB limit
//	}
//	if err := cfgx.GenerateFromFile(opts); err != nil {
//		log.Fatal(err)
//	}
//
//	// TOML with file references:
//	// [server]
//	// tls_cert = "file:certs/server.crt"
//	// This generates a []byte field with embedded file contents
//
//	// Programmatic usage
//	tomlData := []byte(`[server]
//	addr = ":8080"`)
//	code, err := cfgx.Generate(tomlData, "config", true)
//	if err != nil {
//		log.Fatal(err)
//	}
package cfgx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/gomantics/cfgx/internal/envoverride"
	"github.com/gomantics/cfgx/internal/generator"
	"github.com/gomantics/cfgx/internal/pkgutil"
)

// DefaultMaxFileSize is the default maximum file size (1 MB) for files referenced with "file:" prefix.
const DefaultMaxFileSize = 1024 * 1024 // 1 MB

// GenerateOptions contains all options for generating configuration code.
type GenerateOptions struct {
	// InputFile is the path to the input TOML file
	InputFile string

	// OutputFile is the path where the generated Go code will be written
	OutputFile string

	// PackageName is the Go package name for the generated code.
	// If empty, it will be inferred from the output file path.
	PackageName string

	// EnableEnv enables environment variable override support
	EnableEnv bool

	// MaxFileSize is the maximum size in bytes for files referenced with "file:" prefix.
	// If zero, defaults to DefaultMaxFileSize (1 MB).
	MaxFileSize int64

	// Mode specifies the generation mode:
	//   "static" - values baked at build time (default)
	//   "getter" - generate getter methods with runtime env var overrides
	// If empty, defaults to "static".
	Mode string
}

// GenerateFromFile generates Go code from a TOML file and writes it to the output file.
// This is the main entry point for file-based generation.
func GenerateFromFile(opts *GenerateOptions) error {
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	if opts.OutputFile == "" {
		return fmt.Errorf("output file is required")
	}

	// Read input file
	data, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file %s: %w", opts.InputFile, err)
	}

	// Parse TOML to apply environment variable overrides if enabled
	var configData map[string]any
	if err := toml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Apply environment variable overrides
	if opts.EnableEnv {
		if err := envoverride.Apply(configData); err != nil {
			return fmt.Errorf("failed to apply environment overrides: %w", err)
		}

		// Re-marshal to TOML for generation
		// This ensures the overridden values are used
		var buf bytes.Buffer
		enc := toml.NewEncoder(&buf)
		if err := enc.Encode(configData); err != nil {
			return fmt.Errorf("failed to re-encode TOML: %w", err)
		}
		data = buf.Bytes()
	}

	// Infer package name if not provided
	packageName := opts.PackageName
	if packageName == "" {
		packageName = pkgutil.InferName(opts.OutputFile)
	}

	// Extract input directory for resolving file: references
	inputDir := filepath.Dir(opts.InputFile)

	// Set default max file size if not specified
	maxFileSize := opts.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = DefaultMaxFileSize
	}

	// Set default mode if not specified
	mode := opts.Mode
	if mode == "" {
		mode = "static"
	}

	// Generate code
	generated, err := GenerateWithOptions(data, packageName, opts.EnableEnv, inputDir, maxFileSize, mode)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(opts.OutputFile)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write output file
	if err := os.WriteFile(opts.OutputFile, generated, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// Generate generates Go code from TOML data with the specified package name.
// This is useful for programmatic usage where you have the TOML data in memory.
//
// Parameters:
//   - tomlData: The TOML configuration data as bytes
//   - packageName: The Go package name for the generated code
//   - enableEnv: Whether to enable environment variable override markers in generated code
//
// Returns the generated Go code as bytes, or an error if generation fails.
//
// Note: This function does not support file: references since no input directory is provided.
// Use GenerateWithOptions for full file embedding support.
func Generate(tomlData []byte, packageName string, enableEnv bool) ([]byte, error) {
	return GenerateWithOptions(tomlData, packageName, enableEnv, "", DefaultMaxFileSize, "static")
}

// GenerateWithOptions generates Go code from TOML data with full options support.
// This is useful for programmatic usage where you have the TOML data in memory
// and need to control file embedding behavior.
//
// Parameters:
//   - tomlData: The TOML configuration data as bytes
//   - packageName: The Go package name for the generated code
//   - enableEnv: Whether to enable environment variable override markers in generated code
//   - inputDir: Directory to resolve file: references from (empty string to disable)
//   - maxFileSize: Maximum file size in bytes for file: references (0 for default 1MB)
//   - mode: Generation mode ("static" or "getter")
//
// Returns the generated Go code as bytes, or an error if generation fails.
func GenerateWithOptions(tomlData []byte, packageName string, enableEnv bool, inputDir string, maxFileSize int64, mode string) ([]byte, error) {
	if packageName == "" {
		packageName = "config"
	}

	if maxFileSize == 0 {
		maxFileSize = DefaultMaxFileSize
	}

	if mode == "" {
		mode = "static"
	}

	gen := generator.New(
		generator.WithPackageName(packageName),
		generator.WithEnvOverride(enableEnv),
		generator.WithInputDir(inputDir),
		generator.WithMaxFileSize(maxFileSize),
		generator.WithMode(mode),
	)

	return gen.Generate(tomlData)
}
