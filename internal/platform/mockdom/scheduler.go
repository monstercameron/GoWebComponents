package mockdom

import (
	goRuntime "runtime"
	"sync"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// MockDeadline implements runtime.Deadline
type MockDeadline struct {
	timeRemaining float64
	didTimeout    bool
}

var _ runtime.Deadline = (*MockDeadline)(nil)

func (parseD *MockDeadline) TimeRemaining() float64 {
	return parseD.timeRemaining
}

func (parseD *MockDeadline) DidTimeout() bool {
	return parseD.didTimeout
}

// MockScheduler implements runtime.Scheduler for testing
type MockScheduler struct {
	mu               sync.Mutex
	pendingCallbacks []func(runtime.Deadline)
	timeouts         []func()
	synchronous      bool // If true, execute callbacks immediately
}

var _ runtime.Scheduler = (*MockScheduler)(nil)

// NewMockScheduler creates a MockScheduler; when synchronous is true callbacks are executed inline.
func NewMockScheduler(isSynchronous bool) *MockScheduler {
	return &MockScheduler{
		pendingCallbacks: make([]func(runtime.Deadline), 0),
		timeouts:         make([]func(), 0),
		synchronous:      isSynchronous,
	}
}

func (parseS *MockScheduler) RequestIdleCallback(parseCallback func(deadline runtime.Deadline)) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()

	if parseS.synchronous {
		// Execute immediately for deterministic testing
		parseDeadline := &MockDeadline{
			timeRemaining: 16.0, // Simulate 16ms available
			didTimeout:    false,
		}
		parseCallback(parseDeadline)
	} else {
		parseS.pendingCallbacks = append(parseS.pendingCallbacks, parseCallback)
	}
}

func (parseS *MockScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()

	if parseS.synchronous {
		parseCallback()
	} else {
		parseS.timeouts = append(parseS.timeouts, parseCallback)
	}
}

// FlushIdleCallbacks executes all pending idle callbacks
func (parseS *MockScheduler) FlushIdleCallbacks() {
	parseS.mu.Lock()
	parseCallbacks := parseS.pendingCallbacks
	parseS.pendingCallbacks = make([]func(runtime.Deadline), 0)
	parseS.mu.Unlock()

	for _, parseCb := range parseCallbacks {
		parseDeadline := &MockDeadline{
			timeRemaining: 16.0,
			didTimeout:    false,
		}
		parseCb(parseDeadline)
	}
}

// FlushTimeouts executes all pending timeouts
func (parseS *MockScheduler) FlushTimeouts() {
	parseS.mu.Lock()
	parseTimeouts := parseS.timeouts
	parseS.timeouts = make([]func(), 0)
	parseS.mu.Unlock()

	for _, parseCb := range parseTimeouts {
		parseCb()
	}
}

// FlushAll executes all queued idle callbacks and timeouts until the scheduler settles.
func (parseS *MockScheduler) FlushAll() {
	parseIdlePasses := 0
	for {
		parseS.FlushIdleCallbacks()
		parseS.FlushTimeouts()

		// Let goroutine-driven loaders enqueue follow-up callbacks before we
		// decide the scheduler is truly settled.
		goRuntime.Gosched()

		if parseS.GetPendingCount() == 0 && parseS.GetPendingTimeoutCount() == 0 {
			parseIdlePasses++
			if parseIdlePasses >= 2 {
				return
			}
			continue
		}
		parseIdlePasses = 0
	}
}

// GetPendingCount returns the number of pending callbacks
func (parseS *MockScheduler) GetPendingCount() int {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return len(parseS.pendingCallbacks)
}

// GetPendingTimeoutCount returns the number of queued timeouts.
func (parseS *MockScheduler) GetPendingTimeoutCount() int {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	return len(parseS.timeouts)
}
