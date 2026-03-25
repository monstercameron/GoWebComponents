//go:build js && wasm
// +build js,wasm

package browser

import (
	"strings"
	"sync"
	"testing"
)

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
type CoordinationHarness struct {
	tb  testing.TB
	env *Environment

	mu             sync.Mutex
	isOffline      bool
	storeRetryByID map[string]RetryEntry
	storeRetryIDs  []string
	storeReconnect ReconnectSnapshot
}

// NewCoordinationHarness creates one coordination harness over the browser environment.
func NewCoordinationHarness(tb testing.TB, options ...Options) *CoordinationHarness {
	tb.Helper()
	harness := &CoordinationHarness{
		tb:             tb,
		env:            Install(tb, options...),
		storeRetryByID: map[string]RetryEntry{},
	}
	return harness
}

// Environment returns the underlying browser environment for advanced setup.
func (h *CoordinationHarness) Environment() *Environment {
	if h == nil {
		return nil
	}
	return h.env
}

// OpenTab opens one additional mock browser tab and returns that tab handle.
func (h *CoordinationHarness) OpenTab(path string, name string) *MockWindow {
	if h == nil || h.env == nil {
		return nil
	}
	parsePath := strings.TrimSpace(path)
	if parsePath == "" {
		parsePath = "/"
	}
	h.env.Window().Call("open", parsePath, strings.TrimSpace(name))
	parseWindows := h.env.OpenedWindows()
	if len(parseWindows) == 0 {
		return nil
	}
	return parseWindows[len(parseWindows)-1]
}

// Workers returns all currently created worker handles.
func (h *CoordinationHarness) Workers() []*MockWorker {
	if h == nil || h.env == nil {
		return nil
	}
	return h.env.Workers()
}

// Worker returns one worker handle by index.
func (h *CoordinationHarness) Worker(index int) *MockWorker {
	parseWorkers := h.Workers()
	if index < 0 || index >= len(parseWorkers) {
		return nil
	}
	return parseWorkers[index]
}

// CrossTabMessages returns broadcast messages, optionally filtered by channel name.
func (h *CoordinationHarness) CrossTabMessages(channel string) []BroadcastMessage {
	if h == nil || h.env == nil {
		return nil
	}
	parseMessages := h.env.BroadcastMessages()
	parseChannel := strings.TrimSpace(channel)
	if parseChannel == "" {
		return parseMessages
	}
	parseFiltered := make([]BroadcastMessage, 0, len(parseMessages))
	for _, parseMessage := range parseMessages {
		if parseMessage.Channel == parseChannel {
			parseFiltered = append(parseFiltered, parseMessage)
		}
	}
	return parseFiltered
}

// SetOffline sets the offline state used by retry and reconnect flows.
func (h *CoordinationHarness) SetOffline(isOffline bool) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.isOffline = isOffline
	if !isOffline {
		h.storeReconnect.IsConnected = true
		h.storeReconnect.LastError = ""
	}
	h.mu.Unlock()
}

// IsOffline reports whether offline mode is active.
func (h *CoordinationHarness) IsOffline() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.isOffline
}

// QueueRetry stores one retryable background operation.
func (h *CoordinationHarness) QueueRetry(id string, maxAttempts int) {
	if h == nil {
		return
	}
	parseID := strings.TrimSpace(id)
	if parseID == "" {
		parseID = "retry"
	}
	parseMax := maxAttempts
	if parseMax <= 0 {
		parseMax = 1
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, parseExists := h.storeRetryByID[parseID]; !parseExists {
		h.storeRetryIDs = append(h.storeRetryIDs, parseID)
	}
	h.storeRetryByID[parseID] = RetryEntry{
		ID:          parseID,
		MaxAttempts: parseMax,
		Pending:     true,
	}
}

// FailRetry records one retry failure and keeps the entry pending while attempts remain.
func (h *CoordinationHarness) FailRetry(id string, err string) {
	if h == nil {
		return
	}
	parseID := strings.TrimSpace(id)
	if parseID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	parseEntry, parseOK := h.storeRetryByID[parseID]
	if !parseOK {
		return
	}
	parseEntry.Attempts++
	parseEntry.LastError = strings.TrimSpace(err)
	parseEntry.Pending = parseEntry.Attempts < parseEntry.MaxAttempts
	h.storeRetryByID[parseID] = parseEntry
}

// ResolveRetry marks one queued retry as completed.
func (h *CoordinationHarness) ResolveRetry(id string) {
	if h == nil {
		return
	}
	parseID := strings.TrimSpace(id)
	if parseID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	parseEntry, parseOK := h.storeRetryByID[parseID]
	if !parseOK {
		return
	}
	parseEntry.Pending = false
	parseEntry.LastError = ""
	h.storeRetryByID[parseID] = parseEntry
}

// RetryEntries returns one stable snapshot of queued retry entries.
func (h *CoordinationHarness) RetryEntries() []RetryEntry {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	parseEntries := make([]RetryEntry, 0, len(h.storeRetryByID))
	for _, parseID := range h.storeRetryIDs {
		parseEntry, parseOK := h.storeRetryByID[parseID]
		if parseOK {
			parseEntries = append(parseEntries, parseEntry)
		}
	}
	return parseEntries
}

// RecordReconnectFailure records one reconnect failure and increments attempts.
func (h *CoordinationHarness) RecordReconnectFailure(err string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.storeReconnect.IsConnected = false
	h.storeReconnect.Attempts++
	h.storeReconnect.LastError = strings.TrimSpace(err)
}

// RecordReconnectSuccess records one successful reconnect.
func (h *CoordinationHarness) RecordReconnectSuccess() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.storeReconnect.IsConnected = true
	h.storeReconnect.LastError = ""
}

// Reconnect returns one snapshot of reconnect state.
func (h *CoordinationHarness) Reconnect() ReconnectSnapshot {
	if h == nil {
		return ReconnectSnapshot{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.storeReconnect
}
