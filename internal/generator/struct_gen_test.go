package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator_Types(t *testing.T) {
	tests := []struct {
		name string
		toml string
		want []string
	}{
		{
			name: "string type",
			toml: `[config]
value = "hello"`,
			want: []string{"Value", "string", `Value: "hello"`},
		},
		{
			name: "int type",
			toml: `[config]
value = 42`,
			want: []string{"Value", "int64", "Value: 42"},
		},
		{
			name: "float type",
			toml: `[config]
value = 3.14`,
			want: []string{"Value", "float64", "Value: 3.14"},
		},
		{
			name: "bool type",
			toml: `[config]
value = true`,
			want: []string{"Value", "bool", "Value: true"},
		},
		{
			name: "string array",
			toml: `[config]
values = ["a", "b", "c"]`,
			want: []string{"Values", "[]string", `[]string{"a", "b", "c"}`},
		},
		{
			name: "int array",
			toml: `[config]
values = [1, 2, 3]`,
			want: []string{"Values", "[]int64", "[]int64{1, 2, 3}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New()
			output, err := gen.Generate([]byte(tt.toml))
			require.NoError(t, err, "Generate() should not error")

			outputStr := string(output)
			for _, want := range tt.want {
				require.Contains(t, outputStr, want, "output missing expected string")
			}
		})
	}
}

func TestGenerator_NestedStructs(t *testing.T) {
	toml := `[database.pool]
max_connections = 10
min_connections = 5`

	gen := New()
	output, err := gen.Generate([]byte(toml))
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Check for nested struct types
	require.Contains(t, outputStr, "type DatabaseConfig struct", "missing parent struct")
	require.Contains(t, outputStr, "type DatabasePoolConfig struct", "missing nested struct")
	require.Contains(t, outputStr, "Pool DatabasePoolConfig", "missing field reference")
}

func TestGenerator_ArrayOfTables(t *testing.T) {
	toml := `[[servers]]
name = "web1"
port = 8080

[[servers]]
name = "web2"
port = 8081`

	gen := New()
	output, err := gen.Generate([]byte(toml))
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Check for array of tables struct
	require.Contains(t, outputStr, "type ServersItem struct", "missing array item struct")
	require.Contains(t, outputStr, "Servers = []ServersItem", "missing array variable")
	require.Contains(t, outputStr, `Name: "web1"`, "missing first item")
	require.Contains(t, outputStr, `Name: "web2"`, "missing second item")
}

func TestGenerator_DeeplyNestedStructs(t *testing.T) {
	toml := `[app.logging.rotation]
enabled = true
max_size = 100`

	gen := New()
	output, err := gen.Generate([]byte(toml))
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Check for deeply nested structs
	require.Contains(t, outputStr, "type AppConfig struct", "missing top-level struct")
	require.Contains(t, outputStr, "type AppLoggingConfig struct", "missing mid-level struct")
	require.Contains(t, outputStr, "type AppLoggingRotationConfig struct", "missing deep struct")
}
