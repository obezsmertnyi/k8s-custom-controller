package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// TestServerCommandDefined checks if the server command is defined and has the expected properties
func TestServerCommandDefined(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Should be able to get current directory")

	// Go up one directory to the project root
	projectRoot := filepath.Dir(cwd)

	// First check that the command exists
	cmd := exec.Command(filepath.Join(projectRoot, "k8s-cli"), "help")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "Help command should run without errors")
	outputStr := string(output)
	assert.Contains(t, outputStr, "server", "Server command should be defined")

	// Now check the server command properties
	cmd = exec.Command(filepath.Join(projectRoot, "k8s-cli"), "help", "server")
	output, err = cmd.CombinedOutput()
	assert.NoError(t, err, "Server help command should run without errors")
	outputStr = string(output)

	// Check if the command has the expected description
	assert.Contains(t, outputStr, "Start a FastHTTP server", "Server command should have the correct description")

	// Check if the output contains the expected flags
	assert.Contains(t, outputStr, "--host", "Server command should have a host flag")
	assert.Contains(t, outputStr, "--port", "Server command should have a port flag")
	assert.Contains(t, outputStr, "--log-level", "Server command should have a log-level flag")
}
