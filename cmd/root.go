package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	logLevel string
	cfgFile  string
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

		// Demonstration of different logging levels
		log.Info().Msg("This is an info log")
		log.Debug().Msg("This is a debug log")
		log.Trace().Msg("This is a trace log")
		log.Warn().Msg("This is a warn log")
		log.Error().Msg("This is an error log")

		// Display configuration information
		log.Info().Str("namespace", config.Kubernetes.Namespace).Msg("Using Kubernetes namespace")
		log.Info().Bool("in_cluster", config.Kubernetes.InCluster).Msg("Kubernetes client mode")

		fmt.Println("Welcome to k8s-custom-controller CLI!")
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

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default is $HOME/.k8s-custom-controller/config.yaml)")
	rootCmd.AddCommand(ConfigCmd())
}
