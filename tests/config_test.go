package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestConfigDefaults verifies that default configuration values are set correctly
func TestConfigDefaults(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfig()

	// Assert Kubernetes default values
	assert.Equal(t, "", config.Kubernetes.Kubeconfig)
	assert.Equal(t, "", config.Kubernetes.Context)
	assert.Equal(t, "default", config.Kubernetes.Namespace)
	assert.Equal(t, 30*time.Second, config.Kubernetes.Timeout)
	assert.Equal(t, float32(50), config.Kubernetes.QPS)
	assert.Equal(t, 100, config.Kubernetes.Burst)
	assert.Equal(t, true, config.Kubernetes.InCluster)
	assert.Equal(t, false, config.Kubernetes.DisableInformer)
	assert.Equal(t, false, config.Kubernetes.DisableAPI)
	
	// Assert Logging default values
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	
	// Assert Informer default values
	assert.Equal(t, true, config.Informer.Enabled)
	assert.Equal(t, "", config.Informer.Namespace)
	assert.Equal(t, 10*time.Minute, config.Informer.ResyncPeriod)
	assert.Equal(t, "", config.Informer.LabelSelector)
	assert.Equal(t, "", config.Informer.FieldSelector)
	assert.Equal(t, false, config.Informer.Logging.EnableEventLogging)
	assert.Equal(t, "info", config.Informer.Logging.LogLevel)
	assert.Equal(t, 2, config.Informer.Workers.Count)
	
	// Assert APIServer default values
	assert.Equal(t, true, config.APIServer.Enabled)
	assert.Equal(t, "0.0.0.0", config.APIServer.Host)
	assert.Equal(t, 8080, config.APIServer.Port)
	assert.Equal(t, true, config.APIServer.EnableSwagger)
	assert.Equal(t, 100, config.APIServer.Security.RateLimitRequestsPerSecond)
	assert.Equal(t, 10, config.APIServer.Security.MaxConnsPerIP)
	assert.Equal(t, 30, config.APIServer.Security.ReadTimeoutSeconds)
	assert.Equal(t, 30, config.APIServer.Security.WriteTimeoutSeconds)
	assert.Equal(t, 60, config.APIServer.Security.IdleTimeoutSeconds)
	assert.Equal(t, false, config.APIServer.Security.DisableKeepalive)
	assert.Equal(t, true, config.APIServer.SwaggerUI.Enabled)
	assert.Equal(t, false, config.APIServer.SwaggerUI.CORSEnabled)
	assert.Equal(t, "*", config.APIServer.SwaggerUI.CORSAllowOrigin)
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", config.APIServer.SwaggerUI.CORSAllowMethods)
	assert.Equal(t, "Content-Type, Authorization", config.APIServer.SwaggerUI.CORSAllowHeaders)
	assert.Equal(t, 86400, config.APIServer.SwaggerUI.CORSMaxAge)
	assert.Equal(t, true, config.APIServer.SwaggerUI.UseStrictCSP)
	
	// Assert ControllerRuntime default values
	assert.Equal(t, false, config.ControllerRuntime.LeaderElection.Enabled)
	assert.Equal(t, "k8s-controller", config.ControllerRuntime.LeaderElection.ID)
	assert.Equal(t, "kube-system", config.ControllerRuntime.LeaderElection.Namespace)
	assert.Equal(t, ":8081", config.ControllerRuntime.Metrics.BindAddress)
}

// TestConfigFromEnv verifies that environment variables override defaults
func TestConfigFromEnv(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfigWithEnv()

	// Assert Kubernetes environment values
	assert.Equal(t, "/home/user/.kube/custom-config", config.Kubernetes.Kubeconfig)
	assert.Equal(t, "test-context", config.Kubernetes.Context)
	assert.Equal(t, "test-namespace", config.Kubernetes.Namespace)
	assert.Equal(t, 45*time.Second, config.Kubernetes.Timeout)
	assert.Equal(t, float32(75), config.Kubernetes.QPS)
	assert.Equal(t, 150, config.Kubernetes.Burst)
	assert.Equal(t, false, config.Kubernetes.InCluster)
	
	// Assert Logging environment values
	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	
	// Assert Informer environment values
	assert.Equal(t, true, config.Informer.Enabled)
	assert.Equal(t, "env-namespace", config.Informer.Namespace)
	assert.Equal(t, 5*time.Minute, config.Informer.ResyncPeriod)
	assert.Equal(t, "env=test", config.Informer.LabelSelector)
	assert.Equal(t, true, config.Informer.Logging.EnableEventLogging)
	
	// Assert API server environment values
	assert.Equal(t, "127.0.0.1", config.APIServer.Host)
	assert.Equal(t, 9090, config.APIServer.Port)
	assert.Equal(t, 20, config.APIServer.Security.MaxConnsPerIP)
}

// TestConfigFromFile verifies that config file values are loaded correctly
func TestConfigFromFile(t *testing.T) {
	// Use mock instead of real Viper
	config := MockConfigWithFile()

	// Assert Kubernetes file values
	assert.Equal(t, "/path/to/kubeconfig", config.Kubernetes.Kubeconfig)
	assert.Equal(t, "file-context", config.Kubernetes.Context)
	assert.Equal(t, "file-namespace", config.Kubernetes.Namespace)
	assert.Equal(t, 60*time.Second, config.Kubernetes.Timeout)
	assert.Equal(t, float32(100), config.Kubernetes.QPS)
	assert.Equal(t, 200, config.Kubernetes.Burst)
	assert.Equal(t, false, config.Kubernetes.InCluster)
	assert.Equal(t, true, config.Kubernetes.DisableInformer)
	
	// Assert Logging file values
	assert.Equal(t, "trace", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	
	// Assert Informer file values
	assert.Equal(t, false, config.Informer.Enabled)
	assert.Equal(t, "file-informer-namespace", config.Informer.Namespace)
	assert.Equal(t, 30*time.Minute, config.Informer.ResyncPeriod)
	assert.Equal(t, "env=prod", config.Informer.LabelSelector)
	assert.Equal(t, "status.phase=Running", config.Informer.FieldSelector)
	assert.Equal(t, "debug", config.Informer.Logging.LogLevel)
	assert.Equal(t, 4, config.Informer.Workers.Count)
	
	// Assert API Server file values
	assert.Equal(t, true, config.APIServer.Enabled)
	assert.Equal(t, "localhost", config.APIServer.Host)
	assert.Equal(t, 8000, config.APIServer.Port)
	assert.Equal(t, true, config.APIServer.EnableSwagger)
	assert.Equal(t, 50, config.APIServer.Security.RateLimitRequestsPerSecond)
	assert.Equal(t, 5, config.APIServer.Security.MaxConnsPerIP)
	assert.Equal(t, 15, config.APIServer.Security.ReadTimeoutSeconds)
	assert.Equal(t, 15, config.APIServer.Security.WriteTimeoutSeconds)
	assert.Equal(t, 30, config.APIServer.Security.IdleTimeoutSeconds)
	assert.Equal(t, true, config.APIServer.Security.DisableKeepalive)
	assert.Equal(t, true, config.APIServer.SwaggerUI.CORSEnabled)
	assert.Equal(t, "https://example.com", config.APIServer.SwaggerUI.CORSAllowOrigin)
	
	// Assert Controller Runtime file values
	assert.Equal(t, true, config.ControllerRuntime.LeaderElection.Enabled)
	assert.Equal(t, "file-controller", config.ControllerRuntime.LeaderElection.ID)
	assert.Equal(t, ":9100", config.ControllerRuntime.Metrics.BindAddress)
}
