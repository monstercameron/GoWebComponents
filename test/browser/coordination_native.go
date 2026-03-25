//go:build !js || !wasm
// +build !js !wasm

package browser

import "testing"

// RetryEntry captures one queued retry flow in the coordination harness.
type RetryEntry struct {
	ID          string
	Attempts    int
	MaxAttempts int
	LastError   string
	Pending     bool
}

// ReconnectSnapshot captures reconnect progress in the coordination harness.
type ReconnectSnapshot struct {
	IsConnected bool
	Attempts    int
	LastError   string
}

// CoordinationHarness provides deterministic cross-tab, worker, and offline controls.
type CoordinationHarness struct{}

// NewCoordinationHarness requires a js/wasm test environment.
func NewCoordinationHarness(tb testing.TB, options ...Options) *CoordinationHarness {
	tb.Helper()
	tb.Fatal("test/browser NewCoordinationHarness requires js/wasm tests")
	return nil
}

// Environment returns the underlying browser environment for advanced setup.
func (h *CoordinationHarness) Environment() *Environment { return nil }

// OpenTab opens one additional mock browser tab and returns that tab handle.
func (h *CoordinationHarness) OpenTab(path string, name string) *MockWindow { return nil }

// Workers returns all currently created worker handles.
func (h *CoordinationHarness) Workers() []*MockWorker { return nil }

// Worker returns one worker handle by index.
func (h *CoordinationHarness) Worker(index int) *MockWorker { return nil }

// CrossTabMessages returns broadcast messages, optionally filtered by channel name.
func (h *CoordinationHarness) CrossTabMessages(channel string) []BroadcastMessage { return nil }

// SetOffline sets the offline state used by retry and reconnect flows.
func (h *CoordinationHarness) SetOffline(isOffline bool) {}

// IsOffline reports whether offline mode is active.
func (h *CoordinationHarness) IsOffline() bool { return false }

// QueueRetry stores one retryable background operation.
func (h *CoordinationHarness) QueueRetry(id string, maxAttempts int) {}

// FailRetry records one retry failure and keeps the entry pending while attempts remain.
func (h *CoordinationHarness) FailRetry(id string, err string) {}

// ResolveRetry marks one queued retry as completed.
func (h *CoordinationHarness) ResolveRetry(id string) {}

// RetryEntries returns one stable snapshot of queued retry entries.
func (h *CoordinationHarness) RetryEntries() []RetryEntry { return nil }

// RecordReconnectFailure records one reconnect failure and increments attempts.
func (h *CoordinationHarness) RecordReconnectFailure(err string) {}

// RecordReconnectSuccess records one successful reconnect.
func (h *CoordinationHarness) RecordReconnectSuccess() {}

// Reconnect returns one snapshot of reconnect state.
func (h *CoordinationHarness) Reconnect() ReconnectSnapshot { return ReconnectSnapshot{} }
