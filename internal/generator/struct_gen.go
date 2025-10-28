package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/gomantics/sx"
)

// generateStructsAndVars orchestrates the generation of all struct type definitions
// and variable declarations from the parsed TOML data. It processes the data in two
// phases:
//
//  1. Collects all struct definitions (including nested ones) by traversing the data
//     and building a complete map of struct types needed.
//  2. Generates the Go code for structs first (sorted alphabetically for deterministic
//     output), then generates variable declarations with their initializations.
//
// This function handles top-level tables, arrays of tables, and nested structures,
// ensuring proper naming conventions (e.g., "DatabaseConfig", "ServersItem") and
// correct type references.
func (g *Generator) generateStructsAndVars(buf *bytes.Buffer, data map[string]any) error {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic output

	allStructs := make(map[string]map[string]any)
	for _, key := range keys {
		if m, ok := data[key].(map[string]any); ok {
			structName := sx.PascalCase(key) + "Config"
			g.collectNestedStructs(allStructs, structName, m)
		} else if arr, ok := data[key].([]map[string]any); ok {
			if len(arr) > 0 {
				structName := sx.PascalCase(key) + "Item"
				g.collectNestedStructs(allStructs, structName, arr[0])
			}
		}
	}

	structNames := make([]string, 0, len(allStructs))
	for name := range allStructs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		fields := allStructs[name]
		if err := g.generateStruct(buf, name, fields); err != nil {
			return err
		}
		buf.WriteString("\n\n")
	}

	buf.WriteString("var (\n")

	for _, key := range keys {
		varName := sx.PascalCase(key)
		value := data[key]

		switch val := value.(type) {
		case map[string]any:
			structName := sx.PascalCase(key) + "Config"
			fmt.Fprintf(buf, "\t%s = %s", varName, structName)
			if err := g.generateStructInit(buf, structName, val, 0); err != nil {
				return err
			}
			buf.WriteString("\n")
		case []map[string]any:
			if len(val) > 0 {
				structName := sx.PascalCase(key) + "Item"
				fmt.Fprintf(buf, "\t%s = []%s", varName, structName)
				if err := g.writeArrayOfTablesInit(buf, structName, val, 0); err != nil {
					return err
				}
				buf.WriteString("\n")
			} else {
				fmt.Fprintf(buf, "\t%s []%sItem\n", varName, sx.PascalCase(key))
			}
		case []any:
			if len(val) > 0 {
				if _, ok := val[0].(map[string]any); ok {
					structName := sx.PascalCase(key) + "Item"
					fmt.Fprintf(buf, "\t%s = []%s", varName, structName)
					if err := g.writeArrayOfTablesInit(buf, structName, val, 0); err != nil {
						return err
					}
					buf.WriteString("\n")
				} else {
					goType := g.toGoType(value)
					fmt.Fprintf(buf, "\t%s %s = ", varName, goType)
					g.writeValue(buf, value)
					buf.WriteString("\n")
				}
			} else {
				goType := g.toGoType(value)
				fmt.Fprintf(buf, "\t%s %s\n", varName, goType)
			}
		default:
			// Generate simple variable
			goType := g.toGoType(value)
			fmt.Fprintf(buf, "\t%s %s = ", varName, goType)
			g.writeValue(buf, value)
			buf.WriteString("\n")
		}
	}

	buf.WriteString(")\n")

	return nil
}

// collectNestedStructs recursively collects all struct definitions needed for the
// generated code. It traverses nested maps and arrays to discover all struct types
// that must be defined.
//
// The function builds unique struct names by concatenating parent and child names
// (e.g., "DatabaseConfig" -> "DatabaseCredentialsConfig" for nested credentials).
// It handles:
//   - Nested maps (inline tables) - suffixed with "Config"
//   - Arrays of maps (array of tables) - suffixed with "Item"
//
// The structs map is populated with name->fields mapping, ensuring each struct type
// is only processed once (deduplication via existence check).
func (g *Generator) collectNestedStructs(structs map[string]map[string]any, name string, data map[string]any) {
	if _, exists := structs[name]; exists {
		return
	}

	structs[name] = data

	for key, val := range data {
		switch v := val.(type) {
		case map[string]any:
			nestedName := stripSuffix(name) + sx.PascalCase(key) + "Config"
			g.collectNestedStructs(structs, nestedName, v)
		case []any:
			// Check if it's an array of maps
			if len(v) > 0 {
				if m, ok := v[0].(map[string]any); ok {
					nestedName := stripSuffix(name) + sx.PascalCase(key) + "Item"
					g.collectNestedStructs(structs, nestedName, m)
				}
			}
		case []map[string]any:
			if len(v) > 0 {
				nestedName := stripSuffix(name) + sx.PascalCase(key) + "Item"
				g.collectNestedStructs(structs, nestedName, v[0])
			}
		}
	}
}

// generateStruct generates a struct type definition with properly typed fields.
// Field names are converted to Go-idiomatic CamelCase, and field types are determined
// based on the value types in the TOML data.
//
// For nested structures, the function constructs type names by prefixing the parent
// struct name to maintain uniqueness (e.g., "DatabaseConfig" with a "server" field
// becomes "DatabaseConfigServerConfig" type).
//
// Fields are sorted alphabetically for deterministic output.
func (g *Generator) generateStruct(buf *bytes.Buffer, name string, fields map[string]any) error {
	fmt.Fprintf(buf, "type %s struct {\n", name)

	fieldNames := make([]string, 0, len(fields))
	for k := range fields {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		value := fields[fieldName]
		goFieldName := sx.PascalCase(fieldName)
		goType := g.toGoType(value)

		// Handle nested structs - prefix with parent struct name
		if _, ok := value.(map[string]any); ok {
			goType = stripSuffix(name) + sx.PascalCase(fieldName) + "Config"
		} else if arr, ok := value.([]any); ok && len(arr) > 0 {
			if _, isMap := arr[0].(map[string]any); isMap {
				goType = "[]" + stripSuffix(name) + sx.PascalCase(fieldName) + "Item"
			}
		} else if arr, ok := value.([]map[string]any); ok && len(arr) > 0 {
			goType = "[]" + stripSuffix(name) + sx.PascalCase(fieldName) + "Item"
		}

		fmt.Fprintf(buf, "\t%s %s\n", goFieldName, goType)
	}

	buf.WriteString("}")
	return nil
}

// generateStructInit generates struct initialization code with proper indentation
// and nested struct literals. This function recursively creates the initialization
// syntax for complex nested structures.
//
// For nested maps, it generates inline struct literals with the appropriate type name.
// For arrays of structs, it delegates to writeArrayOfStructs or handles simple arrays.
// Simple values are written as literals using writeValue.
//
// The indent parameter controls the indentation level for proper formatting of nested
// structures. Fields are sorted alphabetically for deterministic output.
func (g *Generator) generateStructInit(buf *bytes.Buffer, parentStructName string, data map[string]any, indent int) error {
	buf.WriteString("{\n")

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic output

	indentStr := strings.Repeat("\t", indent+1)
	for _, key := range keys {
		value := data[key]
		fieldName := sx.PascalCase(key)

		buf.WriteString(indentStr)
		fmt.Fprintf(buf, "%s: ", fieldName)

		switch val := value.(type) {
		case map[string]any:
			structType := stripSuffix(parentStructName) + sx.PascalCase(key) + "Config"
			buf.WriteString(structType)
			if err := g.generateStructInit(buf, structType, val, indent+1); err != nil {
				return err
			}
		case []any:
			if len(val) > 0 {
				if _, ok := val[0].(map[string]any); ok {
					g.writeArrayOfStructs(buf, val, indent+1)
				} else {
					g.writeValueWithIndent(buf, value, indent+1)
				}
			} else {
				g.writeValueWithIndent(buf, value, indent+1)
			}
		case []map[string]any:
			g.writeArrayOfStructs(buf, val, indent+1)
		default:
			g.writeValueWithIndent(buf, value, indent+1)
		}

		buf.WriteString(",\n")
	}

	buf.WriteString(strings.Repeat("\t", indent))
	buf.WriteString("}")
	return nil
}

// writeArrayOfTablesInit writes an array initialization for top-level tables,
// specifically handling TOML's [[array.of.tables]] syntax. This generates code
// for slices of structs where each element is initialized with its fields.
//
// The function handles both []any and []map[string]any types from the TOML parser,
// generating struct literals with proper indentation. Each element is initialized
// by calling generateStructInit for the nested structure.
//
// The type name is omitted from each element to comply with gofmt -s simplification rules.
//
// Example output:
//
//	[]ServerItem{
//	    {Host: "localhost", Port: 8080},
//	    {Host: "example.com", Port: 443},
//	}
func (g *Generator) writeArrayOfTablesInit(buf *bytes.Buffer, structName string, arr any, indent int) error {
	buf.WriteString("{\n")
	indentStr := strings.Repeat("\t", indent+1)

	switch val := arr.(type) {
	case []any:
		for _, item := range val {
			if m, ok := item.(map[string]any); ok {
				buf.WriteString(indentStr)
				// Omit type name for gofmt -s compliance
				if err := g.generateStructInit(buf, structName, m, indent+1); err != nil {
					return err
				}
				buf.WriteString(",\n")
			}
		}
	case []map[string]any:
		for _, m := range val {
			buf.WriteString(indentStr)
			// Omit type name for gofmt -s compliance
			if err := g.generateStructInit(buf, structName, m, indent+1); err != nil {
				return err
			}
			buf.WriteString(",\n")
		}
	}

	buf.WriteString(strings.Repeat("\t", indent))
	buf.WriteString("}")
	return nil
}

// writeArrayOfStructs writes an array of struct initializations using compact inline
// syntax. Unlike writeArrayOfTablesInit, this generates inline struct literals without
// the type name prefix, making it more suitable for deeply nested structures.
//
// Example output:
//
//	{
//	    {Host: "localhost", Port: 8080},
//	    {Host: "example.com", Port: 443},
//	}
//
// Fields within each struct are written in sorted order and separated by commas on a
// single line. This function handles both []any and []map[string]any input types.
func (g *Generator) writeArrayOfStructs(buf *bytes.Buffer, arr any, indent int) {
	buf.WriteString("{\n")
	indentStr := strings.Repeat("\t", indent+1)

	switch val := arr.(type) {
	case []any:
		for _, item := range val {
			if m, ok := item.(map[string]any); ok {
				buf.WriteString(indentStr)
				buf.WriteString("{")
				// Inline struct fields
				keys := make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				for i, k := range keys {
					if i > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(sx.PascalCase(k))
					buf.WriteString(": ")
					g.writeValue(buf, m[k])
				}
				buf.WriteString("},\n")
			}
		}
	case []map[string]any:
		for _, m := range val {
			buf.WriteString(indentStr)
			buf.WriteString("{")
			// Inline struct fields
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for i, k := range keys {
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(sx.PascalCase(k))
				buf.WriteString(": ")
				g.writeValue(buf, m[k])
			}
			buf.WriteString("},\n")
		}
	}

	buf.WriteString(strings.Repeat("\t", indent))
	buf.WriteString("}")
}

// generateStructsAndGetters generates empty struct types and getter methods for getter mode.
// This is an alternative to generateStructsAndVars that creates methods instead of fields.
func (g *Generator) generateStructsAndGetters(buf *bytes.Buffer, data map[string]any) error {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic output

	// Collect all struct names
	allStructs := make(map[string]map[string]any)
	for _, key := range keys {
		if m, ok := data[key].(map[string]any); ok {
			structName := sx.PascalCase(key) + "Config"
			g.collectNestedStructsForGetters(allStructs, structName, m)
		} else if arr, ok := data[key].([]map[string]any); ok {
			if len(arr) > 0 {
				structName := sx.PascalCase(key) + "Item"
				g.collectNestedStructsForGetters(allStructs, structName, arr[0])
			}
		}
	}

	// Generate empty struct types (no fields, just methods)
	structNames := make([]string, 0, len(allStructs))
	for name := range allStructs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		fmt.Fprintf(buf, "type %s struct{}\n\n", name)
	}

	// Generate getter methods for each struct
	generated := make(map[string]bool)
	for _, name := range structNames {
		fields := allStructs[name]
		if err := g.generateGetterMethods(buf, name, fields, "", generated); err != nil {
			return err
		}
	}

	// Generate var declarations
	buf.WriteString("var (\n")
	for _, key := range keys {
		varName := sx.PascalCase(key)
		value := data[key]

		switch value.(type) {
		case map[string]any:
			structName := sx.PascalCase(key) + "Config"
			fmt.Fprintf(buf, "\t%s %s\n", varName, structName)
		case []map[string]any:
			structName := sx.PascalCase(key) + "Item"
			fmt.Fprintf(buf, "\t%s []%s\n", varName, structName)
		case []any:
			goType := g.toGoType(value)
			fmt.Fprintf(buf, "\t%s %s\n", varName, goType)
		default:
			goType := g.toGoType(value)
			fmt.Fprintf(buf, "\t%s %s\n", varName, goType)
		}
	}
	buf.WriteString(")\n")

	return nil
}

// collectNestedStructsForGetters is similar to collectNestedStructs but for getter mode.
func (g *Generator) collectNestedStructsForGetters(structs map[string]map[string]any, name string, data map[string]any) {
	if _, exists := structs[name]; exists {
		return
	}

	structs[name] = data

	for key, val := range data {
		switch v := val.(type) {
		case map[string]any:
			nestedName := stripSuffix(name) + sx.PascalCase(key) + "Config"
			g.collectNestedStructsForGetters(structs, nestedName, v)
		case []any:
			if len(v) > 0 {
				if m, ok := v[0].(map[string]any); ok {
					nestedName := stripSuffix(name) + sx.PascalCase(key) + "Item"
					g.collectNestedStructsForGetters(structs, nestedName, m)
				}
			}
		case []map[string]any:
			if len(v) > 0 {
				nestedName := stripSuffix(name) + sx.PascalCase(key) + "Item"
				g.collectNestedStructsForGetters(structs, nestedName, v[0])
			}
		}
	}
}

// generateGetterMethods generates getter methods for a struct type.
func (g *Generator) generateGetterMethods(buf *bytes.Buffer, structName string, fields map[string]any, envPrefix string, generated map[string]bool) error {
	// Skip if already generated
	if generated[structName] {
		return nil
	}
	generated[structName] = true

	fieldNames := make([]string, 0, len(fields))
	for k := range fields {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		value := fields[fieldName]
		goFieldName := sx.PascalCase(fieldName)

		// Build env var name
		var envVarName string
		if envPrefix == "" {
			envVarName = g.envVarName(structName, fieldName)
		} else {
			envVarName = envPrefix + "_" + strings.ToUpper(fieldName)
		}

		// Handle nested structs - they need their own getter methods
		if nestedMap, ok := value.(map[string]any); ok {
			nestedStructName := stripSuffix(structName) + sx.PascalCase(fieldName) + "Config"
			// Generate method that returns nested struct
			fmt.Fprintf(buf, "func (%s) %s() %s {\n", structName, goFieldName, nestedStructName)
			fmt.Fprintf(buf, "\treturn %s{}\n", nestedStructName)
			buf.WriteString("}\n\n")
			// Generate methods for nested struct (pass along env prefix)
			if err := g.generateGetterMethods(buf, nestedStructName, nestedMap, envVarName, generated); err != nil {
				return err
			}
			continue
		}

		// Handle arrays of structs - for now, return empty slice (limitation)
		if arr, ok := value.([]any); ok && len(arr) > 0 {
			if _, isMap := arr[0].(map[string]any); isMap {
				nestedStructName := stripSuffix(structName) + sx.PascalCase(fieldName) + "Item"
				goType := "[]" + nestedStructName
				// For arrays of structs, return default empty value
				fmt.Fprintf(buf, "func (%s) %s() %s {\n", structName, goFieldName, goType)
				fmt.Fprintf(buf, "\t// Arrays of structs cannot be overridden via env vars\n")
				fmt.Fprintf(buf, "\treturn nil\n")
				buf.WriteString("}\n\n")
				continue
			}
		}

		if arr, ok := value.([]map[string]any); ok && len(arr) > 0 {
			nestedStructName := stripSuffix(structName) + sx.PascalCase(fieldName) + "Item"
			goType := "[]" + nestedStructName
			fmt.Fprintf(buf, "func (%s) %s() %s {\n", structName, goFieldName, goType)
			fmt.Fprintf(buf, "\t// Arrays of structs cannot be overridden via env vars\n")
			fmt.Fprintf(buf, "\treturn nil\n")
			buf.WriteString("}\n\n")
			continue
		}

		// Get the Go type
		goType := g.toGoType(value)

		// Generate getter method based on type
		if err := g.generateGetterMethod(buf, structName, goFieldName, goType, envVarName, value); err != nil {
			return err
		}
	}

	return nil
}

// generateGetterMethod generates a single getter method with env var override.
func (g *Generator) generateGetterMethod(buf *bytes.Buffer, structName, fieldName, goType, envVarName string, defaultValue any) error {
	fmt.Fprintf(buf, "func (%s) %s() %s {\n", structName, fieldName, goType)

	// Special handling for []byte (file references) - check for file path in env var
	if goType == "[]byte" {
		buf.WriteString("\t// Check for file path to load\n")
		fmt.Fprintf(buf, "\tif path := os.Getenv(%q); path != \"\" {\n", envVarName)
		buf.WriteString("\t\tif data, err := os.ReadFile(path); err == nil {\n")
		buf.WriteString("\t\t\treturn data\n")
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}\n")
		// Write default value
		buf.WriteString("\treturn ")
		g.writeValue(buf, defaultValue)
		buf.WriteString("\n")
		buf.WriteString("}\n\n")
		return nil
	}

	// For other types, check env var with type conversion
	fmt.Fprintf(buf, "\tif v := os.Getenv(%q); v != \"\" {\n", envVarName)

	// Generate type-specific parsing
	switch goType {
	case "string":
		buf.WriteString("\t\treturn v\n")
	case "int64":
		buf.WriteString("\t\tif i, err := strconv.ParseInt(v, 10, 64); err == nil {\n")
		buf.WriteString("\t\t\treturn i\n")
		buf.WriteString("\t\t}\n")
	case "float64":
		buf.WriteString("\t\tif f, err := strconv.ParseFloat(v, 64); err == nil {\n")
		buf.WriteString("\t\t\treturn f\n")
		buf.WriteString("\t\t}\n")
	case "bool":
		buf.WriteString("\t\tif b, err := strconv.ParseBool(v); err == nil {\n")
		buf.WriteString("\t\t\treturn b\n")
		buf.WriteString("\t\t}\n")
	case "time.Duration":
		buf.WriteString("\t\tif d, err := time.ParseDuration(v); err == nil {\n")
		buf.WriteString("\t\t\treturn d\n")
		buf.WriteString("\t\t}\n")
	default:
		// Handle arrays of primitives (for now, don't support env override)
		if strings.HasPrefix(goType, "[]") {
			buf.WriteString("\t\t// Array overrides not supported via env vars\n")
		}
	}

	buf.WriteString("\t}\n")

	// Write default value
	buf.WriteString("\treturn ")
	g.writeValue(buf, defaultValue)
	buf.WriteString("\n")

	buf.WriteString("}\n\n")
	return nil
}

// envVarName generates an environment variable name from a struct name and field name.
// Format: CONFIG_SECTION_KEY
func (g *Generator) envVarName(structName, fieldName string) string {
	// Remove "Config" or "Item" suffix from struct name
	section := stripSuffix(structName)
	section = strings.TrimSuffix(section, "Config")
	section = strings.TrimSuffix(section, "Item")

	// Convert to uppercase snake case
	sectionUpper := strings.ToUpper(sx.SnakeCase(section))
	fieldUpper := strings.ToUpper(fieldName)

	return "CONFIG_" + sectionUpper + "_" + fieldUpper
}
