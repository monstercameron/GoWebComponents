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
func NewCoordinationHarness(parseTb testing.TB, parseOptions ...Options) *CoordinationHarness {
	parseTb.Helper()
	parseHarness := &CoordinationHarness{
		tb:             parseTb,
		env:            Install(parseTb, parseOptions...),
		storeRetryByID: map[string]RetryEntry{},
	}
	return parseHarness
}

// Environment returns the underlying browser environment for advanced setup.
func (parseH *CoordinationHarness) Environment() *Environment {
	if parseH == nil {
		return nil
	}
	return parseH.env
}

// OpenTab opens one additional mock browser tab and returns that tab handle.
func (parseH *CoordinationHarness) OpenTab(parsePath string, parseName string) *MockWindow {
	if parseH == nil || parseH.env == nil {
		return nil
	}
	parsePath := strings.TrimSpace(parsePath)
	if parsePath == "" {
		parsePath = "/"
	}
	parseH.env.Window().Call("open", parsePath, strings.TrimSpace(parseName))
	parseWindows := parseH.env.OpenedWindows()
	if len(parseWindows) == 0 {
		return nil
	}
	return parseWindows[len(parseWindows)-1]
}

// Workers returns all currently created worker handles.
func (parseH *CoordinationHarness) Workers() []*MockWorker {
	if parseH == nil || parseH.env == nil {
		return nil
	}
	return parseH.env.Workers()
}

// Worker returns one worker handle by index.
func (parseH *CoordinationHarness) Worker(parseIndex int) *MockWorker {
	parseWorkers := parseH.Workers()
	if parseIndex < 0 || parseIndex >= len(parseWorkers) {
		return nil
	}
	return parseWorkers[parseIndex]
}

// CrossTabMessages returns broadcast messages, optionally filtered by channel name.
func (parseH *CoordinationHarness) CrossTabMessages(parseChannel string) []BroadcastMessage {
	if parseH == nil || parseH.env == nil {
		return nil
	}
	parseMessages := parseH.env.BroadcastMessages()
	parseChannel := strings.TrimSpace(parseChannel)
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
func (parseH *CoordinationHarness) SetOffline(isOffline bool) {
	if parseH == nil {
		return
	}
	parseH.mu.Lock()
	parseH.isOffline = isOffline
	if !isOffline {
		parseH.storeReconnect.IsConnected = true
		parseH.storeReconnect.LastError = ""
	}
	parseH.mu.Unlock()
}

// IsOffline reports whether offline mode is active.
func (parseH *CoordinationHarness) IsOffline() bool {
	if parseH == nil {
		return false
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.isOffline
}

// QueueRetry stores one retryable background operation.
func (parseH *CoordinationHarness) QueueRetry(parseId string, parseMaxAttempts int) {
	if parseH == nil {
		return
	}
	parseID := strings.TrimSpace(parseId)
	if parseID == "" {
		parseID = "retry"
	}
	parseMax := parseMaxAttempts
	if parseMax <= 0 {
		parseMax = 1
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	if _, parseExists := parseH.storeRetryByID[parseID]; !parseExists {
		parseH.storeRetryIDs = append(parseH.storeRetryIDs, parseID)
	}
	parseH.storeRetryByID[parseID] = RetryEntry{
		ID:          parseID,
		MaxAttempts: parseMax,
		Pending:     true,
	}
}

// FailRetry records one retry failure and keeps the entry pending while attempts remain.
func (parseH *CoordinationHarness) FailRetry(parseId string, parseErr string) {
	if parseH == nil {
		return
	}
	parseID := strings.TrimSpace(parseId)
	if parseID == "" {
		return
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseEntry, parseOK := parseH.storeRetryByID[parseID]
	if !parseOK {
		return
	}
	parseEntry.Attempts++
	parseEntry.LastError = strings.TrimSpace(parseErr)
	parseEntry.Pending = parseEntry.Attempts < parseEntry.MaxAttempts
	parseH.storeRetryByID[parseID] = parseEntry
}

// ResolveRetry marks one queued retry as completed.
func (parseH *CoordinationHarness) ResolveRetry(parseId string) {
	if parseH == nil {
		return
	}
	parseID := strings.TrimSpace(parseId)
	if parseID == "" {
		return
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseEntry, parseOK := parseH.storeRetryByID[parseID]
	if !parseOK {
		return
	}
	parseEntry.Pending = false
	parseEntry.LastError = ""
	parseH.storeRetryByID[parseID] = parseEntry
}

// RetryEntries returns one stable snapshot of queued retry entries.
func (parseH *CoordinationHarness) RetryEntries() []RetryEntry {
	if parseH == nil {
		return nil
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseEntries := make([]RetryEntry, 0, len(parseH.storeRetryByID))
	for _, parseID := range parseH.storeRetryIDs {
		parseEntry, parseOK := parseH.storeRetryByID[parseID]
		if parseOK {
			parseEntries = append(parseEntries, parseEntry)
		}
	}
	return parseEntries
}

// RecordReconnectFailure records one reconnect failure and increments attempts.
func (parseH *CoordinationHarness) RecordReconnectFailure(parseErr string) {
	if parseH == nil {
		return
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseH.storeReconnect.IsConnected = false
	parseH.storeReconnect.Attempts++
	parseH.storeReconnect.LastError = strings.TrimSpace(parseErr)
}

// RecordReconnectSuccess records one successful reconnect.
func (parseH *CoordinationHarness) RecordReconnectSuccess() {
	if parseH == nil {
		return
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseH.storeReconnect.IsConnected = true
	parseH.storeReconnect.LastError = ""
}

// Reconnect returns one snapshot of reconnect state.
func (parseH *CoordinationHarness) Reconnect() ReconnectSnapshot {
	if parseH == nil {
		return ReconnectSnapshot{}
	}
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.storeReconnect
}
