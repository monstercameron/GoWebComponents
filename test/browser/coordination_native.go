//go:build !js || !wasm

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

var nativeCoordinationHarnessFatal = func(parseTb testing.TB) {
	parseTb.Helper()
	parseTb.Fatal("test/browser NewCoordinationHarness requires js/wasm tests")
}

// NewCoordinationHarness requires a js/wasm test environment.
func NewCoordinationHarness(parseTb testing.TB, parseOptions ...Options) *CoordinationHarness {
	parseTb.Helper()
	nativeCoordinationHarnessFatal(parseTb)
	return nil
}

// Environment returns the underlying browser environment for advanced setup.
func (parseH *CoordinationHarness) Environment() *Environment { return nil }

// OpenTab opens one additional mock browser tab and returns that tab handle.
func (parseH *CoordinationHarness) OpenTab(parsePath string, parseName string) *MockWindow {
	return nil
}

// Workers returns all currently created worker handles.
func (parseH *CoordinationHarness) Workers() []*MockWorker { return nil }

// Worker returns one worker handle by index.
func (parseH *CoordinationHarness) Worker(parseIndex int) *MockWorker { return nil }

// CrossTabMessages returns broadcast messages, optionally filtered by channel name.
func (parseH *CoordinationHarness) CrossTabMessages(parseChannel string) []BroadcastMessage {
	return nil
}

// SetOffline sets the offline state used by retry and reconnect flows.
func (parseH *CoordinationHarness) SetOffline(isOffline bool) {}

// IsOffline reports whether offline mode is active.
func (parseH *CoordinationHarness) IsOffline() bool { return false }

// QueueRetry stores one retryable background operation.
func (parseH *CoordinationHarness) QueueRetry(parseId string, parseMaxAttempts int) {}

// FailRetry records one retry failure and keeps the entry pending while attempts remain.
func (parseH *CoordinationHarness) FailRetry(parseId string, parseErr string) {}

// ResolveRetry marks one queued retry as completed.
func (parseH *CoordinationHarness) ResolveRetry(parseId string) {}

// RetryEntries returns one stable snapshot of queued retry entries.
func (parseH *CoordinationHarness) RetryEntries() []RetryEntry { return nil }

// RecordReconnectFailure records one reconnect failure and increments attempts.
func (parseH *CoordinationHarness) RecordReconnectFailure(parseErr string) {}

// RecordReconnectSuccess records one successful reconnect.
func (parseH *CoordinationHarness) RecordReconnectSuccess() {}

// Reconnect returns one snapshot of reconnect state.
func (parseH *CoordinationHarness) Reconnect() ReconnectSnapshot { return ReconnectSnapshot{} }
