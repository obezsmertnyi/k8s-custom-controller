package tests

import (
	"github.com/spf13/cobra"
)

// MockServerCommand creates a mock server command for testing
// This is kept for backward compatibility with existing tests
func MockServerCommand() *cobra.Command {
	// Create a new command with the same properties as the real server command
	mockCmd := &cobra.Command{
		Use:   "server",
		Short: "Start a FastHTTP server",
		Long:  `Start a FastHTTP server with configurable host and port.`,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Add the same flags as the real command
	var serverHost string
	var serverPort int
	mockCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Host address to bind the server to")
	mockCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")

	return mockCmd
}

// MockCliCommand creates a mock k8s-cli command for testing
func MockCliCommand() *cobra.Command {
	// Create a new command with the same properties as the real root CLI command
	mockCmd := &cobra.Command{
		Use:   "k8s-cli",
		Short: "Kubernetes custom controller and CLI tool",
		Long:  `k8s-cli is a CLI tool and custom controller for Kubernetes.`,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Add the same flags as the real command
	var serverHost string
	var serverPort int
	var enableSwagger bool
	var logLevel string
	var cfgFile string
	
	// Add persistent flags
	mockCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
	mockCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default is $HOME/.k8s-custom-controller/config.yaml)")
	
	// Add API server flags
	mockCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Host address to bind the server to")
	mockCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
	mockCmd.Flags().BoolVar(&enableSwagger, "enable-swagger", true, "Enable Swagger UI documentation")

	return mockCmd
}
