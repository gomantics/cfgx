// Command cfgx generates type-safe Go code from TOML configuration files.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/gomantics/cfgx"
)

var (
	// version is set via ldflags at build time
	version = "dev"
	// commit is set via ldflags at build time
	commit = "none"
	// date is set via ldflags at build time
	date = "unknown"
)

var (
	inputFile   string
	outputFile  string
	packageName string
	noEnv       bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "cfgx",
	Short: "Type-safe config generation for Go",
	Long: `cfgx generates type-safe Go code from TOML configuration files.

It creates strongly-typed structs with values from the TOML file, with optional environment variable overrides.`,
}

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

		// Use the public API
		opts := &cfgx.GenerateOptions{
			InputFile:   inputFile,
			OutputFile:  outputFile,
			PackageName: packageName,
			EnableEnv:   !noEnv,
		}

		if err := cfgx.GenerateFromFile(opts); err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", outputFile)
		return nil
	},
	SilenceUsage: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cfgx version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built at: %s\n", date)
		fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	// Generate command flags
	generateCmd.Flags().StringVarP(&inputFile, "in", "i", "config.toml", "input TOML file")
	generateCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output Go file (required)")
	generateCmd.Flags().StringVarP(&packageName, "pkg", "p", "", "package name (default: inferred from output path or 'config')")
	generateCmd.Flags().BoolVar(&noEnv, "no-env", false, "disable environment variable overrides")

	generateCmd.MarkFlagRequired("out")

	// Add subcommands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(versionCmd)
}
