package mockdom

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// MockDeadline implements runtime.Deadline
type MockDeadline struct {
	timeRemaining float64
	didTimeout    bool
}

var _ runtime.Deadline = (*MockDeadline)(nil)

func (d *MockDeadline) TimeRemaining() float64 {
	return d.timeRemaining
}

func (d *MockDeadline) DidTimeout() bool {
	return d.didTimeout
}

// MockScheduler implements runtime.Scheduler for testing
type MockScheduler struct {
	mu               sync.Mutex
	pendingCallbacks []func(runtime.Deadline)
	timeouts         []func()
	synchronous      bool // If true, execute callbacks immediately
}

var _ runtime.Scheduler = (*MockScheduler)(nil)

func NewMockScheduler(synchronous bool) *MockScheduler {
	return &MockScheduler{
		pendingCallbacks: make([]func(runtime.Deadline), 0),
		timeouts:         make([]func(), 0),
		synchronous:      synchronous,
	}
}

func (s *MockScheduler) RequestIdleCallback(callback func(deadline runtime.Deadline)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.synchronous {
		// Execute immediately for deterministic testing
		deadline := &MockDeadline{
			timeRemaining: 16.0, // Simulate 16ms available
			didTimeout:    false,
		}
		callback(deadline)
	} else {
		s.pendingCallbacks = append(s.pendingCallbacks, callback)
	}
}

func (s *MockScheduler) SetTimeout(callback func(), delay int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.synchronous {
		callback()
	} else {
		s.timeouts = append(s.timeouts, callback)
	}
}

// FlushIdleCallbacks executes all pending idle callbacks
func (s *MockScheduler) FlushIdleCallbacks() {
	s.mu.Lock()
	callbacks := s.pendingCallbacks
	s.pendingCallbacks = make([]func(runtime.Deadline), 0)
	s.mu.Unlock()

	for _, cb := range callbacks {
		deadline := &MockDeadline{
			timeRemaining: 16.0,
			didTimeout:    false,
		}
		cb(deadline)
	}
}

// FlushTimeouts executes all pending timeouts
func (s *MockScheduler) FlushTimeouts() {
	s.mu.Lock()
	timeouts := s.timeouts
	s.timeouts = make([]func(), 0)
	s.mu.Unlock()

	for _, cb := range timeouts {
		cb()
	}
}

// GetPendingCount returns the number of pending callbacks
func (s *MockScheduler) GetPendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pendingCallbacks)
}
