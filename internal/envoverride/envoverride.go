// Package envoverride provides environment variable override functionality for TOML data.
package envoverride

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Apply applies environment variable overrides to TOML data.
// Environment variables follow the pattern: CONFIG_<SECTION>_<KEY>
func Apply(data map[string]any) error {
	for key, value := range data {
		prefix := "CONFIG_" + strings.ToUpper(key)

		switch val := value.(type) {
		case map[string]any:
			// Nested map - recursively apply overrides
			if err := applyNested(val, prefix); err != nil {
				return fmt.Errorf("error in section %s: %w", key, err)
			}
		default:
			// Top-level value - check for override
			envKey := prefix
			if envVal := os.Getenv(envKey); envVal != "" {
				converted, err := convertValue(envVal, value)
				if err != nil {
					return fmt.Errorf("invalid value for %s: %w", envKey, err)
				}
				data[key] = converted
			}
		}
	}

	return nil
}

// applyNested applies environment variable overrides to nested maps
func applyNested(data map[string]any, prefix string) error {
	for key, value := range data {
		envKey := prefix + "_" + strings.ToUpper(key)

		switch val := value.(type) {
		case map[string]any:
			// Further nested map
			if err := applyNested(val, envKey); err != nil {
				return err
			}
		case []any:
			// Arrays - check for override
			// For arrays, we support comma-separated values for primitives
			if envVal := os.Getenv(envKey); envVal != "" {
				if len(val) > 0 {
					// Determine element type from first element
					converted, err := convertArray(envVal, val[0])
					if err != nil {
						return fmt.Errorf("invalid array value for %s: %w", envKey, err)
					}
					data[key] = converted
				}
			}
		default:
			// Primitive value - check for override
			if envVal := os.Getenv(envKey); envVal != "" {
				converted, err := convertValue(envVal, value)
				if err != nil {
					return fmt.Errorf("invalid value for %s: %w", envKey, err)
				}
				data[key] = converted
			}
		}
	}

	return nil
}

// convertValue converts an environment variable string to match the type of the original value
func convertValue[T any](envVal string, originalVal T) (any, error) {
	switch any(originalVal).(type) {
	case string:
		return envVal, nil

	case int64, int:
		v, err := strconv.ParseInt(envVal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected integer: %w", err)
		}
		return v, nil

	case float64:
		v, err := strconv.ParseFloat(envVal, 64)
		if err != nil {
			return nil, fmt.Errorf("expected float: %w", err)
		}
		return v, nil

	case bool:
		v, err := strconv.ParseBool(envVal)
		if err != nil {
			return nil, fmt.Errorf("expected boolean: %w", err)
		}
		return v, nil

	default:
		return envVal, nil
	}
}

// convertArray converts a comma-separated environment variable to an array
func convertArray[T any](envVal string, sampleElem T) (any, error) {
	parts := strings.Split(envVal, ",")
	result := make([]any, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		converted, err := convertValue(part, sampleElem)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}
