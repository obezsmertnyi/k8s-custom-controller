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
)

// Config structure for storing application configuration
type Config struct {
	// Kubernetes settings
	Kubernetes struct {
		Kubeconfig string        `mapstructure:"kubeconfig"`
		Context    string        `mapstructure:"context"`
		Namespace  string        `mapstructure:"namespace"`
		Timeout    time.Duration `mapstructure:"timeout"`
		QPS        float32       `mapstructure:"qps"`
		Burst      int           `mapstructure:"burst"`
		InCluster  bool          `mapstructure:"in_cluster"`
	} `mapstructure:"kubernetes"`

	// Logging settings
	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
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
	config.Logging.Level = "info"
	config.Logging.Format = "text"

	// Set default kubeconfig path
	if home := homeDir(); home != "" {
		config.Kubernetes.Kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Configure Viper
	if cfgFile != "" {
		// Use configuration file specified via --config flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for default configuration file
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Search for configuration file in standard locations
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.k8s-custom-controller")
	viper.AddConfigPath("/etc/k8s-custom-controller")

	// Automatically use environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("KCUSTOM") // KCUSTOM_KUBERNETES_NAMESPACE
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind environment variables to configuration keys
	viper.BindEnv("kubernetes.namespace")
	viper.BindEnv("kubernetes.kubeconfig")
	viper.BindEnv("kubernetes.qps")
	viper.BindEnv("kubernetes.burst")
	viper.BindEnv("kubernetes.timeout")
	viper.BindEnv("kubernetes.in_cluster")
	viper.BindEnv("logging.level")
	viper.BindEnv("logging.format")

	// Try to read configuration file
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
		log.Warn().Msg("No config file found, using defaults and environment variables")
	}

	// Bind environment variables to configuration
	err := viper.Unmarshal(config)
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
		},
	}

	configCmd.AddCommand(configViewCmd)
	return configCmd
}
