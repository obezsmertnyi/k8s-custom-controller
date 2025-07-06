// Package cmd contains the main application commands and API implementation
// @title Kubernetes Multi-Cluster Controller API
// @version 1.0
// @description API server for managing and monitoring Kubernetes resources across multiple clusters
// @BasePath /

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/swaggo/swag"
	"github.com/valyala/fasthttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	// Import the docs package to ensure Swagger docs are registered
	_ "github.com/obezsmertnyi/k8s-custom-controller/docs"
	"github.com/obezsmertnyi/k8s-custom-controller/pkg/ctrl"
	"github.com/obezsmertnyi/k8s-custom-controller/pkg/informer"
)

// apiServer holds the Kubernetes client and informer factory for API handlers
type apiServer struct {
	clientset       *kubernetes.Clientset
	informerFactory informers.SharedInformerFactory
	config          *Config // Reference to application config for API settings
	// Multi-cluster deployment controller manager
	multiClusterManager *ctrl.MultiClusterManager
	// Request rate limiter
	ipLimiter         *perIPLimiter   // Per-IP rate limiter
	requestLimiter    *time.Ticker    // Legacy global rate limiter (deprecated)
	requestLimiterMux sync.Mutex      // Mutex to protect rate limiter initialization
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

	// Apply rate limiting based on configuration
	if s.config != nil && s.config.APIServer.Security.RateLimitRequestsPerSecond > 0 {
		if !s.checkRateLimit(clientIP, logger) {
			ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
			ctx.SetBodyString(`{"error": "Rate limit exceeded", "retry_after": "1s"}`)
			logger.Warn().Str("client_ip", clientIP).Int("limit", s.config.APIServer.Security.RateLimitRequestsPerSecond).Msg("Rate limit exceeded")
			return
		}
	}

	// Set security headers
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	ctx.Response.Header.Set("X-Frame-Options", "DENY")

	// Log request details
	logger.Debug().Str("method", method).Str("path", path).Str("client", clientIP).Msg("Request received")

	// Set content type for JSON responses
	ctx.SetContentType("application/json; charset=utf8")

	// Route handling based on path
	switch {
	case strings.HasPrefix(string(ctx.Path()), "/swagger/") || string(ctx.Path()) == "/swagger.json" || string(ctx.Path()) == "/swagger":
		if s.config != nil && s.config.APIServer.EnableSwagger {
			path := string(ctx.Path())

			// Handle Swagger JSON endpoint
			if path == "/swagger/swagger.json" || path == "/swagger.json" {
				s.handleSwaggerJSON(ctx)
				return
			}

			// Handle Swagger UI - redirect both /swagger and /swagger/ to index.html
			if path == "/swagger/" || path == "/swagger" {
				// Redirect to index.html
				ctx.Redirect("/swagger/index.html", fasthttp.StatusFound)
				return
			}

			// For Swagger index.html - serve UI
			if path == "/swagger/index.html" {
				s.serveSwaggerUI(ctx)
				return
			}
		} else {
			// Swagger is disabled
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			json.NewEncoder(ctx).Encode(map[string]string{"error": "Not found"})
		}
	case string(ctx.Path()) == "/health":
		s.handleHealth(ctx)
	case string(ctx.Path()) == "/clusters":
		s.handleClusters(ctx)
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

// @Summary Get API server health status
// @Description Returns health status of the API server and Kubernetes connection state
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
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

// @Summary Get Kubernetes deployments
// @Description Returns list of Kubernetes deployments across all connected clusters
// @Tags kubernetes,deployments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /deployments [get]
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
		"names":     names,            // Simple names array
		"items":     []interface{}{},  // Detailed items
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

// @Summary Get Kubernetes pods
// @Description Returns list of Kubernetes pods across all connected clusters
// @Tags kubernetes,pods
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /pods [get]
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
			"name":    pod.Name,
			"phase":   string(pod.Status.Phase),
			"node":    pod.Spec.NodeName,
			"ip":      pod.Status.PodIP,
			"created": pod.CreationTimestamp.Format(time.RFC3339),
		})
	}
	response["items"] = items

	// Return JSON response
	json.NewEncoder(ctx).Encode(response)
}

// @Summary Get Kubernetes services
// @Description Returns list of Kubernetes services across all connected clusters
// @Tags kubernetes,services
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /services [get]
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

// @Summary Get Kubernetes nodes
// @Description Returns list of Kubernetes nodes across all connected clusters
// @Tags kubernetes,nodes
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /nodes [get]
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

// @Summary Get Kubernetes clusters information
// @Description Returns information about connected Kubernetes clusters
// @Tags kubernetes,clusters
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /clusters [get]
func (s *apiServer) handleClusters(ctx *fasthttp.RequestCtx) {
	logger := getRequestLogger(ctx)
	method := string(ctx.Method())
	logger.Debug().Str("method", method).Msg("Processing clusters request")

	switch method {
	case "GET":
		// Get list of configured clusters
		clusterCount := s.multiClusterManager.GetClusterCount()

		// Create response object
		clustersList := s.multiClusterManager.GetClusters()
		// Convert slice to map
		clustersMap := make(map[string]ctrl.ClusterConfig)
		for _, cfg := range clustersList {
			clustersMap[cfg.ClusterID] = cfg
		}

		response := struct {
			Count    int                           `json:"count"`
			Clusters map[string]ctrl.ClusterConfig `json:"clusters"`
		}{
			Count:    clusterCount,
			Clusters: clustersMap,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to marshal clusters response")
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Internal server error"}`)
			return
		}

		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(jsonResponse)
		logger.Debug().Int("cluster_count", clusterCount).Msg("Returned cluster list")

	case "POST":
		// Add a new cluster
		var clusterConfig ctrl.ClusterConfig
		if err := json.Unmarshal(ctx.PostBody(), &clusterConfig); err != nil {
			logger.Error().Err(err).Msg("Invalid cluster configuration JSON")
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString(`{"error": "Invalid cluster configuration format"}`)
			return
		}

		// Validate required fields
		if clusterConfig.ClusterID == "" {
			logger.Error().Msg("Missing required cluster_id field")
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString(`{"error": "Missing required cluster_id field"}`)
			return
		}

		// Add the cluster to the manager
		if err := s.multiClusterManager.AddCluster(ctx, clusterConfig); err != nil {
			logger.Error().Err(err).Str("cluster_id", clusterConfig.ClusterID).Msg("Failed to add cluster")
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(fmt.Sprintf(`{"error": "Failed to add cluster: %s"}`, err.Error()))
			return
		}

		logger.Info().Str("cluster_id", clusterConfig.ClusterID).Msg("Added new cluster to manager")
		ctx.SetStatusCode(fasthttp.StatusCreated)
		ctx.SetBodyString(fmt.Sprintf(`{"message": "Cluster %s added successfully"}`, clusterConfig.ClusterID))

	case "DELETE":
		// Get cluster ID from query parameters
		clusterID := string(ctx.QueryArgs().Peek("id"))
		if clusterID == "" {
			logger.Error().Msg("Missing cluster_id parameter")
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString(`{"error": "Missing cluster_id parameter"}`)
			return
		}

		// Remove the cluster from the manager
		s.multiClusterManager.RemoveCluster(clusterID)

		logger.Info().Str("cluster_id", clusterID).Msg("Removed cluster from manager")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString(fmt.Sprintf(`{"message": "Cluster %s removed successfully"}`, clusterID))

	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		ctx.SetBodyString(`{"error": "Method not allowed"}`)
	}
}

// StartAPIServer starts the API server with FastHTTP
func StartAPIServer(ctx context.Context, clientset *kubernetes.Clientset, factory informers.SharedInformerFactory, host string, port int, appConfig *Config) error {
	// Initialize the multi-cluster manager
	multiClusterManager := ctrl.NewMultiClusterManager()

	// Add the current cluster to the manager
	// Use the same kubeconfig path determination logic as in runtime.go
	kubePath := kubeconfig
	if kubePath == "" && appConfig != nil {
		kubePath = appConfig.Kubernetes.Kubeconfig
	}
	log.Debug().Str("kubeconfig", kubePath).Msg("Using kubeconfig file for multi-cluster manager")

	// Check if we're using in-cluster configuration
	inCluster := false
	if appConfig != nil {
		inCluster = appConfig.Kubernetes.InCluster
	}

	currentClusterConfig := ctrl.ClusterConfig{
		Name:       "primary",
		ClusterID:  "primary-cluster",
		KubeConfig: kubePath,
		InCluster:  inCluster, // Use the same setting as the main app
	}

	// Apply leader election settings if configured
	if appConfig != nil && appConfig.ControllerRuntime.LeaderElection.Enabled {
		currentClusterConfig.LeaderElection.Enabled = true
		currentClusterConfig.LeaderElection.Namespace = appConfig.ControllerRuntime.LeaderElection.Namespace
		currentClusterConfig.LeaderElection.ID = appConfig.ControllerRuntime.LeaderElection.ID

		if appConfig.ControllerRuntime.LeaderElection.ID == "" {
			currentClusterConfig.LeaderElection.ID = "k8s-custom-controller-leader-election"
		}

		if appConfig.ControllerRuntime.LeaderElection.Namespace == "" {
			currentClusterConfig.LeaderElection.Namespace = "default"
		}

		log.Debug().Bool("enabled", true).Str("id", currentClusterConfig.LeaderElection.ID).Str("namespace", currentClusterConfig.LeaderElection.Namespace).Msg("Configured leader election")
	}

	// Apply metrics settings if configured
	if appConfig != nil && appConfig.ControllerRuntime.Metrics.BindAddress != "" {
		currentClusterConfig.MetricsBindAddress = appConfig.ControllerRuntime.Metrics.BindAddress
		log.Debug().Str("bind_address", currentClusterConfig.MetricsBindAddress).Msg("Configured metrics server")
	}

	log.Info().Str("cluster_id", currentClusterConfig.ClusterID).Msg("Adding primary cluster to multi-cluster manager")

	err := multiClusterManager.AddCluster(ctx, currentClusterConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add primary cluster to manager")
		return err
	}

	// Start all cluster managers
	go func() {
		log.Info().Msg("Starting multi-cluster manager")
		if err := multiClusterManager.StartAll(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to start multi-cluster manager")
		}
	}()

	// Set up clientset and informer factory for the API server
	server := &apiServer{
		clientset:           clientset,
		informerFactory:     factory,
		config:              appConfig,
		multiClusterManager: multiClusterManager,
		// Rate limiter will be initialized on first request
		requestLimiter: nil,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	log.Info().Str("address", address).Msg("Starting API server")

	// Apply Swagger configuration from app config if available
	if appConfig != nil && appConfig.APIServer.EnableSwagger {
		enableSwagger = appConfig.APIServer.EnableSwagger
		log.Debug().Bool("enable_swagger", enableSwagger).Msg("Swagger configuration applied from config file")
	}

	// Create a server instance with production-ready settings
	fasthttpServer := &fasthttp.Server{
		Handler:            server.requestHandler,
		Name:               "k8s-cli-server", // Using consistent naming as per preferences
		Concurrency:        1000,
		MaxRequestBodySize: 10 * 1024 * 1024, // 10MB max request size
		ReduceMemoryUsage:  true,             // Optimize memory usage
	}

	// Apply security settings from config if available, otherwise use defaults
	if appConfig != nil && appConfig.APIServer.Security.ReadTimeoutSeconds > 0 {
		fasthttpServer.ReadTimeout = time.Duration(appConfig.APIServer.Security.ReadTimeoutSeconds) * time.Second
		fasthttpServer.WriteTimeout = time.Duration(appConfig.APIServer.Security.WriteTimeoutSeconds) * time.Second
		fasthttpServer.IdleTimeout = time.Duration(appConfig.APIServer.Security.IdleTimeoutSeconds) * time.Second
		fasthttpServer.MaxConnsPerIP = appConfig.APIServer.Security.MaxConnsPerIP
		fasthttpServer.DisableKeepalive = appConfig.APIServer.Security.DisableKeepalive
		fasthttpServer.TCPKeepalive = !appConfig.APIServer.Security.DisableKeepalive

		logger := log.Debug()
		logger.Int("read_timeout", appConfig.APIServer.Security.ReadTimeoutSeconds)
		logger.Int("write_timeout", appConfig.APIServer.Security.WriteTimeoutSeconds)
		logger.Int("idle_timeout", appConfig.APIServer.Security.IdleTimeoutSeconds)
		logger.Int("max_conns_per_ip", appConfig.APIServer.Security.MaxConnsPerIP)
		logger.Bool("disable_keepalive", appConfig.APIServer.Security.DisableKeepalive)
		logger.Msg("Applied security settings from configuration")
	} else {
		// Use default security settings
		fasthttpServer.ReadTimeout = 10 * time.Second
		fasthttpServer.WriteTimeout = 30 * time.Second
		fasthttpServer.IdleTimeout = 120 * time.Second
		fasthttpServer.MaxConnsPerIP = 100
		fasthttpServer.DisableKeepalive = false
		fasthttpServer.TCPKeepalive = true
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		log.Info().Msgf("Starting API server on %s:%d", host, port)
		if err := fasthttpServer.ListenAndServe(address); err != nil {
			log.Error().Err(err).Msg("Failed to start API server")
			// Signal the main goroutine that there was an error
			close(sigChan)
		}
	}()

	// Wait for interrupt signal or context cancellation
	select {
	case <-sigChan:
		log.Info().Msg("Received shutdown signal")
	case <-ctx.Done():
		log.Info().Msg("Context canceled, shutting down")
	}

	// Shutdown multi-cluster manager
	log.Info().Msg("Stopping multi-cluster manager")
	multiClusterManager.StopAll(ctx)

	// Shutdown server gracefully
	log.Info().Msg("Shutting down API server")
	if err := fasthttpServer.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error shutting down API server")
		return err
	}

	log.Info().Msg("API server gracefully stopped")
	return nil
}

// handleSwaggerJSON serves the Swagger API documentation JSON
func (s *apiServer) handleSwaggerJSON(ctx *fasthttp.RequestCtx) {
	// Get the logger with request ID
	logger := getRequestLogger(ctx)
	logger.Debug().Msg("Swagger documentation JSON request received")

	// Set proper content type
	ctx.Response.Header.Set("Content-Type", "application/json")

	// Set security headers
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")

	// Add CORS headers if enabled in config
	if s.config != nil && s.config.APIServer.SwaggerUI.CORSEnabled {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	}

	// Get swagger doc
	doc, err := swag.ReadDoc()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read Swagger documentation")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "Failed to read API documentation"})
		return
	}

	// Write swagger JSON
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(doc)
}

// serveSwaggerUI serves Swagger UI HTML page
func (s *apiServer) serveSwaggerUI(ctx *fasthttp.RequestCtx) {
	logger := getRequestLogger(ctx)
	logger.Debug().Msg("Swagger UI request received")

	// Set content type
	ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")

	// Set security headers
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")

	// Add CORS headers if enabled in config
	if s.config != nil && s.config.APIServer.SwaggerUI.CORSEnabled {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	}

	// Simple Swagger UI HTML
	swaggerHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>API Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@latest/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@latest/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      const ui = SwaggerUIBundle({
        url: "/swagger.json",
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
	ctx.WriteString(swaggerHTML)
}

// Global variable for Swagger documentation enablement
var enableSwagger bool

// tokenBucket implements a simple token bucket rate limiting algorithm
type tokenBucket struct {
	tokens      int           // Current number of tokens
	capacity    int           // Maximum number of tokens
	refillRate  int           // Tokens added per second
	lastRefill  time.Time     // Time of last token refill
	mu          sync.Mutex    // Mutex for thread safety
}

// newTokenBucket creates a new token bucket with the given capacity and refill rate
func newTokenBucket(tokensPerSecond int) *tokenBucket {
	return &tokenBucket{
		tokens:     tokensPerSecond, // Start with full bucket
		capacity:   tokensPerSecond, // Capacity equals tokens per second
		refillRate: tokensPerSecond, 
		lastRefill: time.Now(),
	}
}

// take attempts to take a token from the bucket
// Returns true if a token was available and taken, false otherwise
func (tb *tokenBucket) take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed since last refill
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	newTokens := int(elapsed * float64(tb.refillRate))

	if newTokens > 0 {
		tb.tokens = tb.tokens + newTokens
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity // Cap at maximum capacity
		}
		tb.lastRefill = now
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// perIPLimiter manages rate limiters for individual IP addresses
type perIPLimiter struct {
	limiters    map[string]*tokenBucket
	mu          sync.Mutex
	tokensPerSec int
	cleanupInterval time.Duration
}

// newPerIPLimiter creates a new per-IP rate limiter
func newPerIPLimiter(tokensPerSecond int) *perIPLimiter {
	limiter := &perIPLimiter{
		limiters:    make(map[string]*tokenBucket),
		tokensPerSec: tokensPerSecond,
		cleanupInterval: 5 * time.Minute,
	}

	// Start background cleanup
	go limiter.cleanupRoutine()

	return limiter
}

// getLimit returns the current rate limit setting
func (p *perIPLimiter) getLimit() int {
	return p.tokensPerSec
}

// updateLimit updates the rate limit for all IPs
func (p *perIPLimiter) updateLimit(tokensPerSecond int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Update the global setting
	p.tokensPerSec = tokensPerSecond

	// Create new buckets for all existing IPs
	for ip := range p.limiters {
		p.limiters[ip] = newTokenBucket(tokensPerSecond)
	}
}

// allow checks if a request from the given IP is allowed
func (p *perIPLimiter) allow(ip string) bool {
	p.mu.Lock()

	// Create a new limiter for this IP if it doesn't exist
	limiter, exists := p.limiters[ip]
	if !exists {
		limiter = newTokenBucket(p.tokensPerSec)
		p.limiters[ip] = limiter
	}
	p.mu.Unlock()

	// Try to take a token
	return limiter.take()
}

// cleanupRoutine periodically removes unused IP limiters
func (p *perIPLimiter) cleanupRoutine() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		// In a real implementation, we would track last access time
		// and remove limiters that haven't been used recently
		// For now, we just keep the map to a reasonable size
		if len(p.limiters) > 1000 {
			// Create a new map with just the most recently used IPs
			// This is a simple approach; a more sophisticated one would
			// track actual usage timestamps
			p.limiters = make(map[string]*tokenBucket)
		}
		p.mu.Unlock()
	}
}

// checkRateLimit implements a rate limiting mechanism on a per-IP basis
func (s *apiServer) checkRateLimit(clientIP string, logger zerolog.Logger) bool {
	// Initialize rate limiter if not already created
	s.requestLimiterMux.Lock()
	if s.ipLimiter == nil {
		// Default to 10 requests per second, but use config value if available
		rateLimit := 10 // Default value
		if s.config != nil && s.config.APIServer.Security.RateLimitRequestsPerSecond > 0 {
			rateLimit = s.config.APIServer.Security.RateLimitRequestsPerSecond
		}
		s.ipLimiter = newPerIPLimiter(rateLimit)
		logger.Debug().Int("requests_per_second", rateLimit).Msg("Per-IP rate limiter initialized")
	} else if s.config != nil && s.config.APIServer.Security.RateLimitRequestsPerSecond > 0 {
		// Update rate limit if configuration has changed
		currentLimit := s.ipLimiter.getLimit()
		configLimit := s.config.APIServer.Security.RateLimitRequestsPerSecond
		
		if currentLimit != configLimit {
			s.ipLimiter.updateLimit(configLimit)
			logger.Debug().Int("requests_per_second", configLimit).Msg("Rate limit updated")
		}
	}
	s.requestLimiterMux.Unlock()

	// Check if this IP is allowed
	return s.ipLimiter.allow(clientIP)
}

func init() {
	// Add server flags to root command
	rootCmd.PersistentFlags().StringVar(&serverHost, "host", "0.0.0.0", "Host address to bind the server to")
	rootCmd.PersistentFlags().IntVar(&serverPort, "port", 8080, "Port to run the server on")

	// Add flag for Swagger documentation (disabled in production by default)
	rootCmd.PersistentFlags().BoolVar(&enableSwagger, "enable-swagger", false, "Enable Swagger API documentation (development only)")
}
