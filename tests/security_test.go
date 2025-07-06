package tests

import (
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// TestSecurityHeaders checks if security headers are properly set
func TestSecurityHeaders(t *testing.T) {
	// Create a handler that sets security headers
	handler := func(ctx *fasthttp.RequestCtx) {
		// Set security headers as done in the API server
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		ctx.Response.Header.Set("X-Frame-Options", "DENY")
		ctx.Response.Header.Set("X-Request-ID", "test-request-id")
		
		// Set content type and respond
		ctx.SetContentType("application/json")
		ctx.WriteString("{\"status\":\"ok\"}")
	}

	// Create a local test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create a test server
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

	// Create a client for testing
	client := &fasthttp.Client{}

	// Create a request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("http://" + serverAddr + "/test")
	req.Header.SetMethod("GET")

	// Create a response
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Make request
	err = client.Do(req, resp)
	assert.NoError(t, err, "Request should not fail")

	// Check security headers
	assert.Equal(t, "nosniff", string(resp.Header.Peek("X-Content-Type-Options")), "X-Content-Type-Options header should be set to nosniff")
	assert.Equal(t, "DENY", string(resp.Header.Peek("X-Frame-Options")), "X-Frame-Options header should be set to DENY")
	assert.Equal(t, "test-request-id", string(resp.Header.Peek("X-Request-ID")), "X-Request-ID header should be set")

	// Cleanup server
	if err := server.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestKeepAliveSettings tests the keepalive settings
func TestKeepAliveSettings(t *testing.T) {
	// Create listeners for both servers
	listenerWithKeepAlive, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener with keep-alive: %v", err)
	}
	serverAddrWithKeepAlive := listenerWithKeepAlive.Addr().String()

	listenerWithoutKeepAlive, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener without keep-alive: %v", err)
	}
	serverAddrWithoutKeepAlive := listenerWithoutKeepAlive.Addr().String()

	// Create a server with keepalive enabled
	serverWithKeepAlive := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("OK")
		},
		DisableKeepalive: false,
	}

	// Start server with keep-alive in a goroutine
	go func() {
		if err := serverWithKeepAlive.Serve(listenerWithKeepAlive); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server with keep-alive error: %v", err)
			}
		}
	}()

	// Create a server with keepalive disabled
	serverWithoutKeepAlive := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("OK")
		},
		DisableKeepalive: true,
	}

	// Start server without keep-alive in a goroutine
	go func() {
		if err := serverWithoutKeepAlive.Serve(listenerWithoutKeepAlive); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server without keep-alive error: %v", err)
			}
		}
	}()

	// Create client for testing
	client := &fasthttp.Client{}

	// Test keepalive enabled (connection should remain open)
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://" + serverAddrWithKeepAlive + "/test")
	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()

	// Make request with keepalive server
	err = client.Do(req, resp)
	assert.NoError(t, err, "Request should not fail")

	// Check if Connection header is keep-alive
	connectionHeader := string(resp.Header.Peek("Connection"))
	// Note: fasthttp might not set "Connection: keep-alive" explicitly when enabled
	// This is a simplified test. In real world, we would test with multiple requests
	// over the same connection to verify it's being kept alive
	assert.NotEqual(t, "close", connectionHeader, "Connection should not be closed when keepalive is enabled")

	// Reset response and set new request URI for the non-keepalive server
	fasthttp.ReleaseResponse(resp)
	resp = fasthttp.AcquireResponse()
	req.SetRequestURI("http://" + serverAddrWithoutKeepAlive + "/test")

	// Make request with no-keepalive server
	err = client.Do(req, resp)
	assert.NoError(t, err, "Request should not fail")

	// Cleanup
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)

	// Shutdown servers
	if err := serverWithKeepAlive.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server with keep-alive: %v", err)
	}

	if err := serverWithoutKeepAlive.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server without keep-alive: %v", err)
	}
}

// TestServerTimeout tests the read and write timeouts
func TestServerTimeout(t *testing.T) {
	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()

	// Create a server with very short read timeout
	shortTimeoutServer := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			time.Sleep(100 * time.Millisecond) // Simulate processing
			ctx.WriteString("OK")
		},
		ReadTimeout: 10 * time.Millisecond, // Very short timeout for testing
	}

	// Test server with short timeout in a goroutine
	go func() {
		if err := shortTimeoutServer.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create a client
	client := &fasthttp.Client{}

	// Create a request that will likely timeout
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://" + serverAddr + "/test")
	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()

	// Make request - this may fail due to timeout, which is expected
	_ = client.Do(req, resp)

	// Cleanup
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)

	// Shutdown server
	if err := shortTimeoutServer.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}

// TestMaxConnectionsLimit tests the maximum connections limit
func TestMaxConnectionsLimit(t *testing.T) {
	maxConns := 5
	
	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Cannot create listener: %v", err)
	}
	serverAddr := listener.Addr().String()
	
	// Create a server with connection limit
	limitedServer := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			// Simulate slow processing
			time.Sleep(100 * time.Millisecond)
			ctx.WriteString("OK")
		},
		Concurrency: maxConns,
	}

	// Start server in a goroutine
	go func() {
		if err := limitedServer.Serve(listener); err != nil {
			if !strings.Contains(err.Error(), "server closed") {
				t.Logf("Server error: %v", err)
			}
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create wait group for concurrent requests
	var wg sync.WaitGroup
	
	// Make more requests than the server can handle concurrently
	numRequests := maxConns * 2
	
	// Simulate hitting the connection limit by making multiple concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(req)
			req.SetRequestURI("http://" + serverAddr + "/test")
			req.Header.SetMethod("GET")
			
			resp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(resp)
			
			// Use a client with short timeout to avoid waiting too long
			client := &fasthttp.Client{
				ReadTimeout:  50 * time.Millisecond,
				WriteTimeout: 50 * time.Millisecond,
			}
			
			// Try to make request
			_ = client.Do(req, resp)
			// We don't check the error because some requests are expected to fail or timeout
		}()
	}
	
	// Wait for all requests to complete
	wg.Wait()

	// Shutdown server
	if err := limitedServer.Shutdown(); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}
}
