package envoverride

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApply_String(t *testing.T) {
	data := map[string]any{
		"server": map[string]any{
			"addr": ":8080",
		},
	}

	os.Setenv("CONFIG_SERVER_ADDR", ":9090")
	defer os.Unsetenv("CONFIG_SERVER_ADDR")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	serverMap := data["server"].(map[string]any)
	require.Equal(t, ":9090", serverMap["addr"])
}

func TestApply_Int(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"max_conns": int64(10),
		},
	}

	os.Setenv("CONFIG_DATABASE_MAX_CONNS", "50")
	defer os.Unsetenv("CONFIG_DATABASE_MAX_CONNS")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	dbMap := data["database"].(map[string]any)
	require.Equal(t, int64(50), dbMap["max_conns"])
}

func TestApply_Float(t *testing.T) {
	data := map[string]any{
		"cache": map[string]any{
			"ttl": float64(30.5),
		},
	}

	os.Setenv("CONFIG_CACHE_TTL", "60.75")
	defer os.Unsetenv("CONFIG_CACHE_TTL")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	cacheMap := data["cache"].(map[string]any)
	require.Equal(t, float64(60.75), cacheMap["ttl"])
}

func TestApply_Bool(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"debug": false,
		},
	}

	os.Setenv("CONFIG_APP_DEBUG", "true")
	defer os.Unsetenv("CONFIG_APP_DEBUG")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	appMap := data["app"].(map[string]any)
	require.Equal(t, true, appMap["debug"])
}

func TestApply_Array(t *testing.T) {
	data := map[string]any{
		"service": map[string]any{
			"ports": []any{int64(8080), int64(8081)},
		},
	}

	os.Setenv("CONFIG_SERVICE_PORTS", "9000,9001,9002")
	defer os.Unsetenv("CONFIG_SERVICE_PORTS")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	serviceMap := data["service"].(map[string]any)
	ports := serviceMap["ports"].([]any)

	require.Len(t, ports, 3, "expected 3 ports")

	expected := []int64{9000, 9001, 9002}
	for i, port := range ports {
		require.Equal(t, expected[i], port, "port[%d] should match", i)
	}
}

func TestApply_StringArray(t *testing.T) {
	data := map[string]any{
		"service": map[string]any{
			"origins": []any{"http://localhost"},
		},
	}

	os.Setenv("CONFIG_SERVICE_ORIGINS", "https://example.com,https://api.example.com")
	defer os.Unsetenv("CONFIG_SERVICE_ORIGINS")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	serviceMap := data["service"].(map[string]any)
	origins := serviceMap["origins"].([]any)

	require.Len(t, origins, 2, "expected 2 origins")

	expected := []string{"https://example.com", "https://api.example.com"}
	for i, origin := range origins {
		require.Equal(t, expected[i], origin, "origin[%d] should match", i)
	}
}

func TestApply_DeepNesting(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"logging": map[string]any{
				"rotation": map[string]any{
					"max_size": int64(100),
				},
			},
		},
	}

	os.Setenv("CONFIG_APP_LOGGING_ROTATION_MAX_SIZE", "500")
	defer os.Unsetenv("CONFIG_APP_LOGGING_ROTATION_MAX_SIZE")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	appMap := data["app"].(map[string]any)
	loggingMap := appMap["logging"].(map[string]any)
	rotationMap := loggingMap["rotation"].(map[string]any)

	require.Equal(t, int64(500), rotationMap["max_size"])
}

func TestApply_NoEnvVar(t *testing.T) {
	data := map[string]any{
		"server": map[string]any{
			"addr": ":8080",
		},
	}

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	// Value should remain unchanged
	serverMap := data["server"].(map[string]any)
	require.Equal(t, ":8080", serverMap["addr"], "addr should remain unchanged")
}

func TestApply_InvalidInt(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"max_conns": int64(10),
		},
	}

	os.Setenv("CONFIG_DATABASE_MAX_CONNS", "not-a-number")
	defer os.Unsetenv("CONFIG_DATABASE_MAX_CONNS")

	err := Apply(data)
	require.Error(t, err, "expected error for invalid int value")
}

func TestApply_InvalidBool(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"debug": false,
		},
	}

	os.Setenv("CONFIG_APP_DEBUG", "not-a-bool")
	defer os.Unsetenv("CONFIG_APP_DEBUG")

	err := Apply(data)
	require.Error(t, err, "expected error for invalid bool value")
}

func TestApply_InvalidFloat(t *testing.T) {
	data := map[string]any{
		"cache": map[string]any{
			"ttl": float64(30.5),
		},
	}

	os.Setenv("CONFIG_CACHE_TTL", "not-a-float")
	defer os.Unsetenv("CONFIG_CACHE_TTL")

	err := Apply(data)
	require.Error(t, err, "expected error for invalid float value")
}

func TestApply_MultipleSections(t *testing.T) {
	data := map[string]any{
		"server": map[string]any{
			"addr": ":8080",
			"port": int64(8080),
		},
		"database": map[string]any{
			"dsn":       "localhost",
			"max_conns": int64(10),
		},
	}

	os.Setenv("CONFIG_SERVER_ADDR", ":9090")
	os.Setenv("CONFIG_DATABASE_DSN", "postgres://prod-db/myapp")
	os.Setenv("CONFIG_DATABASE_MAX_CONNS", "100")
	defer os.Unsetenv("CONFIG_SERVER_ADDR")
	defer os.Unsetenv("CONFIG_DATABASE_DSN")
	defer os.Unsetenv("CONFIG_DATABASE_MAX_CONNS")

	err := Apply(data)
	require.NoError(t, err, "Apply() should not error")

	serverMap := data["server"].(map[string]any)
	require.Equal(t, ":9090", serverMap["addr"])

	dbMap := data["database"].(map[string]any)
	require.Equal(t, "postgres://prod-db/myapp", dbMap["dsn"])
	require.Equal(t, int64(100), dbMap["max_conns"])
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name        string
		envVal      string
		originalVal any
		want        any
		wantErr     bool
	}{
		{"string", "hello", "original", "hello", false},
		{"int64", "42", int64(0), int64(42), false},
		{"int", "42", int(0), int64(42), false},
		{"float64", "3.14", float64(0), float64(3.14), false},
		{"bool true", "true", false, true, false},
		{"bool false", "false", true, false, false},
		{"invalid int", "abc", int64(0), nil, true},
		{"invalid float", "abc", float64(0), nil, true},
		{"invalid bool", "abc", false, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertValue(tt.envVal, tt.originalVal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
