// Command cfgx generates type-safe Go code from TOML configuration files.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/gomantics/cfgx"
)

var (
	// version is set via ldflags at build time
	version = "dev"
)

var (
	inputFile   string
	outputFile  string
	packageName string
	noEnv       bool
	maxFileSize string
	mode        string
	debounce    int
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// parseFileSize parses a human-readable file size string like "10MB", "1GB", "512KB"
// into bytes. Returns 0 and error if parsing fails.
func parseFileSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	// Define multipliers in order from longest to shortest to avoid prefix issues
	multipliers := []struct {
		suffix     string
		multiplier int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	// Try to parse with suffix (check longest first)
	for _, m := range multipliers {
		if strings.HasSuffix(sizeStr, m.suffix) {
			numStr := strings.TrimSuffix(sizeStr, m.suffix)
			numStr = strings.TrimSpace(numStr)

			num, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size format: %s", sizeStr)
			}

			return num * m.multiplier, nil
		}
	}

	// Try to parse as plain number (bytes)
	num, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return num, nil
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

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch TOML file and auto-regenerate on changes",
	Long:  `Watch a TOML configuration file and automatically regenerate Go code when it changes.`,
	Example: `  # Watch and auto-regenerate
  cfgx watch --in config.toml --out config/config.go

  # With custom debounce (default 100ms)
  cfgx watch --in config.toml --out config.go --debounce 200

  # Watch with custom mode
  cfgx watch --in config.toml --out config.go --mode getter`,
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

		// Get absolute path for watching (fsnotify works better with absolute paths)
		absInputFile, err := filepath.Abs(inputFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		// Create generate options
		opts := &cfgx.GenerateOptions{
			InputFile:   inputFile,
			OutputFile:  outputFile,
			PackageName: packageName,
			EnableEnv:   !noEnv,
			MaxFileSize: maxFileSizeBytes,
			Mode:        mode,
		}

		// Perform initial generation
		fmt.Printf("Generating %s...\n", outputFile)
		if err := cfgx.GenerateFromFile(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Println("Continuing to watch for changes...")
		} else {
			fmt.Printf("✓ Generated %s\n", outputFile)
		}

		// Create file watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		defer watcher.Close()

		// Add file to watcher
		if err := watcher.Add(absInputFile); err != nil {
			return fmt.Errorf("failed to watch %s: %w", absInputFile, err)
		}

		fmt.Printf("\nWatching %s for changes (Ctrl+C to stop)...\n", inputFile)

		// Setup signal handler for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Debounce timer
		var debounceTimer *time.Timer
		debounceDuration := time.Duration(debounce) * time.Millisecond

		// Watch loop
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}

				// Handle file events (Write, Create, Remove)
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					// Debounce: reset timer on each event
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceDuration, func() {
						fmt.Printf("\n[%s] Change detected, regenerating...\n", time.Now().Format("15:04:05"))
						if err := cfgx.GenerateFromFile(opts); err != nil {
							fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
						} else {
							fmt.Printf("✓ Generated %s\n", outputFile)
						}
					})
				} else if event.Has(fsnotify.Remove) {
					// File was removed - common with some editors (vim, etc.)
					// Try to re-add the watcher when file is recreated
					fmt.Println("File removed, waiting for recreation...")
					// Remove from watcher (it's already gone)
					watcher.Remove(absInputFile)

					// Try to re-add (with retries for editor save patterns)
					go func() {
						for i := 0; i < 10; i++ {
							time.Sleep(100 * time.Millisecond)
							if err := watcher.Add(absInputFile); err == nil {
								fmt.Println("File recreated, watching again...")
								return
							}
						}
						fmt.Fprintf(os.Stderr, "Warning: Could not re-watch file after removal\n")
					}()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)

			case <-sigChan:
				fmt.Println("\nStopping watch...")
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				return nil
			}
		}
	},
	SilenceUsage: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cfgx %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	},
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

	// Generate command flags
	generateCmd.Flags().StringVarP(&inputFile, "in", "i", "config.toml", "input TOML file")
	generateCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output Go file (required)")
	generateCmd.Flags().StringVarP(&packageName, "pkg", "p", "", "package name (default: inferred from output path or 'config')")
	generateCmd.Flags().BoolVar(&noEnv, "no-env", false, "disable environment variable overrides")
	generateCmd.Flags().StringVar(&maxFileSize, "max-file-size", "1MB", "maximum file size for file: references (e.g., 10MB, 1GB, 512KB)")
	generateCmd.Flags().StringVar(&mode, "mode", "static", "generation mode: 'static' (values baked at build time) or 'getter' (runtime env var overrides)")

	generateCmd.MarkFlagRequired("out")

	// Watch command flags (reuse generate flags)
	watchCmd.Flags().StringVarP(&inputFile, "in", "i", "config.toml", "input TOML file")
	watchCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output Go file (required)")
	watchCmd.Flags().StringVarP(&packageName, "pkg", "p", "", "package name (default: inferred from output path or 'config')")
	watchCmd.Flags().BoolVar(&noEnv, "no-env", false, "disable environment variable overrides")
	watchCmd.Flags().StringVar(&maxFileSize, "max-file-size", "1MB", "maximum file size for file: references (e.g., 10MB, 1GB, 512KB)")
	watchCmd.Flags().StringVar(&mode, "mode", "static", "generation mode: 'static' (values baked at build time) or 'getter' (runtime env var overrides)")
	watchCmd.Flags().IntVar(&debounce, "debounce", 100, "debounce delay in milliseconds (prevents rapid regeneration)")

	watchCmd.MarkFlagRequired("out")

	// Add subcommands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(versionCmd)
}
