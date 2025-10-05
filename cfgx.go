// Package cfgx generates type-safe Go code from TOML configuration files.
//
// This package provides a clean API for generating strongly-typed configuration code
// from TOML files, with optional environment variable override support.
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

	// Generate code
	generated, err := Generate(data, packageName, opts.EnableEnv)
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
func Generate(tomlData []byte, packageName string, enableEnv bool) ([]byte, error) {
	if packageName == "" {
		packageName = "config"
	}

	gen := generator.New(
		generator.WithPackageName(packageName),
		generator.WithEnvOverride(enableEnv),
	)

	return gen.Generate(tomlData)
}
