package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/obezsmertnyi/k8s-custom-controller/pkg/informer"
)

// apiServer holds the Kubernetes client and informer factory for API handlers
type apiServer struct {
	clientset *kubernetes.Clientset
	informerFactory informers.SharedInformerFactory
}

// requestHandler processes HTTP requests with logging
func (s *apiServer) requestHandler(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	method := string(ctx.Method())
	path := string(ctx.Path())
	clientIP := ctx.RemoteIP().String()

	// Generate unique request ID
	requestID := uuid.New().String()
	ctx.Response.Header.Set("X-Request-ID", requestID)

	// Create logger with request ID
	logger := log.With().Str("request_id", requestID).Logger()

	// Log request details
	logger.Debug().Str("method", method).Str("path", path).Str("client", clientIP).Msg("Request received")

	// Set content type for JSON responses
	ctx.SetContentType("application/json; charset=utf8")

	// Route handling based on path
	switch {
	case string(ctx.Path()) == "/health":
		s.handleHealth(ctx)
	case string(ctx.Path()) == "/deployments":
		s.handleDeployments(ctx)
	case string(ctx.Path()) == "/pods":
		s.handlePods(ctx)
	case string(ctx.Path()) == "/services":
		s.handleServices(ctx)
	case string(ctx.Path()) == "/nodes":
		s.handleNodes(ctx)
	default:
		// Handle unknown paths
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "Not found"})
	}

	// Log response details at info level
	latency := time.Since(start)
	statusCode := ctx.Response.StatusCode()
	logger.Info().Str("method", method).Str("path", path).Int("status", statusCode).Dur("latency", latency).Msg("Request completed")
}

// handleHealth responds with server health status
func (s *apiServer) handleHealth(ctx *fasthttp.RequestCtx) {
	// Get the logger with request_id from context
	logger := log.With().Str("request_id", string(ctx.Response.Header.Peek("X-Request-ID"))).Logger()
	logger.Info().Msg("Health check request received")

	response := map[string]interface{}{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0", // Assuming you have a GetVersion function or constant
	}

	// Add Kubernetes client status if available
	if s.clientset != nil {
		response["kubernetes_connected"] = true
	} else {
		response["kubernetes_connected"] = false
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

// handleDeployments lists deployments from the informer cache
func (s *apiServer) handleDeployments(ctx *fasthttp.RequestCtx) {
	// Get the logger with request ID
	logger := getRequestLogger(ctx)
	logger.Info().Msg("Deployments request received")

	// Check if informer factory is available
	if s.informerFactory == nil {
		logger.Error().Msg("Informer factory not configured")
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Informer factory not configured",
		})
		return
	}

	// Get namespace from query parameter
	namespace := getNamespaceFromQuery(ctx)

	// Get deployment informer
	deploymentInformer := s.informerFactory.Apps().V1().Deployments().Informer()
	
	// Get deployments from cache
	deployments, err := informer.ListDeploymentsInCache(deploymentInformer, namespace)
	if err != nil {
		logger.Error().Err(err).Str("namespace", namespace).Msg("Failed to list deployments from cache")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Failed to list deployments from cache",
		})
		return
	}
	
	logger.Info().Int("count", len(deployments)).Str("namespace", namespace).Msg("Deployments retrieved from cache")

	// Set response headers
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	
	// Extract deployment names
	names := make([]string, 0, len(deployments))
	for _, d := range deployments {
		names = append(names, d.Name)
	}
	
	// Log the names
	logger.Info().Msgf("Deployments: %v", names)
	
	// Simple format if requested
	if isSimpleFormat(ctx) {
		writeSimpleJsonArray(ctx, names)
		return
	}
	
	// Full detailed response (default)
	response := map[string]interface{}{
		"namespace": namespace,
		"count":     len(deployments),
		"source":    "informer-cache", // Indicate data comes from cache
		"names":     names,           // Simple names array
		"items":     []interface{}{}, // Detailed items
	}

	// Add detailed deployment items
	items := make([]interface{}, 0, len(deployments))
	for _, d := range deployments {
		items = append(items, map[string]interface{}{
			"name":      d.Name,
			"replicas":  d.Status.Replicas,
			"available": d.Status.AvailableReplicas,
		})
	}
	response["items"] = items

	// Return detailed JSON response
	json.NewEncoder(ctx).Encode(response)
}

// handlePods lists pods from the informer cache or Kubernetes API
func (s *apiServer) handlePods(ctx *fasthttp.RequestCtx) {
	// Get the logger with request ID
	logger := getRequestLogger(ctx)
	logger.Info().Msg("Pods request received")

	// Check if Kubernetes client is available
	if !s.checkKubeClient(ctx, logger) {
		return
	}

	// Get namespace from query parameter
	namespace := getNamespaceFromQuery(ctx)

	// Get pods directly from Kubernetes API
	pods, err := s.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Error().Err(err).Str("namespace", namespace).Msg("Failed to list pods")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Failed to list pods",
		})
		return
	}

	logger.Info().Int("count", len(pods.Items)).Str("namespace", namespace).Msg("Pods retrieved")
	
	// Set response headers
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	
	// Extract pod names
	names := make([]string, 0, len(pods.Items))
	for _, pod := range pods.Items {
		names = append(names, pod.Name)
	}
	
	// Log the names
	logger.Info().Msgf("Pods: %v", names)
	
	// Simple format if requested
	if isSimpleFormat(ctx) {
		writeSimpleJsonArray(ctx, names)
		return
	}
	
	// Full detailed response
	response := map[string]interface{}{
		"namespace": namespace,
		"count":     len(pods.Items),
		"source":    "kubernetes-api",
		"names":     names,
		"items":     []interface{}{},
	}

	// Add detailed pod items
	items := make([]interface{}, 0, len(pods.Items))
	for _, pod := range pods.Items {
		items = append(items, map[string]interface{}{
			"name":      pod.Name,
			"phase":     string(pod.Status.Phase),
			"node":      pod.Spec.NodeName,
			"ip":        pod.Status.PodIP,
			"created":   pod.CreationTimestamp.Format(time.RFC3339),
		})
	}
	response["items"] = items

	// Return JSON response
	json.NewEncoder(ctx).Encode(response)
}

// handleServices lists services from Kubernetes API
func (s *apiServer) handleServices(ctx *fasthttp.RequestCtx) {
	// Get the logger with request ID
	logger := getRequestLogger(ctx)
	logger.Info().Msg("Services request received")

	// Check if Kubernetes client is available
	if !s.checkKubeClient(ctx, logger) {
		return
	}

	// Get namespace from query parameter
	namespace := getNamespaceFromQuery(ctx)

	// Get services from Kubernetes API
	services, err := s.clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Error().Err(err).Str("namespace", namespace).Msg("Failed to list services")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Failed to list services",
		})
		return
	}

	logger.Info().Int("count", len(services.Items)).Str("namespace", namespace).Msg("Services retrieved")
	
	// Set response headers
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	
	// Extract service names
	names := make([]string, 0, len(services.Items))
	for _, svc := range services.Items {
		names = append(names, svc.Name)
	}
	
	// Log the names
	logger.Info().Msgf("Services: %v", names)
	
	// Simple format if requested
	if isSimpleFormat(ctx) {
		writeSimpleJsonArray(ctx, names)
		return
	}
	
	// Full detailed response
	response := map[string]interface{}{
		"namespace": namespace,
		"count":     len(services.Items),
		"source":    "kubernetes-api",
		"names":     names,
		"items":     []interface{}{},
	}

	// Add detailed service items
	items := make([]interface{}, 0, len(services.Items))
	for _, svc := range services.Items {
		portInfo := make([]map[string]interface{}, 0, len(svc.Spec.Ports))
		for _, port := range svc.Spec.Ports {
			portInfo = append(portInfo, map[string]interface{}{
				"name":       port.Name,
				"port":       port.Port,
				"targetPort": port.TargetPort.String(),
				"protocol":   string(port.Protocol),
			})
		}
		
		items = append(items, map[string]interface{}{
			"name":      svc.Name,
			"type":      string(svc.Spec.Type),
			"clusterIP": svc.Spec.ClusterIP,
			"ports":     portInfo,
			"created":   svc.CreationTimestamp.Format(time.RFC3339),
		})
	}
	response["items"] = items

	// Return JSON response
	json.NewEncoder(ctx).Encode(response)
}

// handleNodes lists nodes from Kubernetes API
func (s *apiServer) handleNodes(ctx *fasthttp.RequestCtx) {
	// Get the logger with request ID
	logger := getRequestLogger(ctx)
	logger.Info().Msg("Nodes request received")

	// Check if Kubernetes client is available
	if !s.checkKubeClient(ctx, logger) {
		return
	}

	// Get nodes from Kubernetes API
	nodes, err := s.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list nodes")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{
			"error": "Failed to list nodes",
		})
		return
	}

	logger.Info().Int("count", len(nodes.Items)).Msg("Nodes retrieved")
	
	// Set response headers
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	
	// Extract node names
	names := make([]string, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		names = append(names, node.Name)
	}
	
	// Log the names
	logger.Info().Msgf("Nodes: %v", names)
	
	// Simple format if requested
	if isSimpleFormat(ctx) {
		writeSimpleJsonArray(ctx, names)
		return
	}
	
	// Full detailed response
	response := map[string]interface{}{
		"count":  len(nodes.Items),
		"source": "kubernetes-api",
		"names":  names,
		"items":  []interface{}{},
	}

	// Add detailed node items with key info
	items := make([]interface{}, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		// Get node capacity
		capacity := make(map[string]string)
		for k, v := range node.Status.Capacity {
			capacity[string(k)] = v.String()
		}
		
		// Get node conditions
		conditions := make([]map[string]interface{}, 0, len(node.Status.Conditions))
		for _, condition := range node.Status.Conditions {
			if condition.Status == "True" {
				conditions = append(conditions, map[string]interface{}{
					"type":   string(condition.Type),
					"status": string(condition.Status),
				})
			}
		}
		
		// Get node addresses
		addresses := make(map[string]string)
		for _, addr := range node.Status.Addresses {
			addresses[string(addr.Type)] = addr.Address
		}
		
		items = append(items, map[string]interface{}{
			"name":       node.Name,
			"addresses":  addresses,
			"conditions": conditions,
			"capacity":   capacity,
			"version":    node.Status.NodeInfo.KubeletVersion,
			"created":    node.CreationTimestamp.Format(time.RFC3339),
		})
	}
	response["items"] = items

	// Return JSON response
	json.NewEncoder(ctx).Encode(response)
}

// Helper functions to reduce code duplication

// getRequestLogger creates a logger with the request ID from context
func getRequestLogger(ctx *fasthttp.RequestCtx) zerolog.Logger {
	return log.With().Str("request_id", string(ctx.Response.Header.Peek("X-Request-ID"))).Logger()
}

// writeSimpleJsonArray writes a simple JSON array of strings
func writeSimpleJsonArray(ctx *fasthttp.RequestCtx, names []string) {
	ctx.Write([]byte("["))
	for i, name := range names {
		ctx.WriteString("\"")
		ctx.WriteString(name)
		ctx.WriteString("\"")
		if i < len(names)-1 {
			ctx.WriteString(",")
		}
	}
	ctx.Write([]byte("]"))
}

// checkKubeClient verifies if Kubernetes client is available
func (s *apiServer) checkKubeClient(ctx *fasthttp.RequestCtx, logger zerolog.Logger) bool {
	if s.clientset == nil {
		logger.Error().Msg("Kubernetes client not configured")
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "Kubernetes client not configured"})
		return false
	}
	return true
}

// getNamespaceFromQuery extracts namespace from query parameters
func getNamespaceFromQuery(ctx *fasthttp.RequestCtx) string {
	return string(ctx.QueryArgs().Peek("namespace"))
}

// isSimpleFormat checks if simple response format is requested
func isSimpleFormat(ctx *fasthttp.RequestCtx) bool {
	return string(ctx.QueryArgs().Peek("format")) == "simple"
}

// Starts FastHTTP API server
func StartAPIServer(ctx context.Context, clientset *kubernetes.Clientset, factory informers.SharedInformerFactory, host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &apiServer{
		clientset:      clientset,
		informerFactory: factory,
	}

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
