package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator_needsTimeImport(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
		want bool
	}{
		{
			name: "simple duration string",
			data: map[string]any{"timeout": "30s"},
			want: true,
		},
		{
			name: "nested duration string",
			data: map[string]any{
				"config": map[string]any{
					"timeout": "5m",
				},
			},
			want: true,
		},
		{
			name: "no duration",
			data: map[string]any{"value": "hello"},
			want: false,
		},
		{
			name: "duration in array",
			data: map[string]any{"timeouts": []any{"30s", "1m"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			got := g.needsTimeImport(tt.data)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_isDurationString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"valid seconds", "30s", true},
		{"valid minutes", "5m", true},
		{"valid hours", "2h", true},
		{"valid complex", "2h30m", true},
		{"invalid string", "hello", false},
		{"invalid format", "30", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			got := g.isDurationString(tt.s)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_validateFileReferences(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]any
		inputDir  string
		wantError bool
	}{
		{
			name:      "valid file reference",
			data:      map[string]any{"content": "file:files/small.txt"},
			inputDir:  "../../testdata",
			wantError: false,
		},
		{
			name:      "missing file",
			data:      map[string]any{"content": "file:files/nonexistent.txt"},
			inputDir:  "../../testdata",
			wantError: true,
		},
		{
			name:      "no file references",
			data:      map[string]any{"value": "hello"},
			inputDir:  "",
			wantError: false,
		},
		{
			name: "nested file reference",
			data: map[string]any{
				"config": map[string]any{
					"cert": "file:files/cert.txt",
				},
			},
			inputDir:  "../../testdata",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(WithInputDir(tt.inputDir))
			err := g.validateFileReferences(tt.data)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
