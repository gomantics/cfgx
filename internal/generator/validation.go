package generator

import (
	"slices"
	"time"
)

// validateFileReferences recursively validates all file: references in the data.
// This ensures all referenced files exist and don't exceed size limits before generation.
func (g *Generator) validateFileReferences(data map[string]any) error {
	for _, v := range data {
		if err := g.validateFileReferencesValue(v); err != nil {
			return err
		}
	}
	return nil
}

// validateFileReferencesValue validates file references in a single value.
func (g *Generator) validateFileReferencesValue(v any) error {
	switch val := v.(type) {
	case string:
		if g.isFileReference(val) {
			// Try to load the file to validate it exists and size is OK
			_, err := g.loadFileContent(val)
			if err != nil {
				return err
			}
		}
	case map[string]any:
		return g.validateFileReferences(val)
	case []any:
		for _, item := range val {
			if err := g.validateFileReferencesValue(item); err != nil {
				return err
			}
		}
	case []map[string]any:
		for _, m := range val {
			if err := g.validateFileReferences(m); err != nil {
				return err
			}
		}
	}
	return nil
}

// needsTimeImport checks if any value in the data map is a duration string,
// recursively traversing nested maps and arrays to determine if the generated
// code needs to import the "time" package.
func (g *Generator) needsTimeImport(data map[string]any) bool {
	for _, v := range data {
		if g.needsTimeImportValue(v) {
			return true
		}
	}
	return false
}

func (g *Generator) needsTimeImportValue(v any) bool {
	switch val := v.(type) {
	case string:
		// Check if string is a valid duration
		if g.isDurationString(val) {
			return true
		}
	case map[string]any:
		return g.needsTimeImport(val)
	case []any:
		if slices.ContainsFunc(val, g.needsTimeImportValue) {
			return true
		}
	case []map[string]any:
		if slices.ContainsFunc(val, g.needsTimeImport) {
			return true
		}
	}
	return false
}

// isDurationString checks if a string can be parsed as a time.Duration.
func (g *Generator) isDurationString(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}
