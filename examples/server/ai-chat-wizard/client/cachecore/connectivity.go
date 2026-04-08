package cachecore

import (
	"strings"
	"sync"
	"time"
)

// ConnectivityState stores connectivity and app-lifecycle readiness inputs.
type ConnectivityState struct {
	IsOnline      bool
	IsTunnelReady bool
	IsAppActive   bool
	UpdatedAt     string
}

// ConnectivityHook handles one connectivity state transition.
type ConnectivityHook func(ConnectivityState)

// ConnectivityCoordinator tracks connectivity transitions and emits change notifications.
type ConnectivityCoordinator struct {
	parseMu    sync.Mutex
	parseState ConnectivityState
	parseHooks map[string]ConnectivityHook
}

// BuildConnectivityCoordinator creates one connectivity coordinator with one initial state.
func BuildConnectivityCoordinator(parseInitialState ConnectivityState) *ConnectivityCoordinator {
	parseInitialState = normalizeConnectivityState(parseInitialState)
	return &ConnectivityCoordinator{
		parseState: parseInitialState,
		parseHooks: map[string]ConnectivityHook{},
	}
}

// GetState returns one snapshot of current connectivity state.
func (parseCoordinator *ConnectivityCoordinator) GetState() ConnectivityState {
	if parseCoordinator == nil {
		return normalizeConnectivityState(ConnectivityState{})
	}
	parseCoordinator.parseMu.Lock()
	defer parseCoordinator.parseMu.Unlock()
	return parseCoordinator.parseState
}

// SetHook registers one named connectivity state hook.
func (parseCoordinator *ConnectivityCoordinator) SetHook(parseHookKey string, parseHook ConnectivityHook) {
	if parseCoordinator == nil || parseHook == nil {
		return
	}
	parseHookKey = strings.TrimSpace(parseHookKey)
	if parseHookKey == "" {
		return
	}
	parseCoordinator.parseMu.Lock()
	defer parseCoordinator.parseMu.Unlock()
	parseCoordinator.parseHooks[parseHookKey] = parseHook
}

// DeleteHook removes one named connectivity state hook.
func (parseCoordinator *ConnectivityCoordinator) DeleteHook(parseHookKey string) {
	if parseCoordinator == nil {
		return
	}
	parseHookKey = strings.TrimSpace(parseHookKey)
	if parseHookKey == "" {
		return
	}
	parseCoordinator.parseMu.Lock()
	defer parseCoordinator.parseMu.Unlock()
	delete(parseCoordinator.parseHooks, parseHookKey)
}

// ApplyState updates coordinator state and broadcasts hooks when values change.
func (parseCoordinator *ConnectivityCoordinator) ApplyState(parseNextState ConnectivityState) ConnectivityState {
	if parseCoordinator == nil {
		return normalizeConnectivityState(parseNextState)
	}
	parseNextState = normalizeConnectivityState(parseNextState)
	parseCoordinator.parseMu.Lock()
	isParseChanged := parseCoordinator.parseState != parseNextState
	parseCoordinator.parseState = parseNextState
	parseHooks := make([]ConnectivityHook, 0, len(parseCoordinator.parseHooks))
	for _, parseHook := range parseCoordinator.parseHooks {
		parseHooks = append(parseHooks, parseHook)
	}
	parseCoordinator.parseMu.Unlock()
	if isParseChanged {
		for _, parseHook := range parseHooks {
			parseHook(parseNextState)
		}
	}
	return parseNextState
}

// ApplyOnlineState updates only online-state and emits one normalized state snapshot.
func (parseCoordinator *ConnectivityCoordinator) ApplyOnlineState(isParseOnline bool) ConnectivityState {
	parseState := parseCoordinator.GetState()
	parseState.IsOnline = isParseOnline
	return parseCoordinator.ApplyState(parseState)
}

// ApplyTunnelState updates only tunnel-readiness state and emits one normalized state snapshot.
func (parseCoordinator *ConnectivityCoordinator) ApplyTunnelState(isParseTunnelReady bool) ConnectivityState {
	parseState := parseCoordinator.GetState()
	parseState.IsTunnelReady = isParseTunnelReady
	return parseCoordinator.ApplyState(parseState)
}

// ApplyAppActiveState updates only app-active lifecycle state and emits one normalized state snapshot.
func (parseCoordinator *ConnectivityCoordinator) ApplyAppActiveState(isParseAppActive bool) ConnectivityState {
	parseState := parseCoordinator.GetState()
	parseState.IsAppActive = isParseAppActive
	return parseCoordinator.ApplyState(parseState)
}

// ShouldRunBackgroundRefresh reports whether background refresh work is currently allowed.
func (parseCoordinator *ConnectivityCoordinator) ShouldRunBackgroundRefresh() bool {
	parseState := parseCoordinator.GetState()
	return parseState.IsOnline && parseState.IsAppActive
}

// ShouldFlushOutbox reports whether outbox flush work is currently allowed.
func (parseCoordinator *ConnectivityCoordinator) ShouldFlushOutbox() bool {
	parseState := parseCoordinator.GetState()
	return parseState.IsOnline && parseState.IsTunnelReady && parseState.IsAppActive
}

// normalizeConnectivityState normalizes one connectivity state with deterministic timestamp behavior.
func normalizeConnectivityState(parseState ConnectivityState) ConnectivityState {
	parseState.UpdatedAt = strings.TrimSpace(parseState.UpdatedAt)
	if parseState.UpdatedAt == "" {
		parseState.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return parseState
}
