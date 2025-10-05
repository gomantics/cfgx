// Package pkgutil provides utilities for working with Go packages.
package pkgutil

import (
	"path/filepath"
)

// InferName attempts to infer a package name from an output file path.
// It uses the directory name as the package name, with special handling for
// common directory names like "internal", "pkg", and "lib".
func InferName(outputPath string) string {
	// Get the directory name
	dir := filepath.Dir(outputPath)

	// If it's current directory, use "config"
	if dir == "." || dir == "" {
		return "config"
	}

	// Use the base name of the directory
	base := filepath.Base(dir)

	// Clean up common path elements
	switch base {
	case "internal", "pkg", "lib":
		// Look one level deeper
		parent := filepath.Dir(dir)
		if parent != "." && parent != "" {
			base = filepath.Base(parent)
		}
	}

	// If still empty or a path separator, default to "config"
	if base == "" || base == "." || base == "/" {
		return "config"
	}

	return base
}
