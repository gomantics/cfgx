package pkgutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInferName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"current directory", "config.go", "config"},
		{"simple directory", "myapp/config.go", "myapp"},
		{"nested directory", "pkg/config/config.go", "config"},
		{"internal directory", "internal/config/config.go", "config"},
		{"lib directory", "lib/config/config.go", "config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferName(tt.path)
			require.Equal(t, tt.want, got, "InferName(%q)", tt.path)
		})
	}
}
