package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// apiServer holds the Kubernetes client for API handlers
type apiServer struct {
	clientset *kubernetes.Clientset
}

// requestHandler processes HTTP requests with logging
func (s *apiServer) requestHandler(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	method := string(ctx.Method())
	path := string(ctx.Path())
	clientIP := ctx.RemoteIP().String()

	// Log request details at debug level
	log.Debug().Str("method", method).Str("path", path).Str("client", clientIP).Msg("Request received")

	// Set content type for JSON responses
	ctx.SetContentType("application/json; charset=utf8")

	// Route handling based on path
	switch {
	case string(ctx.Path()) == "/health":
		s.handleHealth(ctx)
	case string(ctx.Path()) == "/deployments":
		s.handleDeployments(ctx)
	default:
		// Handle unknown paths
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "Not found"})
	}

	// Log response details at info level
	log.Info().Str("method", method).Str("path", path).Str("client", clientIP).Int("status", ctx.Response.StatusCode()).Dur("duration", time.Since(start)).Msg("Request processed")
}

// handleHealth responds with server health status
func (s *apiServer) handleHealth(ctx *fasthttp.RequestCtx) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Add Kubernetes client status if available
	if s.clientset != nil {
		response["kubernetes_connected"] = true
	} else {
		response["kubernetes_connected"] = false
	}

	json.NewEncoder(ctx).Encode(response)
}

// handleDeployments lists deployments from Kubernetes
func (s *apiServer) handleDeployments(ctx *fasthttp.RequestCtx) {
	// Check if Kubernetes client is available
	if s.clientset == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Kubernetes client not configured",
		})
		return
	}

	// Get namespace from query parameter or use default
	namespace := string(ctx.QueryArgs().Peek("namespace"))
	if namespace == "" {
		namespace = "default"
	}

	// Get deployments from Kubernetes
	deployments, err := s.clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Str("namespace", namespace).Msg("Failed to list deployments")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Failed to list deployments",
		})
		return
	}

	// Convert to simplified response format
	response := map[string]interface{}{
		"namespace": namespace,
		"count":     len(deployments.Items),
		"items":     []interface{}{},
	}

	// Add deployment items
	items := make([]interface{}, 0, len(deployments.Items))
	for _, d := range deployments.Items {
		items = append(items, map[string]interface{}{
			"name":      d.Name,
			"replicas":  d.Status.Replicas,
			"ready":     d.Status.ReadyReplicas,
			"available": d.Status.AvailableReplicas,
			"created":   d.CreationTimestamp.Format(time.RFC3339),
		})
	}
	response["items"] = items

	json.NewEncoder(ctx).Encode(response)
}

// Starts FastHTTP API server
func StartAPIServer(ctx context.Context, clientset *kubernetes.Clientset, host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &apiServer{clientset: clientset}

	// Create FastHTTP server
	httpServer := &fasthttp.Server{
		Handler: server.requestHandler,
		Name:    "k8s-cli API Server",
	}

	log.Info().Str("address", addr).Msg("Starting FastHTTP API server")

	// Start server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(addr); err != nil {
			log.Error().Err(err).Msg("Server error")
		}
	}()

	// Wait for context to finish
	<-ctx.Done()
	log.Info().Msg("Shutting down API server...")

	// Give server 5 seconds to finish current requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
		return err
	}

	log.Info().Msg("API server gracefully stopped")
	return nil
}

func init() {
	// Add server flags to root command
	rootCmd.PersistentFlags().StringVar(&serverHost, "host", "0.0.0.0", "Host address to bind the server to")
	rootCmd.PersistentFlags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
}
