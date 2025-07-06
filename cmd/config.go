package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/obezsmertnyi/k8s-custom-controller/pkg/informer"
)

// Config structure for storing application configuration
type Config struct {
	// Kubernetes settings
	Kubernetes struct {
		Kubeconfig      string        `mapstructure:"kubeconfig"`
		Context         string        `mapstructure:"context"`
		Namespace       string        `mapstructure:"namespace"`
		Timeout         time.Duration `mapstructure:"timeout"`
		QPS             float32       `mapstructure:"qps"`
		Burst           int           `mapstructure:"burst"`
		InCluster       bool          `mapstructure:"in_cluster"`
		// Deprecated: Use Informer.Enabled and APIServer.Enabled instead
		DisableInformer bool          `mapstructure:"disable_informer"`
		// Deprecated: Use Informer.Enabled and APIServer.Enabled instead
		DisableAPI      bool          `mapstructure:"disable_api"`
	} `mapstructure:"kubernetes"`

	// Logging settings
	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`

	// Informer settings
	Informer struct {
		Enabled       bool          `mapstructure:"enabled"`
		Namespace     string        `mapstructure:"namespace"`
		ResyncPeriod  time.Duration `mapstructure:"resync_period"`
		LabelSelector string        `mapstructure:"label_selector"`
		FieldSelector string        `mapstructure:"field_selector"`

		// Nested informer configurations
		Logging struct {
			EnableEventLogging bool   `mapstructure:"enable_event_logging"`
			LogLevel           string `mapstructure:"log_level"`
		} `mapstructure:"logging"`
		
		Workers struct {
			Count int `mapstructure:"count"`
		} `mapstructure:"workers"`
	} `mapstructure:"informer"`
	
	// API Server settings
	APIServer struct {
		Enabled       bool   `mapstructure:"enabled"`
		Host          string `mapstructure:"host"`
		Port          int    `mapstructure:"port"`
		EnableSwagger bool   `mapstructure:"enable_swagger"`

		// Security settings
		Security struct {
			RateLimitRequestsPerSecond int `mapstructure:"rate_limit_requests_per_second"`
			MaxConnsPerIP             int `mapstructure:"max_connections_per_ip"`
			ReadTimeoutSeconds        int `mapstructure:"read_timeout_seconds"`
			WriteTimeoutSeconds       int `mapstructure:"write_timeout_seconds"`
			IdleTimeoutSeconds        int `mapstructure:"idle_timeout_seconds"`
			DisableKeepalive          bool `mapstructure:"disable_keepalive"`
		} `mapstructure:"security"`
		
		// Swagger UI specific settings
		SwaggerUI struct {
			Enabled      bool   `mapstructure:"enabled"` 
			CORSEnabled     bool   `mapstructure:"cors_enabled"`
			CORSAllowOrigin string `mapstructure:"cors_allow_origin"`
			CORSAllowMethods string `mapstructure:"cors_allow_methods"`
			CORSAllowHeaders string `mapstructure:"cors_allow_headers"`
			CORSMaxAge       int    `mapstructure:"cors_max_age"`
			UseStrictCSP     bool   `mapstructure:"use_strict_csp"`
		} `mapstructure:"swagger_ui"`
	} `mapstructure:"api_server"`

	// Controller-Runtime settings
	ControllerRuntime struct {
		// Leader Election settings
		LeaderElection struct {
			Enabled   bool   `mapstructure:"enabled"`
			ID        string `mapstructure:"id"`
			Namespace string `mapstructure:"namespace"`
		} `mapstructure:"leader_election"`
		// Metrics server settings
		Metrics struct {
			BindAddress string `mapstructure:"bind_address"`
		} `mapstructure:"metrics"`
	} `mapstructure:"controller_runtime"`
}

// homeDir returns the path to the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}

// LoadConfig loads configuration from file and environment variables
// Exported function for use in tests and other packages
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Set default values for Kubernetes
	config.Kubernetes.Timeout = 30 * time.Second
	config.Kubernetes.QPS = 50
	config.Kubernetes.Burst = 100
	config.Kubernetes.Namespace = "default"
	config.Kubernetes.InCluster = false // Use kubeconfig by default
	config.Kubernetes.DisableInformer = false // Enable informer by default
	config.Kubernetes.DisableAPI = false // Enable API by default

	// Default values for logging
	config.Logging.Level = "info"
	config.Logging.Format = "text"

	// Default values for informer
	config.Informer.Enabled = true // Enable informer by default
	config.Informer.Namespace = "default"
	config.Informer.ResyncPeriod = 10 * time.Minute
	config.Informer.LabelSelector = ""
	config.Informer.FieldSelector = ""
	config.Informer.Logging.EnableEventLogging = false
	config.Informer.Logging.LogLevel = "info"
	config.Informer.Workers.Count = 2

	// Default values for API Server
	config.APIServer.Enabled = true // Enable API server by default
	config.APIServer.Host = "0.0.0.0"
	config.APIServer.Port = 8080
	config.APIServer.EnableSwagger = true // Enable Swagger UI by default
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

	// Default values for Controller Runtime
	config.ControllerRuntime.LeaderElection.Enabled = false
	config.ControllerRuntime.LeaderElection.ID = "k8s-controller"
	config.ControllerRuntime.LeaderElection.Namespace = "kube-system"
	config.ControllerRuntime.Metrics.BindAddress = ":8081"

	// Set default kubeconfig path
	if home := homeDir(); home != "" {
		config.Kubernetes.Kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Configure Viper
	// Automatically use environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("KCUSTOM") // KCUSTOM_KUBERNETES_NAMESPACE
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if cfgFile != "" {
		// Use configuration file specified via --config flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for default configuration file
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Search for configuration file in standard locations
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.k8s-custom-controller")
		viper.AddConfigPath("/etc/k8s-custom-controller")
	}

	// Explicitly bind environment variables to configuration keys
	
	// Kubernetes configuration
	viper.BindEnv("kubernetes.namespace", "KUBERNETES_NAMESPACE")
	viper.BindEnv("kubernetes.kubeconfig", "KUBERNETES_KUBECONFIG")
	viper.BindEnv("kubernetes.context", "KUBERNETES_CONTEXT")
	viper.BindEnv("kubernetes.qps", "KUBERNETES_QPS")
	viper.BindEnv("kubernetes.burst", "KUBERNETES_BURST")
	viper.BindEnv("kubernetes.timeout", "KUBERNETES_TIMEOUT")
	viper.BindEnv("kubernetes.in_cluster", "KUBERNETES_IN_CLUSTER")
	viper.BindEnv("kubernetes.disable_informer", "KUBERNETES_DISABLE_INFORMER")
	viper.BindEnv("kubernetes.disable_api", "KUBERNETES_DISABLE_API")
	
	// Logging configuration
	viper.BindEnv("logging.level", "LOGGING_LEVEL")
	viper.BindEnv("logging.format", "LOGGING_FORMAT")

	// Informer configuration
	viper.BindEnv("informer.enabled", "INFORMER_ENABLED")
	viper.BindEnv("informer.namespace", "INFORMER_NAMESPACE")
	viper.BindEnv("informer.resync_period", "INFORMER_RESYNC_PERIOD")
	viper.BindEnv("informer.label_selector", "INFORMER_LABEL_SELECTOR")
	viper.BindEnv("informer.field_selector", "INFORMER_FIELD_SELECTOR")
	viper.BindEnv("informer.logging.enable_event_logging", "INFORMER_LOGGING_ENABLE_EVENT_LOGGING")
	viper.BindEnv("informer.logging.log_level", "INFORMER_LOGGING_LOG_LEVEL")
	viper.BindEnv("informer.workers.count", "INFORMER_WORKERS_COUNT")
	
	// API Server configuration
	viper.BindEnv("api_server.enabled", "APISERVER_ENABLED")
	viper.BindEnv("api_server.host", "APISERVER_HOST")
	viper.BindEnv("api_server.port", "APISERVER_PORT")
	viper.BindEnv("api_server.enable_swagger", "APISERVER_ENABLE_SWAGGER")
	viper.BindEnv("api_server.security.rate_limit_requests_per_second", "APISERVER_RATE_LIMIT")
	viper.BindEnv("api_server.security.max_connections_per_ip", "APISERVER_MAX_CONNS_PER_IP")
	viper.BindEnv("api_server.security.read_timeout_seconds", "APISERVER_READ_TIMEOUT")
	viper.BindEnv("api_server.security.write_timeout_seconds", "APISERVER_WRITE_TIMEOUT")
	viper.BindEnv("api_server.security.idle_timeout_seconds", "APISERVER_IDLE_TIMEOUT")

	// Controller Runtime configuration
	viper.BindEnv("controller_runtime.leader_election.enabled", "CONTROLLER_LEADER_ELECTION_ENABLED")
	viper.BindEnv("controller_runtime.leader_election.id", "CONTROLLER_LEADER_ELECTION_ID")
	viper.BindEnv("controller_runtime.leader_election.namespace", "CONTROLLER_LEADER_ELECTION_NAMESPACE")
	viper.BindEnv("controller_runtime.metrics.bind_address", "CONTROLLER_METRICS_BIND_ADDRESS")

	// Attempt to read configuration file
	err := viper.ReadInConfig()
	if err != nil {
		if cfgFile != "" {
			// If specific config file was requested but not found, it's an error
			log.Error().Err(err).Str("config_file", cfgFile).Msg("Error reading specified config file")
			return nil, err
		} else {
			// For default config search path, it's just a warning
			log.Debug().Err(err).Msg("No default config file found")
			log.Warn().Msg("Using defaults and environment variables")
			// Continue with defaults and env vars
		}
	} else {
		log.Info().Str("config", viper.ConfigFileUsed()).Msg("Using config file:")
	}

	// Bind environment variables to configuration
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %v", err)
	}

	// Override values from command line flags
	if logLevel != "" {
		config.Logging.Level = logLevel
	}
	
	// Override port from command line flags
	if serverPort != 0 {
		config.APIServer.Port = serverPort
	}
	
	// Override metrics port from command line flags
	if metricsPort != 0 {
		config.ControllerRuntime.Metrics.BindAddress = fmt.Sprintf(":%d", metricsPort)
	}
	
	// Override enable swagger from command line flags
	if enableSwagger {
		config.APIServer.EnableSwagger = true
	}

	return config, nil
}

// ConfigCmd creates a command for working with configuration
// ToInformerOptions converts Config to informer.InformerOptions
func (c *Config) ToInformerOptions() *informer.InformerOptions {
	opts := &informer.InformerOptions{
		Namespace:          c.Informer.Namespace,
		ResyncPeriod:       c.Informer.ResyncPeriod,
		LabelSelector:      c.Informer.LabelSelector,
		FieldSelector:      c.Informer.FieldSelector,
		EnableEventLogging: c.Informer.Logging.EnableEventLogging,
		LogLevel:           c.Informer.Logging.LogLevel,
		QPS:                c.Kubernetes.QPS,
		Burst:              c.Kubernetes.Burst,
		Timeout:            c.Kubernetes.Timeout,
		DisableInformer:    c.Kubernetes.DisableInformer,
	}

	log.Debug().
		Str("namespace", opts.Namespace).
		Dur("resync_period", opts.ResyncPeriod).
		Str("label_selector", opts.LabelSelector).
		Str("field_selector", opts.FieldSelector).
		Bool("enable_event_logging", opts.EnableEventLogging).
		Str("log_level", opts.LogLevel).
		Float32("qps", opts.QPS).
		Int("burst", opts.Burst).
		Dur("timeout", opts.Timeout).
		Bool("disable_informer", opts.DisableInformer).
		Msg("Converted Config to InformerOptions")

	return opts
}

func ConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View and manage configuration for k8s-custom-controller",
	}

	configViewCmd := &cobra.Command{
		Use:   "view",
		Short: "View current configuration",
		Long:  "Display the current configuration being used",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := LoadConfig()
			if err != nil {
				log.Error().Err(err).Msg("Failed to load configuration")
				return
			}

			fmt.Println("Current Configuration:")
			fmt.Println("Kubernetes:")
			fmt.Printf("  Kubeconfig: %s\n", config.Kubernetes.Kubeconfig)
			fmt.Printf("  Context: %s\n", config.Kubernetes.Context)
			fmt.Printf("  Namespace: %s\n", config.Kubernetes.Namespace)
			fmt.Printf("  Timeout: %s\n", config.Kubernetes.Timeout)
			fmt.Printf("  QPS: %f\n", config.Kubernetes.QPS)
			fmt.Printf("  Burst: %d\n", config.Kubernetes.Burst)
			fmt.Printf("  InCluster: %t\n", config.Kubernetes.InCluster)
			fmt.Println("Logging:")
			fmt.Printf("  Level: %s\n", config.Logging.Level)
			fmt.Printf("  Format: %s\n", config.Logging.Format)

			// Add informer configuration output
			fmt.Println("Informer:")
			fmt.Printf("  Namespace: %s\n", config.Informer.Namespace)
			fmt.Printf("  ResyncPeriod: %s\n", config.Informer.ResyncPeriod)
			fmt.Printf("  LabelSelector: %s\n", config.Informer.LabelSelector)
			fmt.Printf("  FieldSelector: %s\n", config.Informer.FieldSelector)
			fmt.Println("  Logging:")
			fmt.Printf("    EnableEventLogging: %t\n", config.Informer.Logging.EnableEventLogging)
			fmt.Printf("    LogLevel: %s\n", config.Informer.Logging.LogLevel)
			fmt.Println("  Workers:")
			fmt.Printf("    Count: %d\n", config.Informer.Workers.Count)

			// Add controller runtime configuration output
			fmt.Println("ControllerRuntime:")
			fmt.Println("  LeaderElection:")
			fmt.Printf("    Enabled: %t\n", config.ControllerRuntime.LeaderElection.Enabled)
			fmt.Printf("    ID: %s\n", config.ControllerRuntime.LeaderElection.ID)
			fmt.Printf("    Namespace: %s\n", config.ControllerRuntime.LeaderElection.Namespace)
			fmt.Println("  Metrics:")
			fmt.Printf("    BindAddress: %s\n", config.ControllerRuntime.Metrics.BindAddress)
		},
	}

	configCmd.AddCommand(configViewCmd)
	return configCmd
}
