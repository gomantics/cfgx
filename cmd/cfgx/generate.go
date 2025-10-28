package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gomantics/cfgx"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate type-safe Go code from TOML config",
	Long:  `Generate type-safe Go code from TOML configuration files.`,
	Example: `  # Generate config code
  cfgx generate --in config.toml --out config/config.go

  # Custom package
  cfgx generate --in app.toml --out pkg/appcfg/config.go --pkg appcfg

  # Disable environment variable overrides
  cfgx generate --in config.toml --out config.go --no-env`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Require -out flag
		if outputFile == "" {
			return fmt.Errorf("--out flag is required")
		}

		// Validate mode
		if mode != "static" && mode != "getter" {
			return fmt.Errorf("invalid --mode value %q: must be 'static' or 'getter'", mode)
		}

		// Parse max file size
		maxFileSizeBytes, err := parseFileSize(maxFileSize)
		if err != nil {
			return fmt.Errorf("invalid --max-file-size: %w", err)
		}

		// Use the public API
		opts := &cfgx.GenerateOptions{
			InputFile:   inputFile,
			OutputFile:  outputFile,
			PackageName: packageName,
			EnableEnv:   !noEnv,
			MaxFileSize: maxFileSizeBytes,
			Mode:        mode,
		}

		if err := cfgx.GenerateFromFile(opts); err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", outputFile)
		return nil
	},
	SilenceUsage: true,
}

func init() {
	// Generate command flags
	generateCmd.Flags().StringVarP(&inputFile, "in", "i", "config.toml", "input TOML file")
	generateCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output Go file (required)")
	generateCmd.Flags().StringVarP(&packageName, "pkg", "p", "", "package name (default: inferred from output path or 'config')")
	generateCmd.Flags().BoolVar(&noEnv, "no-env", false, "disable environment variable overrides")
	generateCmd.Flags().StringVar(&maxFileSize, "max-file-size", "1MB", "maximum file size for file: references (e.g., 10MB, 1GB, 512KB)")
	generateCmd.Flags().StringVar(&mode, "mode", "static", "generation mode: 'static' (values baked at build time) or 'getter' (runtime env var overrides)")

	generateCmd.MarkFlagRequired("out")
}
