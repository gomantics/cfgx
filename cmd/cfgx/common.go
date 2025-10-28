package main

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	inputFile   string
	outputFile  string
	packageName string
	noEnv       bool
	maxFileSize string
	mode        string
)

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
