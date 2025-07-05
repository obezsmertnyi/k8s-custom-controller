package tests

import (
	"github.com/obezsmertnyi/k8s-custom-controller/cmd"
	"time"
)

// MockConfig creates a mock configuration for testing
func MockConfig() *cmd.Config {
	config := &cmd.Config{}
	
	// Set default values
	config.Kubernetes.Timeout = 30 * time.Second
	config.Kubernetes.QPS = 50
	config.Kubernetes.Burst = 100
	config.Kubernetes.Namespace = "default"
	config.Logging.Level = "info"
	config.Logging.Format = "text"
	
	return config
}

// MockConfigWithEnv creates a mock configuration with environment variable values
func MockConfigWithEnv() *cmd.Config {
	config := MockConfig()
	config.Kubernetes.Namespace = "test-namespace"
	config.Logging.Level = "debug"
	config.Logging.Format = "json"
	return config
}

// MockConfigWithFile creates a mock configuration with file values
func MockConfigWithFile() *cmd.Config {
	config := MockConfig()
	config.Kubernetes.Namespace = "file-namespace"
	config.Kubernetes.QPS = 100
	config.Kubernetes.Burst = 200
	config.Logging.Level = "trace"
	config.Logging.Format = "json"
	return config
}
