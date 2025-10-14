package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isFileReference checks if a string value is a file reference (starts with "file:").
func (g *Generator) isFileReference(s string) bool {
	return strings.HasPrefix(s, "file:")
}

// loadFileContent reads a file and returns its contents as bytes.
// The file path is resolved relative to the inputDir.
// Returns an error if the file doesn't exist, can't be read, or exceeds maxFileSize.
func (g *Generator) loadFileContent(filePath string) ([]byte, error) {
	// Strip "file:" prefix
	relativePath := strings.TrimPrefix(filePath, "file:")

	// Resolve path relative to input directory
	var resolvedPath string
	if g.inputDir != "" {
		resolvedPath = filepath.Join(g.inputDir, relativePath)
	} else {
		resolvedPath = relativePath
	}

	// Check file exists and get size
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s (referenced in config)", resolvedPath)
		}
		return nil, fmt.Errorf("failed to stat file %s: %w", resolvedPath, err)
	}

	// Check file size
	if g.maxFileSize > 0 && fileInfo.Size() > g.maxFileSize {
		return nil, fmt.Errorf("file %s exceeds max size %d bytes (actual: %d bytes)",
			resolvedPath, g.maxFileSize, fileInfo.Size())
	}

	// Read file
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", resolvedPath, err)
	}

	return content, nil
}
