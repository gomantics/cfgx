// Command cfgx generates type-safe Go code from TOML configuration files.
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	// version is set via ldflags at build time
	version = "dev"
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

func init() {
	// If version info wasn't set via ldflags (e.g., when using go install),
	// try to get it from build info embedded by Go
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Get version from module version
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
		}
	}

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cfgx %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	},
}
