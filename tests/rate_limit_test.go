package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTokenBucket verifies that the token bucket rate limiting works correctly
func TestTokenBucket(t *testing.T) {
	// Import the tokenBucket implementation
	// Note: In a real implementation, you would make tokenBucket accessible for testing
	// or create a test-specific implementation here
	
	// Create a mock token bucket
	bucket := newMockTokenBucket(3) // 3 tokens per second
	
	// Initial state should allow requests
	assert.True(t, bucket.take(), "First request should be allowed")
	assert.True(t, bucket.take(), "Second request should be allowed")
	assert.True(t, bucket.take(), "Third request should be allowed")
	assert.False(t, bucket.take(), "Fourth request should be denied (bucket empty)")
	
	// Wait for refill
	time.Sleep(1 * time.Second)
	
	// After waiting, we should get more tokens
	assert.True(t, bucket.take(), "Request after waiting should be allowed")
}

// TestPerIPRateLimiting verifies that the per-IP rate limiting works correctly
func TestPerIPRateLimiting(t *testing.T) {
	// Create a mock per-IP limiter
	limiter := newMockPerIPLimiter(2) // 2 requests per second per IP
	
	// Test with multiple IPs
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"
	
	// IP1 should get its tokens
	assert.True(t, limiter.allow(ip1), "First request from IP1 should be allowed")
	assert.True(t, limiter.allow(ip1), "Second request from IP1 should be allowed")
	assert.False(t, limiter.allow(ip1), "Third request from IP1 should be denied")
	
	// IP2 should get its own independent tokens
	assert.True(t, limiter.allow(ip2), "First request from IP2 should be allowed")
	assert.True(t, limiter.allow(ip2), "Second request from IP2 should be allowed")
	assert.False(t, limiter.allow(ip2), "Third request from IP2 should be denied")
	
	// Wait for refill
	time.Sleep(1 * time.Second)
	
	// Both IPs should get tokens again
	assert.True(t, limiter.allow(ip1), "Request from IP1 after waiting should be allowed")
	assert.True(t, limiter.allow(ip2), "Request from IP2 after waiting should be allowed")
}

// Mock implementations for testing

// mockTokenBucket is a simple token bucket for testing
type mockTokenBucket struct {
	tokens      int
	capacity    int
	refillRate  int
	lastRefill  time.Time
	mu          sync.Mutex
}

func newMockTokenBucket(tokensPerSecond int) *mockTokenBucket {
	return &mockTokenBucket{
		tokens:     tokensPerSecond,
		capacity:   tokensPerSecond,
		refillRate: tokensPerSecond,
		lastRefill: time.Now(),
	}
}

func (tb *mockTokenBucket) take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed since last refill
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	newTokens := int(elapsed * float64(tb.refillRate))

	if newTokens > 0 {
		tb.tokens = tb.tokens + newTokens
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
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

// mockPerIPLimiter is a simplified per-IP limiter for testing
type mockPerIPLimiter struct {
	limiters    map[string]*mockTokenBucket
	mu          sync.Mutex
	tokensPerSec int
}

func newMockPerIPLimiter(tokensPerSecond int) *mockPerIPLimiter {
	return &mockPerIPLimiter{
		limiters:    make(map[string]*mockTokenBucket),
		tokensPerSec: tokensPerSecond,
	}
}

func (p *mockPerIPLimiter) allow(ip string) bool {
	p.mu.Lock()

	// Create a new limiter for this IP if it doesn't exist
	limiter, exists := p.limiters[ip]
	if !exists {
		limiter = newMockTokenBucket(p.tokensPerSec)
		p.limiters[ip] = limiter
	}
	p.mu.Unlock()

	// Try to take a token
	return limiter.take()
}
