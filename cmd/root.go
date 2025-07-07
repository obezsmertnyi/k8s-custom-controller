package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/obezsmertnyi/k8s-custom-controller/pkg/informer"
)

var (
	logLevel   string
	cfgFile    string
	serverPort int
	serverHost string
	// enableSwagger is defined in api.go
	enableLeaderElection bool
	leaderElectionID     string
	leaderElectionNS     string
	metricsPort          int
	metricsBindAddress   string
	// kubeconfig defined in kubernetes.go
)

var rootCmd = &cobra.Command{
	Use:   "k8s-cli",
	Short: "Kubernetes custom controller and CLI tool",
	Long: `k8s-cli is a CLI tool and custom controller for Kubernetes.

It provides functionality for interacting with Kubernetes clusters,
managing resources, and implementing custom controllers.

Supports configuration via config files, command-line flags, and environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Now we load the config only once, when cfgFile is already set
		config, err := LoadConfig()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load configuration")
			return
		}

		// Debug log to show exactly what config we're using
		fmt.Printf("DEBUG: In rootCmd.Run with config.Logging.Level='%s'\n", config.Logging.Level)

		// Configure logger with the correct level from config
		level := parseLogLevel(config.Logging.Level)
		logFormat := config.Logging.Format
		
		// Check environment variable for log format override
		if envFormat := os.Getenv("KCUSTOM_LOGGING_FORMAT"); envFormat != "" {
			logFormat = envFormat
		}
		
		// Configure logger with the correct parameters
		configureLogger(level, logFormat)
		
		// Now the logger works with the correct level
		log.Trace().Msg("Logger reconfigured with settings from config file")
		log.Debug().Str("log_level", config.Logging.Level).Str("log_format", config.Logging.Format).Msg("Using configured log settings")

		// Check if --log-level flag was explicitly set
		if cmd.Flags().Changed("log-level") {
			// User explicitly set --log-level flag, so override config
			config.Logging.Level = logLevel
			level := parseLogLevel(logLevel)
			log.Debug().Str("log_level", logLevel).Msg("Overriding log level from command line")
			configureLogger(level, config.Logging.Format)
		}

		// Apply command line flags to configuration if provided

		// Apply API server settings
		if cmd.Flags().Changed("host") {
			config.APIServer.Host = serverHost
			log.Debug().Str("host", serverHost).Msg("Applied API server host from command line")
		}

		if cmd.Flags().Changed("port") {
			config.APIServer.Port = serverPort
			log.Debug().Int("port", serverPort).Msg("Applied API server port from command line")
		}

		if cmd.Flags().Changed("enable-swagger") {
			config.APIServer.EnableSwagger = enableSwagger
			// Also set the global variable used in api.go
			enableSwagger = enableSwagger
			log.Debug().Bool("enable_swagger", enableSwagger).Msg("Applied Swagger UI setting from command line")
		}

		// Apply leader election settings
		if cmd.Flags().Changed("enable-leader-election") {
			config.ControllerRuntime.LeaderElection.Enabled = enableLeaderElection
			log.Debug().Bool("enabled", enableLeaderElection).Msg("Applied leader election flag from command line")
		}

		if cmd.Flags().Changed("leader-election-id") {
			config.ControllerRuntime.LeaderElection.ID = leaderElectionID
			log.Debug().Str("id", leaderElectionID).Msg("Applied leader election ID from command line")
		}

		if cmd.Flags().Changed("leader-election-namespace") {
			config.ControllerRuntime.LeaderElection.Namespace = leaderElectionNS
			log.Debug().Str("namespace", leaderElectionNS).Msg("Applied leader election namespace from command line")
		}

		// Apply metrics settings
		if cmd.Flags().Changed("metrics-port") {
			config.ControllerRuntime.Metrics.BindAddress = fmt.Sprintf("%s:%d", metricsBindAddress, metricsPort)
			log.Debug().Str("bind_address", config.ControllerRuntime.Metrics.BindAddress).Msg("Applied metrics settings from command line")
		}

		// Start all components (API server and informer)
		if err := StartComponents(config); err != nil {
			log.Error().Err(err).Msg("Failed to start components")
			return
		}
	},
}

func parseLogLevel(lvl string) zerolog.Level {
	fmt.Printf("DEBUG: parseLogLevel called with '%s'\n", lvl)

	var result zerolog.Level
	switch strings.ToLower(lvl) {
	case "trace":
		result = zerolog.TraceLevel
	case "debug":
		result = zerolog.DebugLevel
	case "info":
		result = zerolog.InfoLevel
	case "warn":
		result = zerolog.WarnLevel
	case "error":
		result = zerolog.ErrorLevel
	default:
		result = zerolog.InfoLevel
	}

	fmt.Printf("DEBUG: parseLogLevel returning %v (trace=-1, debug=0, info=1)\n", result)
	return result
}

// configureLogger sets up the zerolog logger with the specified level and format
func configureLogger(level zerolog.Level, format string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(level)

	// Configure logging format based on configuration
	switch strings.ToLower(format) {
	case "json":
		// JSON format for structured logging (better for machine processing)
		logger := zerolog.New(os.Stderr).With().Timestamp()
		if level == zerolog.TraceLevel {
			logger = logger.Caller()
		}
		log.Logger = logger.Logger()
	default: // "text" or any other format
		// Console format for better human readability
		console := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05.000",
		}

		if level == zerolog.TraceLevel {
			zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
				return fmt.Sprintf("%s:%d", file, line)
			}
			zerolog.CallerFieldName = "caller"
			console.PartsOrder = []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			}
			log.Logger = log.Output(console).With().Caller().Logger()
		} else {
			console.PartsOrder = []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.MessageFieldName,
			}
			log.Logger = log.Output(console)
		}
	}
}

func setupLogging() {
	logFormat := "text"
	if envFormat := os.Getenv("KCUSTOM_LOGGING_FORMAT"); envFormat != "" {
		logFormat = envFormat
	}
	configureLogger(parseLogLevel("info"), logFormat)
}

func Execute() error {
	// Temporary basic logger setup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	
	// Basic logger configuration for initial output
	// Using INFO level by default
	configureLogger(zerolog.InfoLevel, "text")
	
	return rootCmd.Execute()
}

func init() {
	// Add persistent flags for root command
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default is $HOME/.k8s-custom-controller/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", getDefaultKubeconfig(), "Path to the kubeconfig file (default: ~/.kube/config)")

	// Add flags for leader election
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager")
	rootCmd.Flags().StringVar(&leaderElectionID, "leader-election-id", "k8s-custom-controller-leader-election", "ID for leader election")
	rootCmd.Flags().StringVar(&leaderElectionNS, "leader-election-namespace", "default", "Namespace for leader election resources")

	// Add flags for metrics
	rootCmd.Flags().IntVar(&metricsPort, "metrics-port", 8081, "Port for controller manager metrics")
	rootCmd.Flags().StringVar(&metricsBindAddress, "metrics-bind-address", "0.0.0.0", "Bind address for metrics server")

	rootCmd.AddCommand(ConfigCmd())

	// Setup informer defaults in Viper
	informer.SetupInformerDefaults(viper.GetViper())
}
