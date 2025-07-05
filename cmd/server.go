package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

var (
	serverPort int
	serverHost string
)

// requestHandler processes HTTP requests with logging
func requestHandler(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	method := string(ctx.Method())
	path := string(ctx.Path())
	clientIP := ctx.RemoteIP().String()

	// Log request details at debug level
	log.Debug().Str("method", method).Str("path", path).Str("client", clientIP).Msg("Request received")

	// Set content type and respond
	ctx.SetContentType("text/plain; charset=utf8")
	fmt.Fprintf(ctx, "Hello from FastHTTP! You requested: %s from IP: %s", path, clientIP)

	// Log response details at info level
	log.Info().Str("method", method).Str("path", path).Str("client", clientIP).Int("status", ctx.Response.StatusCode()).Dur("duration", time.Since(start)).Msg("Request processed")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Long: `Start a FastHTTP server with configurable host and port.

Example:
  k8s-cli server --host 0.0.0.0 --port 8080`,
	Run: func(cmd *cobra.Command, args []string) {
		addr := fmt.Sprintf("%s:%d", serverHost, serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)

		// Configure server with sensible defaults
		server := &fasthttp.Server{
			Handler:      requestHandler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
			// 10MB max request size for safety
			MaxRequestBodySize: 10 * 1024 * 1024,
		}

		// Create context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Start server in a goroutine
		go func() {
			log.Info().Msg("Press Ctrl+C to stop the server")
			if err := server.ListenAndServe(addr); err != nil {
				// FastHTTP не має константи ErrServerClosed, тому просто логуємо помилку
				log.Error().Err(err).Msg("Server stopped")
				cancel() // Cancel context on server error
			}
		}()

		// Wait for termination signal
		select {
			case <-sigCh:
				log.Info().Msg("Shutdown signal received, stopping server...")
			case <-ctx.Done():
				log.Info().Msg("Server stopped due to error")
		}

		// Shutdown with timeout
		shutdownTimeout := 5 * time.Second
		log.Info().Msgf("Gracefully shutting down server (timeout: %s)...", shutdownTimeout)

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		// Shutdown the server
		if err := server.ShutdownWithContext(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Error during server shutdown")
		} else {
			log.Info().Msg("Server gracefully stopped")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&serverHost, "host", "127.0.0.1", "Host address to bind the server to")
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
}
