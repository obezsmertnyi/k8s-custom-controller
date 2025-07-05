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
		DisableInformer bool          `mapstructure:"disable_informer"`
		DisableAPI      bool          `mapstructure:"disable_api"`
	} `mapstructure:"kubernetes"`

	// Logging settings
	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`

	// Informer settings
	Informer struct {
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

	// Set default values
	config.Kubernetes.Timeout = 30 * time.Second
	config.Kubernetes.QPS = 50
	config.Kubernetes.Burst = 100
	config.Kubernetes.Namespace = "default"
	config.Kubernetes.DisableInformer = false // Enable informer by default
	config.Logging.Level = "info"
	config.Logging.Format = "text"

	// Default values for informer
	config.Informer.Namespace = "default"
	config.Informer.ResyncPeriod = 30 * time.Second
	config.Informer.LabelSelector = ""
	config.Informer.FieldSelector = ""
	config.Informer.Logging.EnableEventLogging = true
	config.Informer.Logging.LogLevel = "info"
	config.Informer.Workers.Count = 2

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
	viper.BindEnv("kubernetes.namespace")
	viper.BindEnv("kubernetes.kubeconfig")
	viper.BindEnv("kubernetes.qps")
	viper.BindEnv("kubernetes.burst")
	viper.BindEnv("kubernetes.timeout")
	viper.BindEnv("kubernetes.in_cluster")
	viper.BindEnv("kubernetes.disable_informer")
	viper.BindEnv("logging.level")
	viper.BindEnv("logging.format")

	// Bind informer configuration
	viper.BindEnv("informer.namespace")
	viper.BindEnv("informer.resync_period")
	viper.BindEnv("informer.label_selector")
	viper.BindEnv("informer.field_selector")
	viper.BindEnv("informer.logging.enable_event_logging")
	viper.BindEnv("informer.logging.log_level")
	viper.BindEnv("informer.workers.count")

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
		},
	}

	configCmd.AddCommand(configViewCmd)
	return configCmd
}
