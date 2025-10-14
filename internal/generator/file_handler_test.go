package generator

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator_FileEmbedding(t *testing.T) {
	tests := []struct {
		name        string
		toml        string
		inputDir    string
		maxFileSize int64
		wantType    string
		wantError   bool
		checkBytes  bool
	}{
		{
			name: "simple text file",
			toml: `[config]
content = "file:files/small.txt"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantType:    "[]byte",
			wantError:   false,
			checkBytes:  true,
		},
		{
			name: "certificate file",
			toml: `[tls]
cert = "file:files/cert.txt"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantType:    "[]byte",
			wantError:   false,
			checkBytes:  true,
		},
		{
			name: "binary file",
			toml: `[data]
binary = "file:files/binary.dat"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantType:    "[]byte",
			wantError:   false,
			checkBytes:  true,
		},
		{
			name: "file not found",
			toml: `[config]
content = "file:files/nonexistent.txt"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantError:   true,
		},
		{
			name: "file exceeds size limit",
			toml: `[config]
content = "file:files/small.txt"`,
			inputDir:    "../../testdata",
			maxFileSize: 10, // Very small limit
			wantError:   true,
		},
		{
			name: "multiple files in struct",
			toml: `[files]
file1 = "file:files/small.txt"
file2 = "file:files/binary.dat"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantType:    "[]byte",
			wantError:   false,
			checkBytes:  true,
		},
		{
			name: "file in nested struct",
			toml: `[app.config]
content = "file:files/small.txt"`,
			inputDir:    "../../testdata",
			maxFileSize: 10 * 1024 * 1024,
			wantType:    "[]byte",
			wantError:   false,
			checkBytes:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(
				WithInputDir(tt.inputDir),
				WithMaxFileSize(tt.maxFileSize),
			)
			output, err := gen.Generate([]byte(tt.toml))

			if tt.wantError {
				require.Error(t, err, "Generate() should error")
				return
			}

			require.NoError(t, err, "Generate() should not error")
			outputStr := string(output)

			if tt.wantType != "" {
				require.Contains(t, outputStr, tt.wantType, "output missing type")
			}

			if tt.checkBytes {
				// Verify byte array format
				require.Contains(t, outputStr, "[]byte{", "output missing byte array")
				require.Contains(t, outputStr, "0x", "output missing hex format")
			}
		})
	}
}

func TestGenerator_FileEmbeddingByteFormat(t *testing.T) {
	// Test that byte arrays are formatted correctly
	toml := `[config]
content = "file:files/binary.dat"`

	gen := New(
		WithInputDir("../../testdata"),
		WithMaxFileSize(10*1024*1024),
	)
	output, err := gen.Generate([]byte(toml))
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Check for proper hex format
	require.Contains(t, outputStr, "0x00", "should contain first byte (0x00)")
	require.Contains(t, outputStr, "0xff", "should contain byte 0xff")
	require.Contains(t, outputStr, "0x0f", "should contain byte 0x0f")

	// Verify proper formatting (12 bytes per line)
	require.Contains(t, outputStr, "[]byte{", "should have byte array opening")
	require.Contains(t, outputStr, "Content []byte", "should have []byte field type")

	// Read the actual file to verify byte count
	expectedContent, err := os.ReadFile("../../testdata/files/binary.dat")
	require.NoError(t, err, "should read test file")

	// Count hex patterns in output - should match file size
	hexCount := strings.Count(outputStr, "0x")
	require.Equal(t, len(expectedContent), hexCount, "should have correct number of bytes")
}

func TestGenerator_FileEmbeddingInArrayOfTables(t *testing.T) {
	toml := `[[endpoints]]
path = "/api/v1"
cert = "file:files/small.txt"

[[endpoints]]
path = "/api/v2"
cert = "file:files/binary.dat"`

	gen := New(
		WithInputDir("../../testdata"),
		WithMaxFileSize(10*1024*1024),
	)
	output, err := gen.Generate([]byte(toml))
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Verify structure
	require.Contains(t, outputStr, "type EndpointsItem struct", "should have struct")
	require.Contains(t, outputStr, "Cert []byte", "should have []byte field")
	require.Contains(t, outputStr, "[]EndpointsItem{", "should have array")

	// Verify both files are embedded
	require.Contains(t, outputStr, "[]byte{", "should have byte arrays")
}

func TestGenerator_FileSizeLimit(t *testing.T) {
	// Test the file size limit enforcement
	tests := []struct {
		name     string
		fileSize int64
		toml     string
		wantErr  bool
	}{
		{
			name:     "within 1KB limit",
			fileSize: 1024,
			toml: `[config]
content = "file:files/small.txt"`,
			wantErr: false,
		},
		{
			name:     "exceeds 10 byte limit",
			fileSize: 10,
			toml: `[config]
content = "file:files/small.txt"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(
				WithInputDir("../../testdata"),
				WithMaxFileSize(tt.fileSize),
			)
			_, err := gen.Generate([]byte(tt.toml))

			if tt.wantErr {
				require.Error(t, err, "should error due to size limit")
				require.Contains(t, err.Error(), "exceeds max size", "error should mention size limit")
			} else {
				require.NoError(t, err, "should not error")
			}
		})
	}
}
