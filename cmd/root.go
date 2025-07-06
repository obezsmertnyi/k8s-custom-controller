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
	logLevel              string
	cfgFile               string
	serverPort            int
	serverHost            string
	// enableSwagger is defined in api.go
	enableLeaderElection  bool
	leaderElectionID      string
	leaderElectionNS      string
	metricsPort           int
	metricsBindAddress    string
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
		// Load configuration
		config, err := LoadConfig()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load configuration")
			return
		}

		// Configure logging based on configuration
		level := parseLogLevel(config.Logging.Level)
		configureLogger(level, config.Logging.Format)
		
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
	switch strings.ToLower(lvl) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
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
	// Initial logging setup to display errors during configuration loading
	setupLogging()

	config, err := LoadConfig()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return err
	}

	// Apply log level settings from configuration
	level := parseLogLevel(config.Logging.Level)
	
	// Determine log format considering priority order
	logFormat := config.Logging.Format
	if envFormat := os.Getenv("KCUSTOM_LOGGING_FORMAT"); envFormat != "" {
		// Environment variable has priority over configuration from file
		logFormat = envFormat
		log.Debug().Str("format", envFormat).Msg("Using log format from environment variable")
	}

	// Configure logging with correct priority order
	configureLogger(level, logFormat)

	return rootCmd.Execute()
}

func init() {
	// Add persistent flags for root command
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default is $HOME/.k8s-custom-controller/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", getDefaultKubeconfig(), "Path to the kubeconfig file (default: ~/.kube/config)")

	// Add flags for API server
	rootCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Host address to bind the server to")
	rootCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
	rootCmd.Flags().BoolVar(&enableSwagger, "enable-swagger", true, "Enable Swagger UI documentation")
	
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
