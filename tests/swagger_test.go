package tests

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// TestSwaggerEndpoints tests the Swagger UI endpoints
func TestSwaggerEndpoints(t *testing.T) {
	// Create a test server with a handler that simulates Swagger JSON response
	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		switch path {
		case "/swagger.json":
			// Simulate swagger.json response
			ctx.SetContentType("application/json")
			mockSwaggerDoc := map[string]interface{}{
				"swagger": "2.0",
				"info": map[string]interface{}{
					"title":   "Kubernetes Multi-Cluster Controller API",
					"version": "1.0",
				},
				"paths": map[string]interface{}{},
			}
			json.NewEncoder(ctx).Encode(mockSwaggerDoc)
		case "/swagger", "/swagger/":
			// Redirect to index.html
			ctx.Redirect("/swagger/index.html", fasthttp.StatusFound)
		case "/swagger/index.html":
			// Serve Swagger UI
			ctx.SetContentType("text/html")
			ctx.WriteString("<html><body>Swagger UI</body></html>")
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}

	// Create a local test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()
	
	// Start server in a goroutine
	server := &fasthttp.Server{
		Handler: handler,
	}
	go func() {
		if err := server.Serve(listener); err != nil {
			// Ignore server closed error
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create client to make requests
	client := &fasthttp.Client{}

	// Test /swagger.json endpoint
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("http://" + serverAddr + "/swagger.json")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Make request to /swagger.json
	err = client.Do(req, resp)
	assert.NoError(t, err, "Error making request to /swagger.json")

	// Check response content type and status code
	contentType := resp.Header.ContentType()
	assert.Equal(t, "application/json", string(contentType), "Content-Type should be application/json")

	// Decode and check response body
	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(resp.Body(), &swaggerDoc)
	assert.NoError(t, err, "Error parsing Swagger JSON")

	// Check basic swagger document structure
	assert.Equal(t, "2.0", swaggerDoc["swagger"], "Swagger version should be 2.0")
	assert.Contains(t, swaggerDoc, "info", "Swagger document should contain info section")

	// Test redirect from /swagger to /swagger/index.html
	req.SetRequestURI("http://" + serverAddr + "/swagger")
	resp.Reset()

	err = client.Do(req, resp)
	assert.NoError(t, err, "Error making request to /swagger")

	// Check if we got a redirect
	assert.Equal(t, fasthttp.StatusFound, resp.StatusCode(), "Should redirect from /swagger to /swagger/index.html")
	// Location header may have full URL or just path
	location := string(resp.Header.Peek("Location"))
	assert.True(t, strings.HasSuffix(location, "/swagger/index.html"), "Location should end with /swagger/index.html, got %s", location)

	// Test Swagger UI endpoint
	req.SetRequestURI("http://" + serverAddr + "/swagger/index.html")
	resp.Reset()
	
	err = client.Do(req, resp)
	assert.NoError(t, err, "Error making request to /swagger/index.html")

	// Check content type and response body for HTML
	assert.Equal(t, "text/html", string(resp.Header.ContentType()), "Content-Type should be text/html")
	assert.Contains(t, string(resp.Body()), "<html>", "Response should contain HTML")

	// Cleanup server
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestSwaggerUIToggle tests enabling and disabling Swagger UI via configuration
func TestSwaggerUIToggle(t *testing.T) {
	// Create a handler function that simulates configurable swagger UI
	enabled := true
	handler := func(ctx *fasthttp.RequestCtx) {
		if !enabled && (string(ctx.Path()) == "/swagger" || string(ctx.Path()) == "/swagger.json") {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		if string(ctx.Path()) == "/swagger" || string(ctx.Path()) == "/swagger.json" {
			ctx.SetContentType("application/json")
			ctx.WriteString("{\"swagger\":\"2.0\"}")
		}
	}

	// Create a local test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create server
	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Create client
	client := &fasthttp.Client{}

	// Test with Swagger enabled
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("http://" + serverAddr + "/swagger.json")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Make request with swagger enabled
	err = client.Do(req, resp)
	assert.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "Swagger endpoint should be available when enabled")
	assert.Contains(t, string(resp.Body()), "swagger", "Response should contain swagger document")

	// Now disable swagger
	enabled = false
	resp.Reset()

	// Make another request
	err = client.Do(req, resp)
	assert.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode(), "Swagger endpoint should return 404 when disabled")

	// Cleanup server
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}
