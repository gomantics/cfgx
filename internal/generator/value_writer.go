package generator

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// toGoType converts a value to its Go type string representation. This function
// inspects the runtime type of a value and returns the corresponding Go type as a string.
//
// For primitive types (string, int64, float64, bool), it returns the standard type name.
// For slices, it recursively determines the element type. For maps and []map[string]any,
// it returns placeholder strings ("struct", "[]struct") that will be replaced with actual
// struct type names in context by the calling code.
func (g *Generator) toGoType(v any) string {
	switch val := v.(type) {
	case string:
		// Check if this is a file reference
		if g.isFileReference(val) {
			return "[]byte"
		}
		// Check if this is a duration string
		if g.isDurationString(val) {
			return "time.Duration"
		}
		return "string"
	case int64:
		return "int64"
	case int:
		return "int64"
	case float64:
		return "float64"
	case bool:
		return "bool"
	case []any:
		if len(val) > 0 {
			elemType := g.toGoType(val[0])
			return "[]" + elemType
		}
		return "[]any"
	case []map[string]any:
		// This will be replaced with the actual struct type name in context
		return "[]struct"
	case map[string]any:
		// This will be replaced with the actual struct type name in context
		return "struct"
	default:
		return "any"
	}
}

// writeValue writes a Go value literal to the buffer. This function handles the
// serialization of various Go types into their source code representation.
//
// Strings are quoted, numbers are formatted appropriately, duration strings are
// parsed and written as duration literals, and arrays are handled recursively.
// This ensures the generated code is valid Go syntax that can be compiled directly.
// The indent parameter is used for proper formatting of multi-line values like byte arrays.
func (g *Generator) writeValue(buf *bytes.Buffer, v any) {
	g.writeValueWithIndent(buf, v, 0)
}

// writeValueWithIndent is the internal implementation of writeValue with indent support.
func (g *Generator) writeValueWithIndent(buf *bytes.Buffer, v any, indent int) {
	switch val := v.(type) {
	case string:
		// Check if this is a file reference
		if g.isFileReference(val) {
			// File was already validated in validateFileReferences, so this should not fail
			content, err := g.loadFileContent(val)
			if err != nil {
				// This should never happen if validation passed
				fmt.Fprintf(buf, "[]byte{} /* unexpected error: %s */", err)
				return
			}
			g.writeByteArrayLiteral(buf, content, indent)
			return
		}
		// Check if this is a duration string
		if g.isDurationString(val) {
			g.writeDurationLiteral(buf, val)
		} else {
			fmt.Fprintf(buf, "%q", val)
		}
	case int64:
		fmt.Fprintf(buf, "%d", val)
	case int:
		fmt.Fprintf(buf, "%d", val)
	case float64:
		fmt.Fprintf(buf, "%g", val)
	case bool:
		fmt.Fprintf(buf, "%t", val)
	case []any:
		g.writeArray(buf, val)
	default:
		buf.WriteString("nil")
	}
}

// writeByteArrayLiteral writes a byte array in idiomatic Go hex format.
// Format: []byte{0x2d, 0x2d, ...} with 12 bytes per line for readability.
// The indent parameter controls indentation level for proper formatting in nested contexts.
func (g *Generator) writeByteArrayLiteral(buf *bytes.Buffer, data []byte, indent int) {
	if len(data) == 0 {
		buf.WriteString("[]byte{}")
		return
	}

	buf.WriteString("[]byte{\n")
	indentStr := strings.Repeat("\t", indent+1)

	// Write 12 bytes per line (each byte is "0xXX, " = 6 chars, 12*6 = 72 chars)
	const bytesPerLine = 12
	for i := 0; i < len(data); i++ {
		if i%bytesPerLine == 0 {
			buf.WriteString(indentStr)
		}

		fmt.Fprintf(buf, "0x%02x", data[i])

		if i < len(data)-1 {
			buf.WriteString(", ")
		}

		if i%bytesPerLine == bytesPerLine-1 && i < len(data)-1 {
			buf.WriteString("\n")
		}
	}

	buf.WriteString(",\n")
	buf.WriteString(strings.Repeat("\t", indent))
	buf.WriteString("}")
}

// writeDurationLiteral parses a duration string at generation time and writes
// it as a duration literal in a human-readable format using time constants.
// Complex durations like '2h30m' are decomposed into multiple time constants
// (e.g., 2*time.Hour + 30*time.Minute) for better readability.
func (g *Generator) writeDurationLiteral(buf *bytes.Buffer, s string) {
	d, err := time.ParseDuration(s)
	if err != nil {
		// This should never happen since isDurationString already validated it
		fmt.Fprintf(buf, "time.Duration(0) /* invalid: %s */", s)
		return
	}

	if d == 0 {
		buf.WriteString("0")
		return
	}

	// Decompose duration into components from largest to smallest
	components := []struct {
		unit time.Duration
		name string
	}{
		{time.Hour, "time.Hour"},
		{time.Minute, "time.Minute"},
		{time.Second, "time.Second"},
		{time.Millisecond, "time.Millisecond"},
		{time.Microsecond, "time.Microsecond"},
		{time.Nanosecond, "time.Nanosecond"},
	}

	remaining := d
	parts := []string{}

	for _, comp := range components {
		if remaining >= comp.unit {
			count := remaining / comp.unit
			if count > 0 {
				parts = append(parts, fmt.Sprintf("%d*%s", count, comp.name))
				remaining = remaining % comp.unit
			}
		}
	}

	if len(parts) == 0 {
		// Should not happen for non-zero durations, but handle it
		buf.WriteString("0")
		return
	}

	// Join parts with " + "
	// Note: gofmt will add spaces around * for simple expressions (e.g., "30 * time.Second")
	// but keep them compact in complex expressions (e.g., "2*time.Hour + 30*time.Minute")
	buf.WriteString(strings.Join(parts, " + "))
}

// writeArray writes an array literal in Go slice syntax. The function infers the
// element type from the first element and generates a typed slice literal.
//
// Empty arrays are written as "nil". Non-empty arrays are written in the format:
// []Type{elem1, elem2, ...} with elements separated by commas and spaces.
func (g *Generator) writeArray(buf *bytes.Buffer, arr []any) {
	if len(arr) == 0 {
		buf.WriteString("nil")
		return
	}

	elemType := g.toGoType(arr[0])
	fmt.Fprintf(buf, "[]%s{", elemType)

	for i, item := range arr {
		if i > 0 {
			fmt.Fprintf(buf, ", ")
		}
		g.writeValue(buf, item)
	}

	buf.WriteString("}")
}
