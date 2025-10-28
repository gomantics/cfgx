package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

var (
	keysOnly   bool
	diffFormat string
)

var diffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Compare two TOML files and highlight differences",
	Long: `Compare two TOML configuration files and show what's different.

This is useful for understanding changes between environments (dev vs prod) 
or between base and override configurations.`,
	Example: `  # Compare two config files
  cfgx diff config.dev.toml config.prod.toml

  # Show only changed keys
  cfgx diff config.dev.toml config.prod.toml --keys-only

  # Output as JSON for scripting
  cfgx diff base.toml override.toml --format json`,
	Args: cobra.ExactArgs(2),
	Run:  runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&keysOnly, "keys-only", false, "Show only the keys that differ, not their values")
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", "Output format: text or json")
}

func runDiff(cmd *cobra.Command, args []string) {
	file1, file2 := args[0], args[1]

	// Parse both files
	data1, err := parseTomlFile(file1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file1, err)
		os.Exit(1)
	}

	data2, err := parseTomlFile(file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file2, err)
		os.Exit(1)
	}

	// Compute differences
	diffs := computeDiffs(data1, data2, "")

	// Output based on format
	switch diffFormat {
	case "json":
		outputJSON(diffs, file1, file2)
	case "text":
		outputText(diffs, file1, file2)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (use 'text' or 'json')\n", diffFormat)
		os.Exit(1)
	}

	// Exit successfully - differences are not errors
}

// parseTomlFile parses a TOML file into a map
func parseTomlFile(filename string) (map[string]any, error) {
	var data map[string]any
	_, err := toml.DecodeFile(filename, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DiffType represents the type of difference
type DiffType string

const (
	DiffChanged DiffType = "changed"
	DiffAdded   DiffType = "added"
	DiffRemoved DiffType = "removed"
)

// Diff represents a difference between two configs
type Diff struct {
	Key    string   `json:"key"`
	Type   DiffType `json:"type"`
	Value1 any      `json:"value1,omitempty"`
	Value2 any      `json:"value2,omitempty"`
}

// computeDiffs recursively compares two maps and returns differences
func computeDiffs(data1, data2 map[string]any, prefix string) []Diff {
	var diffs []Diff

	// Get all keys from both maps
	allKeys := make(map[string]bool)
	for k := range data1 {
		allKeys[k] = true
	}
	for k := range data2 {
		allKeys[k] = true
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		val1, exists1 := data1[key]
		val2, exists2 := data2[key]

		// Key only in data2 (added)
		if !exists1 && exists2 {
			diffs = append(diffs, Diff{
				Key:    fullKey,
				Type:   DiffAdded,
				Value2: val2,
			})
			continue
		}

		// Key only in data1 (removed)
		if exists1 && !exists2 {
			diffs = append(diffs, Diff{
				Key:    fullKey,
				Type:   DiffRemoved,
				Value1: val1,
			})
			continue
		}

		// Key exists in both - check if values differ
		if exists1 && exists2 {
			// If both are maps, recurse
			map1, isMap1 := val1.(map[string]any)
			map2, isMap2 := val2.(map[string]any)

			if isMap1 && isMap2 {
				// Recursively compare nested maps
				nestedDiffs := computeDiffs(map1, map2, fullKey)
				diffs = append(diffs, nestedDiffs...)
			} else if !deepEqual(val1, val2) {
				// Values are different
				diffs = append(diffs, Diff{
					Key:    fullKey,
					Type:   DiffChanged,
					Value1: val1,
					Value2: val2,
				})
			}
		}
	}

	return diffs
}

// deepEqual compares two values for equality
func deepEqual(v1, v2 any) bool {
	// Use fmt.Sprintf to compare values as strings
	// This handles most TOML types correctly
	return fmt.Sprintf("%v", v1) == fmt.Sprintf("%v", v2)
}

// outputText outputs differences in human-readable text format
func outputText(diffs []Diff, file1, file2 string) {
	if len(diffs) == 0 {
		fmt.Println("No differences found.")
		return
	}

	fmt.Printf("Differences between %s and %s:\n\n", file1, file2)

	for _, diff := range diffs {
		switch diff.Type {
		case DiffChanged:
			if keysOnly {
				fmt.Printf("  ~ %s\n", diff.Key)
			} else {
				fmt.Printf("  %s\n", diff.Key)
				fmt.Printf("    - %s     (%s)\n", formatValue(diff.Value1), file1)
				fmt.Printf("    + %s     (%s)\n", formatValue(diff.Value2), file2)
				fmt.Println()
			}
		case DiffAdded:
			if keysOnly {
				fmt.Printf("  + %s\n", diff.Key)
			} else {
				fmt.Printf("  + %s = %s     (only in %s)\n", diff.Key, formatValue(diff.Value2), file2)
			}
		case DiffRemoved:
			if keysOnly {
				fmt.Printf("  - %s\n", diff.Key)
			} else {
				fmt.Printf("  - %s = %s     (only in %s)\n", diff.Key, formatValue(diff.Value1), file1)
			}
		}
	}
}

// formatValue formats a value for display
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		// Quote strings
		return fmt.Sprintf(`"%s"`, val)
	case []any:
		// Format arrays
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		// For nested maps, just show it's a table
		return "{...}"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// outputJSON outputs differences in JSON format
func outputJSON(diffs []Diff, file1, file2 string) {
	output := map[string]any{
		"file1":       file1,
		"file2":       file2,
		"differences": diffs,
		"count":       len(diffs),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
