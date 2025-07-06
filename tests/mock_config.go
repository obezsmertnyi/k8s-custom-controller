package tests

import (
	"github.com/obezsmertnyi/k8s-custom-controller/cmd"
	"time"
)

// MockConfig creates a mock configuration for testing with complete default values
func MockConfig() *cmd.Config {
	config := &cmd.Config{}
	
	// Set kubernetes default values
	config.Kubernetes.Kubeconfig = ""
	config.Kubernetes.Context = ""
	config.Kubernetes.Namespace = "default"
	config.Kubernetes.Timeout = 30 * time.Second
	config.Kubernetes.QPS = 50
	config.Kubernetes.Burst = 100
	config.Kubernetes.InCluster = true
	config.Kubernetes.DisableInformer = false
	config.Kubernetes.DisableAPI = false
	
	// Set logging default values
	config.Logging.Level = "info"
	config.Logging.Format = "text"
	
	// Set informer default values
	config.Informer.Enabled = true
	config.Informer.Namespace = ""
	config.Informer.ResyncPeriod = 10 * time.Minute
	config.Informer.LabelSelector = ""
	config.Informer.FieldSelector = ""
	config.Informer.Logging.EnableEventLogging = false
	config.Informer.Logging.LogLevel = "info"
	config.Informer.Workers.Count = 2
	
	// Set API server default values
	config.APIServer.Enabled = true
	config.APIServer.Host = "0.0.0.0"
	config.APIServer.Port = 8080
	config.APIServer.EnableSwagger = true
	config.APIServer.Security.RateLimitRequestsPerSecond = 100
	config.APIServer.Security.MaxConnsPerIP = 10
	config.APIServer.Security.ReadTimeoutSeconds = 30
	config.APIServer.Security.WriteTimeoutSeconds = 30
	config.APIServer.Security.IdleTimeoutSeconds = 60
	config.APIServer.Security.DisableKeepalive = false
	config.APIServer.SwaggerUI.Enabled = true
	config.APIServer.SwaggerUI.CORSEnabled = false
	config.APIServer.SwaggerUI.CORSAllowOrigin = "*"
	config.APIServer.SwaggerUI.CORSAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	config.APIServer.SwaggerUI.CORSAllowHeaders = "Content-Type, Authorization"
	config.APIServer.SwaggerUI.CORSMaxAge = 86400
	config.APIServer.SwaggerUI.UseStrictCSP = true
	
	// Set controller runtime default values
	config.ControllerRuntime.LeaderElection.Enabled = false
	config.ControllerRuntime.LeaderElection.ID = "k8s-controller"
	config.ControllerRuntime.LeaderElection.Namespace = "kube-system"
	config.ControllerRuntime.Metrics.BindAddress = ":8081"
	
	return config
}

// MockConfigWithEnv creates a mock configuration with environment variable values
func MockConfigWithEnv() *cmd.Config {
	config := MockConfig()
	// Override kubernetes settings
	config.Kubernetes.Kubeconfig = "/home/user/.kube/custom-config"
	config.Kubernetes.Context = "test-context"
	config.Kubernetes.Namespace = "test-namespace"
	config.Kubernetes.Timeout = 45 * time.Second
	config.Kubernetes.QPS = 75
	config.Kubernetes.Burst = 150
	config.Kubernetes.InCluster = false
	
	// Override logging settings
	config.Logging.Level = "debug"
	config.Logging.Format = "json"
	
	// Override informer settings
	config.Informer.Enabled = true
	config.Informer.Namespace = "env-namespace"
	config.Informer.ResyncPeriod = 5 * time.Minute
	config.Informer.LabelSelector = "env=test"
	config.Informer.Logging.EnableEventLogging = true
	
	// Override API server settings
	config.APIServer.Host = "127.0.0.1"
	config.APIServer.Port = 9090
	config.APIServer.Security.MaxConnsPerIP = 20
	
	return config
}

// MockConfigWithFile creates a mock configuration with file values
func MockConfigWithFile() *cmd.Config {
	config := MockConfig()
	
	// Override kubernetes settings from file
	config.Kubernetes.Kubeconfig = "/path/to/kubeconfig"
	config.Kubernetes.Context = "file-context"
	config.Kubernetes.Namespace = "file-namespace"
	config.Kubernetes.Timeout = 60 * time.Second
	config.Kubernetes.QPS = 100
	config.Kubernetes.Burst = 200
	config.Kubernetes.InCluster = false
	config.Kubernetes.DisableInformer = true
	
	// Override logging settings from file
	config.Logging.Level = "trace"
	config.Logging.Format = "json"
	
	// Override informer settings from file
	config.Informer.Enabled = false
	config.Informer.Namespace = "file-informer-namespace"
	config.Informer.ResyncPeriod = 30 * time.Minute
	config.Informer.LabelSelector = "env=prod"
	config.Informer.FieldSelector = "status.phase=Running"
	config.Informer.Logging.LogLevel = "debug"
	config.Informer.Workers.Count = 4
	
	// Override API server settings from file
	config.APIServer.Enabled = true
	config.APIServer.Host = "localhost"
	config.APIServer.Port = 8000
	config.APIServer.EnableSwagger = true
	config.APIServer.Security.RateLimitRequestsPerSecond = 50
	config.APIServer.Security.MaxConnsPerIP = 5
	config.APIServer.Security.ReadTimeoutSeconds = 15
	config.APIServer.Security.WriteTimeoutSeconds = 15
	config.APIServer.Security.IdleTimeoutSeconds = 30
	config.APIServer.Security.DisableKeepalive = true
	config.APIServer.SwaggerUI.CORSEnabled = true
	config.APIServer.SwaggerUI.CORSAllowOrigin = "https://example.com"
	
	// Override controller runtime settings from file
	config.ControllerRuntime.LeaderElection.Enabled = true
	config.ControllerRuntime.LeaderElection.ID = "file-controller"
	config.ControllerRuntime.Metrics.BindAddress = ":9100"
	
	return config
}
