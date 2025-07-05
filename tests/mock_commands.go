package tests

import (
	"github.com/spf13/cobra"
)

// MockServerCommand creates a mock server command for testing
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
