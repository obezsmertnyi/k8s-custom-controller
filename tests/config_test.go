package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigDefaults verifies that default configuration values are set correctly
func TestConfigDefaults(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfig()

	// Assert default values
	assert.Equal(t, "default", config.Kubernetes.Namespace)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	assert.Equal(t, float32(50), config.Kubernetes.QPS)
	assert.Equal(t, 100, config.Kubernetes.Burst)
}

// TestConfigFromEnv verifies that environment variables override defaults
func TestConfigFromEnv(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfigWithEnv()

	// Assert environment variable values
	assert.Equal(t, "test-namespace", config.Kubernetes.Namespace)
	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
}

// TestConfigFromFile verifies that config file values are loaded correctly
func TestConfigFromFile(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfigWithFile()

	// Assert file values
	assert.Equal(t, "file-namespace", config.Kubernetes.Namespace)
	assert.Equal(t, "trace", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	assert.Equal(t, float32(100), config.Kubernetes.QPS)
	assert.Equal(t, 200, config.Kubernetes.Burst)
}
