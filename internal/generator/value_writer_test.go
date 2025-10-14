package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator_DurationTypes(t *testing.T) {
	tests := []struct {
		name string
		toml string
		want []string
	}{
		{
			name: "simple duration - seconds",
			toml: `[config]
timeout = "30s"`,
			want: []string{"Timeout", "time.Duration", "30 * time.Second", "import \"time\""},
		},
		{
			name: "simple duration - milliseconds",
			toml: `[config]
timeout = "500ms"`,
			want: []string{"Timeout", "time.Duration", "500 * time.Millisecond", "import \"time\""},
		},
		{
			name: "simple duration - minutes",
			toml: `[config]
timeout = "5m"`,
			want: []string{"Timeout", "time.Duration", "5 * time.Minute", "import \"time\""},
		},
		{
			name: "simple duration - hours",
			toml: `[config]
timeout = "2h"`,
			want: []string{"Timeout", "time.Duration", "2 * time.Hour", "import \"time\""},
		},
		{
			name: "zero duration",
			toml: `[config]
timeout = "0s"`,
			want: []string{"Timeout", "time.Duration", "Timeout: 0", "import \"time\""},
		},
		{
			name: "complex duration - hours and minutes",
			toml: `[config]
timeout = "2h30m"`,
			want: []string{"Timeout", "time.Duration", "2*time.Hour + 30*time.Minute", "import \"time\""},
		},
		{
			name: "complex duration - minutes and seconds",
			toml: `[config]
timeout = "5m30s"`,
			want: []string{"Timeout", "time.Duration", "5*time.Minute + 30*time.Second", "import \"time\""},
		},
		{
			name: "complex duration - hours, minutes and seconds",
			toml: `[config]
timeout = "1h30m45s"`,
			want: []string{"Timeout", "time.Duration", "1*time.Hour + 30*time.Minute + 45*time.Second", "import \"time\""},
		},
		{
			name: "complex duration - seconds and milliseconds",
			toml: `[config]
timeout = "3s500ms"`,
			want: []string{"Timeout", "time.Duration", "3*time.Second + 500*time.Millisecond", "import \"time\""},
		},
		{
			name: "complex duration - full decomposition",
			toml: `[config]
timeout = "1h2m3s4ms5us6ns"`,
			want: []string{"Timeout", "time.Duration", "1*time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond + 5*time.Microsecond + 6*time.Nanosecond", "import \"time\""},
		},
		{
			name: "multiple durations with different formats",
			toml: `[config]
short = "500ms"
medium = "5m"
long = "2h"
complex = "1h30m"`,
			want: []string{
				"Short", "Medium", "Long", "Complex",
				"time.Duration",
				"500 * time.Millisecond",
				"5 * time.Minute",
				"2 * time.Hour",
				"1*time.Hour + 30*time.Minute",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New()
			output, err := gen.Generate([]byte(tt.toml))
			require.NoError(t, err, "Generate() should not error")

			outputStr := string(output)
			for _, want := range tt.want {
				require.Contains(t, outputStr, want, "output missing expected string: %s", want)
			}
		})
	}
}

func TestGenerator_toGoType(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"string type", "hello", "string"},
		{"int64 type", int64(42), "int64"},
		{"int type", 42, "int64"},
		{"float64 type", 3.14, "float64"},
		{"bool type", true, "bool"},
		{"string array", []any{"a", "b"}, "[]string"},
		{"int array", []any{int64(1), int64(2)}, "[]int64"},
		{"empty array", []any{}, "[]any"},
		{"map type", map[string]any{"key": "value"}, "struct"},
		{"map array", []map[string]any{{"key": "value"}}, "[]struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			got := g.toGoType(tt.value)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_toGoType_Duration(t *testing.T) {
	g := New()
	got := g.toGoType("30s")
	require.Equal(t, "time.Duration", got)
}

func TestGenerator_toGoType_FileReference(t *testing.T) {
	g := New()
	got := g.toGoType("file:test.txt")
	require.Equal(t, "[]byte", got)
}
