package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/gomantics/cfgx"
)

var (
	debounce int
)

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

		// Setup context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handler for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		// Debounce timer with mutex for thread-safe access
		var (
			debounceTimer *time.Timer
			timerMu       sync.Mutex
		)
		debounceDuration := time.Duration(debounce) * time.Millisecond

		// Track if a file re-add goroutine is already running
		var readdInProgress atomic.Bool

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
					timerMu.Lock()
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
					timerMu.Unlock()
				} else if event.Has(fsnotify.Remove) {
					// File was removed - common with some editors (vim, etc.)
					// Try to re-add the watcher when file is recreated
					fmt.Println("File removed, waiting for recreation...")
					// Remove from watcher (it's already gone)
					watcher.Remove(absInputFile)

					// Only spawn one re-add goroutine at a time
					if readdInProgress.CompareAndSwap(false, true) {
						go func() {
							defer readdInProgress.Store(false)

							for i := 0; i < 10; i++ {
								select {
								case <-ctx.Done():
									// Context cancelled, exit gracefully
									return
								case <-time.After(100 * time.Millisecond):
									if err := watcher.Add(absInputFile); err == nil {
										fmt.Println("File recreated, watching again...")
										return
									}
								}
							}
							fmt.Fprintf(os.Stderr, "Warning: Could not re-watch file after removal\n")
						}()
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)

			case <-sigChan:
				fmt.Println("\nStopping watch...")
				timerMu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				timerMu.Unlock()
				return nil
			}
		}
	},
	SilenceUsage: true,
}

func init() {
	// Watch command flags (reuse generate flags)
	watchCmd.Flags().StringVarP(&inputFile, "in", "i", "config.toml", "input TOML file")
	watchCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output Go file (required)")
	watchCmd.Flags().StringVarP(&packageName, "pkg", "p", "", "package name (default: inferred from output path or 'config')")
	watchCmd.Flags().BoolVar(&noEnv, "no-env", false, "disable environment variable overrides")
	watchCmd.Flags().StringVar(&maxFileSize, "max-file-size", "1MB", "maximum file size for file: references (e.g., 10MB, 1GB, 512KB)")
	watchCmd.Flags().StringVar(&mode, "mode", "static", "generation mode: 'static' (values baked at build time) or 'getter' (runtime env var overrides)")
	watchCmd.Flags().IntVar(&debounce, "debounce", 100, "debounce delay in milliseconds (prevents rapid regeneration)")

	watchCmd.MarkFlagRequired("out")
}
