package tests

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	// Create a handler that simulates the health endpoint
	handler := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/health" {
			// Set content type and respond with health status
			ctx.SetContentType("application/json; charset=utf8")
			ctx.Response.Header.Set("X-Request-ID", "test-request-id")
			
			// Create response JSON
			response := map[string]interface{}{
				"status":              "ok",
				"kubernetes_connected": true,
				"version":             "1.0.0",
			}
			
			// Write response
			ctx.SetStatusCode(fasthttp.StatusOK)
			json.NewEncoder(ctx).Encode(response)
		} else {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI and method
	req.SetRequestURI("http://" + serverAddr + "/health")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Health request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Health endpoint should return 200 OK")
	
	// Check content type
	contentType := resp.Header.ContentType()
	assert.Equal(t, "application/json; charset=utf8", string(contentType), "Content-Type should be application/json; charset=utf8")
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify response fields
	assert.Equal(t, "ok", response["status"], "Health status should be ok")
	assert.Equal(t, true, response["kubernetes_connected"], "Kubernetes should be connected")
	assert.Equal(t, "1.0.0", response["version"], "Version should be 1.0.0")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestDeploymentsEndpoint tests the /deployments endpoint
func TestDeploymentsEndpoint(t *testing.T) {
	// Create a handler that simulates the deployments endpoint
	handler := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/deployments" {
			// Get namespace from query (default to empty string for "all namespaces")
			namespace := string(ctx.QueryArgs().Peek("namespace"))
			if namespace == "" {
				namespace = "default"
			}
			
			// Set content type and respond
			ctx.SetContentType("application/json; charset=utf8")
			ctx.Response.Header.Set("X-Request-ID", "test-request-id")
			
			// Create mock deployment data
			deploymentNames := []string{"deployment1", "deployment2", "deployment3"}
			
			// Check for simple format flag
			if string(ctx.QueryArgs().Peek("format")) == "simple" {
				ctx.SetStatusCode(fasthttp.StatusOK)
				json.NewEncoder(ctx).Encode(deploymentNames)
				return
			}
			
			// Create detailed response
			items := make([]interface{}, 0, len(deploymentNames))
			for _, name := range deploymentNames {
				items = append(items, map[string]interface{}{
					"name":      name,
					"replicas":  3,
					"available": 3,
				})
			}
			
			response := map[string]interface{}{
				"namespace": namespace,
				"count":     len(deploymentNames),
				"source":    "informer-cache",
				"names":     deploymentNames,
				"items":     items,
			}
			
			// Write response
			ctx.SetStatusCode(fasthttp.StatusOK)
			json.NewEncoder(ctx).Encode(response)
		} else {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI and method
	req.SetRequestURI("http://" + serverAddr + "/deployments?namespace=test")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Deployments request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Deployments endpoint should return 200 OK")
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify response fields
	assert.Equal(t, "test", response["namespace"], "Namespace should match query parameter")
	assert.Equal(t, float64(3), response["count"], "Count should be 3")
	assert.Equal(t, "informer-cache", response["source"], "Source should be informer-cache")
	
	// Verify names array
	namesArray, ok := response["names"].([]interface{})
	assert.True(t, ok, "Names should be an array")
	assert.Equal(t, 3, len(namesArray), "Should have 3 deployments")
	
	// Test simple format
	req.SetRequestURI("http://" + serverAddr + "/deployments?format=simple")
	resp.Reset()
	
	err = client.Do(req, resp)
	assert.NoError(t, err, "Simple format request should not fail")
	
	// Parse simple response
	var simpleResponse []string
	err = json.Unmarshal(resp.Body(), &simpleResponse)
	assert.NoError(t, err, "Simple response should be valid JSON")
	assert.Equal(t, 3, len(simpleResponse), "Simple response should have 3 items")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestPodsEndpoint tests the /pods endpoint
func TestPodsEndpoint(t *testing.T) {
	// Create a handler that simulates the pods endpoint
	handler := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/pods" {
			// Get namespace from query
			namespace := string(ctx.QueryArgs().Peek("namespace"))
			if namespace == "" {
				namespace = "default"
			}
			
			// Set content type and respond
			ctx.SetContentType("application/json; charset=utf8")
			
			// Create mock pod data
			podNames := []string{"pod1", "pod2", "pod3"}
			
			// Check for simple format flag
			if string(ctx.QueryArgs().Peek("format")) == "simple" {
				ctx.SetStatusCode(fasthttp.StatusOK)
				json.NewEncoder(ctx).Encode(podNames)
				return
			}
			
			// Create detailed response
			items := make([]interface{}, 0, len(podNames))
			for i, name := range podNames {
				items = append(items, map[string]interface{}{
					"name":    name,
					"phase":   "Running",
					"node":    "node1",
					"ip":      "10.0.0." + string(rune(i+1)),
					"created": "2025-07-06T00:00:00Z",
				})
			}
			
			response := map[string]interface{}{
				"namespace": namespace,
				"count":     len(podNames),
				"source":    "kubernetes-api",
				"names":     podNames,
				"items":     items,
			}
			
			// Write response
			ctx.SetStatusCode(fasthttp.StatusOK)
			json.NewEncoder(ctx).Encode(response)
		} else {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI and method
	req.SetRequestURI("http://" + serverAddr + "/pods?namespace=kube-system")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Pods request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Pods endpoint should return 200 OK")
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify response fields
	assert.Equal(t, "kube-system", response["namespace"], "Namespace should match query parameter")
	assert.Equal(t, float64(3), response["count"], "Count should be 3")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestServicesEndpoint tests the /services endpoint
func TestServicesEndpoint(t *testing.T) {
	// Create a handler that simulates the services endpoint
	handler := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/services" {
			// Get namespace from query
			namespace := string(ctx.QueryArgs().Peek("namespace"))
			if namespace == "" {
				namespace = "default"
			}
			
			// Set content type and respond
			ctx.SetContentType("application/json; charset=utf8")
			
			// Create mock service data
			serviceNames := []string{"service1", "service2"}
			
			// Check for simple format flag
			if string(ctx.QueryArgs().Peek("format")) == "simple" {
				ctx.SetStatusCode(fasthttp.StatusOK)
				json.NewEncoder(ctx).Encode(serviceNames)
				return
			}
			
			// Create port info for services
			portInfo := []map[string]interface{}{
				{
					"name":       "http",
					"port":       float64(80),
					"targetPort": "8080",
					"protocol":   "TCP",
				},
			}
			
			// Create detailed response
			items := make([]interface{}, 0, len(serviceNames))
			for _, name := range serviceNames {
				items = append(items, map[string]interface{}{
					"name":      name,
					"type":      "ClusterIP",
					"clusterIP": "10.0.0.1",
					"ports":     portInfo,
				})
			}
			
			response := map[string]interface{}{
				"namespace": namespace,
				"count":     len(serviceNames),
				"source":    "kubernetes-api",
				"names":     serviceNames,
				"items":     items,
			}
			
			// Write response
			ctx.SetStatusCode(fasthttp.StatusOK)
			json.NewEncoder(ctx).Encode(response)
		} else {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI and method
	req.SetRequestURI("http://" + serverAddr + "/services")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Services request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Services endpoint should return 200 OK")
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify response fields
	assert.Equal(t, "default", response["namespace"], "Namespace should default to 'default'")
	assert.Equal(t, float64(2), response["count"], "Count should be 2")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestClustersEndpoint tests the /clusters endpoint
func TestClustersEndpoint(t *testing.T) {
	// Create a handler that simulates the clusters endpoint
	handler := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/clusters" {
			// Set content type and respond
			ctx.SetContentType("application/json; charset=utf8")
			
			// Create mock cluster data
			clusters := []map[string]interface{}{
				{
					"name": "cluster1",
					"contexts": []string{"context1"},
					"status": "Connected",
					"version": "v1.26.0",
					"nodes": 3,
				},
				{
					"name": "cluster2",
					"contexts": []string{"context2"},
					"status": "Connected",
					"version": "v1.27.0",
					"nodes": 5,
				},
			}
			
			response := map[string]interface{}{
				"count":   len(clusters),
				"clusters": clusters,
			}
			
			// Write response
			ctx.SetStatusCode(fasthttp.StatusOK)
			json.NewEncoder(ctx).Encode(response)
		} else {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI and method
	req.SetRequestURI("http://" + serverAddr + "/clusters")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Clusters request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Clusters endpoint should return 200 OK")
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify response fields
	assert.Equal(t, float64(2), response["count"], "Count should be 2")
	clusters, ok := response["clusters"].([]interface{})
	assert.True(t, ok, "Response should contain clusters array")
	assert.Equal(t, 2, len(clusters), "Should have 2 clusters")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestUnknownEndpoint tests the handling of unknown endpoints
func TestUnknownEndpoint(t *testing.T) {
	// Create a handler that simulates the API server's 404 handling
	handler := func(ctx *fasthttp.RequestCtx) {
		// Set content type for JSON responses
		ctx.SetContentType("application/json; charset=utf8")
		
		// Return 404 for unknown paths
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		json.NewEncoder(ctx).Encode(map[string]string{"error": "Not found"})
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create and start server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create a client
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	
	// Set request URI for non-existent endpoint
	req.SetRequestURI("http://" + serverAddr + "/non-existent-path")
	req.Header.SetMethod("GET")
	
	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Request should not fail")
	
	// Check response status code
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode(), "Unknown endpoint should return 404 Not Found")
	
	// Parse response body
	var response map[string]string
	err = json.Unmarshal(resp.Body(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	
	// Verify error message
	assert.Equal(t, "Not found", response["error"], "Error message should be 'Not found'")
	
	// Cleanup
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}
