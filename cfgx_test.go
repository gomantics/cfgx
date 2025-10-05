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

func TestGenerate_Simple(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.toml")
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

func TestGenerate_Nested(t *testing.T) {
	data, err := os.ReadFile("testdata/nested.toml")
	require.NoError(t, err, "failed to read test file")

	output, err := Generate(data, "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

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

func TestGenerate_Arrays(t *testing.T) {
	data, err := os.ReadFile("testdata/arrays.toml")
	require.NoError(t, err, "failed to read test file")

	output, err := Generate(data, "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

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

func TestGenerate_Complex(t *testing.T) {
	data, err := os.ReadFile("testdata/complex.toml")
	require.NoError(t, err, "failed to read test file")

	output, err := Generate(data, "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

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

	// Set environment variables
	os.Setenv("CONFIG_SERVER_ADDR", ":9090")
	os.Setenv("CONFIG_DATABASE_MAX_CONNS", "100")
	defer os.Unsetenv("CONFIG_SERVER_ADDR")
	defer os.Unsetenv("CONFIG_DATABASE_MAX_CONNS")

	// First parse and apply env overrides manually to simulate what the CLI does
	var data map[string]any
	err := toml.Unmarshal(tomlData, &data)
	require.NoError(t, err)

	err = envoverride.Apply(data)
	require.NoError(t, err, "Apply() should not error")

	// Re-encode to TOML
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	err = enc.Encode(data)
	require.NoError(t, err)

	output, err := Generate(buf.Bytes(), "testconfig", true)
	require.NoError(t, err, "Generate() should not error")

	outputStr := string(output)

	// Check that overridden values are in the generated code
	require.Contains(t, outputStr, `":9090"`, "environment override for server.addr not applied")
	require.Contains(t, outputStr, "100", "environment override for database.max_conns not applied")

	// Original values should NOT be present
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

	expectedVars := []string{
		"var Database = DatabaseConfig",
		"var Features = FeaturesConfig",
		"var Logging = LoggingConfig",
		"var Redis = RedisConfig",
		"var Server = ServerConfig",
	}

	for _, expected := range expectedVars {
		require.Contains(t, outputStr, expected, "expected variable declaration not found: %s", expected)
	}
}
