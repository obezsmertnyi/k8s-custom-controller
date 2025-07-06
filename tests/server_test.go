package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// Test version of request handler for testing
func testRequestHandler(ctx *fasthttp.RequestCtx) {
	// Set content type and respond
	ctx.SetContentType("text/plain; charset=utf8")
	fmt.Fprintf(ctx, "Hello from FastHTTP! You requested: %s from IP: %s", ctx.Path(), ctx.RemoteIP())
}

// TestServerRequestHandler tests the HTTP request handler functionality
func TestServerRequestHandler(t *testing.T) {
	// Create a test request context
	ctx := &fasthttp.RequestCtx{}
	
	// Set test request data
	ctx.Request.SetRequestURI("/test-path")
	ctx.Request.Header.SetMethod("GET")
	
	// Call the test request handler
	testRequestHandler(ctx)
	
	// Check the response
	responseBody := string(ctx.Response.Body())
	assert.Contains(t, responseBody, "You requested: /test-path", "Handler should include the requested path in response")
	assert.Equal(t, "text/plain; charset=utf8", string(ctx.Response.Header.ContentType()), "Content type should be set correctly")
}

// TestServerGracefulShutdown verifies proper server shutdown
func TestServerGracefulShutdown(t *testing.T) {
	// Create a test server
	server := &fasthttp.Server{
		Handler: testRequestHandler,
		Name:    "test-server",
	}
	
	// Start the server in a goroutine
	go func() {
		err := server.ListenAndServe(":0") // Use a random port
		// In a real server we would check the error, but in the test we expect an error when closing
		_ = err
	}()
	
	// Give the server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Shutdown the server
	err := server.Shutdown()
	assert.NoError(t, err, "Server should shutdown gracefully")
}

// TestCliServerFlags checks if the k8s-cli command has the expected server-related flags and properties
func TestCliServerFlags(t *testing.T) {
	// Get mock CLI command
	cliCmd := MockCliCommand()
	
	// Verify CLI command exists
	assert.NotNil(t, cliCmd, "CLI command should be defined")
	
	// Verify CLI command name
	assert.Equal(t, "k8s-cli", cliCmd.Use, "CLI command should be named 'k8s-cli'")
	
	// Verify server-related flags
	portFlag := cliCmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag, "CLI command should have a port flag")
	
	hostFlag := cliCmd.Flags().Lookup("host")
	assert.NotNil(t, hostFlag, "CLI command should have a host flag")
	
	swaggerFlag := cliCmd.Flags().Lookup("enable-swagger")
	assert.NotNil(t, swaggerFlag, "CLI command should have an enable-swagger flag")
	
	// Verify default values
	assert.Equal(t, "0.0.0.0", hostFlag.DefValue, "Default host should be 0.0.0.0")
	assert.Equal(t, "8080", portFlag.DefValue, "Default port should be 8080")
	assert.Equal(t, "true", swaggerFlag.DefValue, "Default Swagger UI state should be enabled")
}
