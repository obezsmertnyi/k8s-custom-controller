package testutil

import (
	"context"
	"sync"
	"testing"
)

// StartTestManager creates a context for testing and returns a cancel function
func StartTestManager(t *testing.T) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// No need to actually start a manager for tests
	return ctx, cancel
}

// MockManager is a simple mock for test purposes
type MockManager struct {
	Started bool
	Stopped bool
	mu      sync.Mutex
}

// NewMockManager creates a new mock manager for testing
func NewMockManager() *MockManager {
	return &MockManager{}
}

// Start mocks starting the manager
func (m *MockManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Started = true
	
	<-ctx.Done()
	
	m.mu.Lock()
	m.Stopped = true
	m.mu.Unlock()
	
	return nil
}

// IsStarted returns whether the manager has been started
func (m *MockManager) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Started
}

// IsStopped returns whether the manager has been stopped
func (m *MockManager) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Stopped
}
