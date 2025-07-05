package informer

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// InformerOptions contains options for configuring the deployment informer
type InformerOptions struct {
	Namespace         string        // Namespace to watch, empty for all namespaces
	ResyncPeriod      time.Duration // How often to resync the cache
	LabelSelector     string        // Label selector for filtering resources
	FieldSelector     string        // Field selector for filtering resources
	EnableEventLogging bool         // Whether to log events
	LogLevel          string        // Log level for informer logs
	QPS               float32       // QPS indicates the maximum QPS to the master from this client
	Burst             int           // Maximum burst for throttle
	Timeout           time.Duration // Timeout for operations
	DisableInformer   bool          // Whether to disable informer
}

// DefaultInformerOptions returns default options for the informer
func DefaultInformerOptions() *InformerOptions {
	return &InformerOptions{
		Namespace:         "default",
		ResyncPeriod:      30 * time.Second,
		LabelSelector:     "",
		FieldSelector:     "",
		EnableEventLogging: true,
		LogLevel:           "info",
		QPS:               5.0,
		Burst:             10,
		Timeout:           10 * time.Second,
		DisableInformer:   false,
	}
}

// GetInformerOptionsFromViper loads informer options from Viper config
func GetInformerOptionsFromViper() (*InformerOptions, error) {
	opts := DefaultInformerOptions()

	// Check if informer section exists in config
	if !viper.IsSet("informer") {
		log.Warn().Msg("No 'informer' section in config, using defaults")
		return opts, nil
	}

	// Load namespace from config
	if viper.IsSet("informer.namespace") {
		opts.Namespace = viper.GetString("informer.namespace")
	}

	// Load resync period from config
	if viper.IsSet("informer.resync_period") {
		opts.ResyncPeriod = viper.GetDuration("informer.resync_period")
	}

	// Load label selector from config
	if viper.IsSet("informer.label_selector") {
		opts.LabelSelector = viper.GetString("informer.label_selector")
	}

	// Load field selector from config
	if viper.IsSet("informer.field_selector") {
		opts.FieldSelector = viper.GetString("informer.field_selector")
	}

	// Load logging configuration
	if viper.IsSet("informer.enable_event_logging") {
		opts.EnableEventLogging = viper.GetBool("informer.enable_event_logging")
	}

	if viper.IsSet("informer.log_level") {
		opts.LogLevel = viper.GetString("informer.log_level")
	}

	// Load kubernetes client configuration from config
	if viper.IsSet("kubernetes.qps") {
		opts.QPS = float32(viper.GetFloat64("kubernetes.qps"))
	}

	if viper.IsSet("kubernetes.burst") {
		opts.Burst = viper.GetInt("kubernetes.burst")
	}

	if viper.IsSet("kubernetes.timeout") {
		opts.Timeout = viper.GetDuration("kubernetes.timeout")
	}

	// Check if informer is disabled
	if viper.IsSet("kubernetes.disable_informer") {
		opts.DisableInformer = viper.GetBool("kubernetes.disable_informer")
	}

	log.Info().
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
		Msg("Loaded informer options from config")

	return opts, nil
}

// SetupInformerDefaults configures default values for the informer in Viper
func SetupInformerDefaults(v *viper.Viper) {
	// Установка дефолтных значений для информера
	defaults := DefaultInformerOptions()

	// Установка значений по умолчанию для пространства имен и периода синхронизации
	v.SetDefault("informer.namespace", defaults.Namespace)
	v.SetDefault("informer.resync_period", defaults.ResyncPeriod)
	
	// Установка значений по умолчанию для селекторов
	v.SetDefault("informer.label_selector", defaults.LabelSelector)
	v.SetDefault("informer.field_selector", defaults.FieldSelector)

	// Установка значений по умолчанию для логирования
	v.SetDefault("informer.enable_event_logging", defaults.EnableEventLogging)
	v.SetDefault("informer.log_level", defaults.LogLevel)

	// Установка значений по умолчанию для клиента Kubernetes
	v.SetDefault("kubernetes.qps", defaults.QPS)
	v.SetDefault("kubernetes.burst", defaults.Burst)
	v.SetDefault("kubernetes.timeout", defaults.Timeout)
	v.SetDefault("kubernetes.disable_informer", defaults.DisableInformer)
}
