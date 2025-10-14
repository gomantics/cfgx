package cfgx

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/gomantics/cfgx/internal/envoverride"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	data, err := os.ReadFile("testdata/test.toml")
	require.NoError(t, err, "failed to read test file")

	output, err := Generate(data, "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

	// Write to temp file and try to compile it
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.go")

	err = os.WriteFile(configFile, output, 0644)
	require.NoError(t, err, "failed to write output file")

	// Try to compile the generated code
	cmd := exec.Command("go", "build", configFile)
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "generated code does not compile: %s", output)
}

func TestGenerate_WithEnvOverrides(t *testing.T) {
	tomlData := []byte(`
[server]
addr = ":8080"
timeout = 30

[database]
dsn = "localhost"
max_conns = 10
`)

	os.Setenv("CONFIG_SERVER_ADDR", ":9090")
	os.Setenv("CONFIG_DATABASE_MAX_CONNS", "100")
	defer os.Unsetenv("CONFIG_SERVER_ADDR")
	defer os.Unsetenv("CONFIG_DATABASE_MAX_CONNS")

	var data map[string]any
	err := toml.Unmarshal(tomlData, &data)
	require.NoError(t, err)

	err = envoverride.Apply(data)
	require.NoError(t, err, "Apply() should not error")

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	err = enc.Encode(data)
	require.NoError(t, err)

	output, err := Generate(buf.Bytes(), "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	require.Contains(t, outputStr, `":9090"`, "environment override for server.addr not applied")
	require.Contains(t, outputStr, "100", "environment override for database.max_conns not applied")

	require.NotContains(t, outputStr, `":8080"`, "original server.addr should have been overridden")
}

func TestGenerateFromFile(t *testing.T) {
	// Create a temporary TOML file
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "config.toml")
	outputFile := filepath.Join(tmpDir, "config.go")

	tomlData := []byte(`
[server]
addr = ":8080"
read_timeout = 30
write_timeout = 30
shutdown_timeout = 10

[database]
dsn = "postgres://localhost:5432/myapp"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300

[redis]
addr = "localhost:6379"
password = ""
db = 0
pool_size = 10

[logging]
level = "info"
format = "json"

[features]
auth_enabled = true
rate_limiting = true
metrics_enabled = true
`)

	err := os.WriteFile(inputFile, tomlData, 0644)
	require.NoError(t, err, "failed to write input file")

	opts := &GenerateOptions{
		InputFile:   inputFile,
		OutputFile:  outputFile,
		PackageName: "config",
		EnableEnv:   true,
	}

	err = GenerateFromFile(opts)
	require.NoError(t, err, "GenerateFromFile() should not error")

	// Verify the file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err, "output file was not created")

	// Read the generated code
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	// Verify the generated code compiles
	cmd := exec.Command("go", "build", outputFile)
	cmd.Dir = tmpDir
	cmdOutput, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated code does not compile: %s", cmdOutput)

	// Verify expected structures are present
	outputStr := string(output)
	expectedStructs := []string{
		"type DatabaseConfig struct",
		"type FeaturesConfig struct",
		"type LoggingConfig struct",
		"type RedisConfig struct",
		"type ServerConfig struct",
	}

	for _, expected := range expectedStructs {
		require.Contains(t, outputStr, expected, "expected struct definition not found: %s", expected)
	}

	require.Contains(t, outputStr, "var (", "expected var block")

	expectedVars := []string{
		"Database = DatabaseConfig",
		"Features = FeaturesConfig",
		"Logging = LoggingConfig",
		"Redis = RedisConfig",
		"Server = ServerConfig",
	}

	for _, expected := range expectedVars {
		require.Contains(t, outputStr, expected, "expected variable declaration not found: %s", expected)
	}
}

func TestGenerateFromFile_WithFileEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "config.toml")
	outputFile := filepath.Join(tmpDir, "generated/config.go")

	filesDir := filepath.Join(tmpDir, "data")
	err := os.MkdirAll(filesDir, 0755)
	require.NoError(t, err)

	testContent := []byte("Hello from embedded file!\nLine 2")
	testFile := filepath.Join(filesDir, "test.txt")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	tomlData := []byte(`
[app]
name = "test"
content = "file:data/test.txt"

[server]
addr = ":8080"
`)

	err = os.WriteFile(inputFile, tomlData, 0644)
	require.NoError(t, err)

	opts := &GenerateOptions{
		InputFile:   inputFile,
		OutputFile:  outputFile,
		PackageName: "config",
		EnableEnv:   false,
		MaxFileSize: 10 * 1024 * 1024,
	}

	err = GenerateFromFile(opts)
	require.NoError(t, err, "GenerateFromFile() should not error")

	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	require.Contains(t, outputStr, "Content []byte", "should have []byte field")
	require.Contains(t, outputStr, "0x48", "should contain 'H' (0x48)")
	require.Contains(t, outputStr, "0x65", "should contain 'e' (0x65)")
	require.Contains(t, outputStr, "[]byte{", "should have byte array literal")

	cmd := exec.Command("go", "build", outputFile)
	cmd.Dir = tmpDir
	cmdOutput, err := cmd.CombinedOutput()
	require.NoError(t, err, "generated code does not compile: %s", cmdOutput)
}

func TestGenerateFromFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "config.toml")
	outputFile := filepath.Join(tmpDir, "config.go")

	// Create TOML with reference to non-existent file
	tomlData := []byte(`
[app]
content = "file:nonexistent.txt"
`)

	err := os.WriteFile(inputFile, tomlData, 0644)
	require.NoError(t, err)

	opts := &GenerateOptions{
		InputFile:  inputFile,
		OutputFile: outputFile,
	}

	err = GenerateFromFile(opts)
	require.Error(t, err, "should error on non-existent file")
	require.Contains(t, err.Error(), "file not found", "error should mention file not found")
}

func TestGenerateFromFile_FileSizeExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "config.toml")
	outputFile := filepath.Join(tmpDir, "config.go")

	// Create a test file
	largeFile := filepath.Join(tmpDir, "large.txt")
	err := os.WriteFile(largeFile, []byte("This file is too large for the limit"), 0644)
	require.NoError(t, err)

	tomlData := []byte(`
[app]
content = "file:large.txt"
`)

	err = os.WriteFile(inputFile, tomlData, 0644)
	require.NoError(t, err)

	opts := &GenerateOptions{
		InputFile:   inputFile,
		OutputFile:  outputFile,
		MaxFileSize: 10, // Very small limit
	}

	err = GenerateFromFile(opts)
	require.Error(t, err, "should error on file size exceeded")
	require.Contains(t, err.Error(), "exceeds max size", "error should mention size limit")
}
