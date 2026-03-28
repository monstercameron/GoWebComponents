package cachecore

import "testing"

// TestApplyStateAndHooks verifies connectivity state transitions trigger hooks and eligibility helpers.
func TestApplyStateAndHooks(parseT *testing.T) {
	parseCoordinator := BuildConnectivityCoordinator(ConnectivityState{
		IsOnline:      false,
		IsTunnelReady: false,
		IsAppActive:   false,
	})
	parseHookCalls := 0
	parseCoordinator.SetHook("hook-1", func(parseState ConnectivityState) {
		if parseState.UpdatedAt == "" {
			parseT.Fatal("expected updated_at value")
		}
		parseHookCalls++
	})
	parseState := parseCoordinator.ApplyOnlineState(true)
	if !parseState.IsOnline {
		parseT.Fatalf("expected online state true, got %+v", parseState)
	}
	parseState = parseCoordinator.ApplyTunnelState(true)
	if !parseState.IsTunnelReady {
		parseT.Fatalf("expected tunnel state true, got %+v", parseState)
	}
	parseState = parseCoordinator.ApplyAppActiveState(true)
	if !parseState.IsAppActive {
		parseT.Fatalf("expected app-active state true, got %+v", parseState)
	}
	if parseHookCalls < 3 {
		parseT.Fatalf("expected hook calls on each transition, got %d", parseHookCalls)
	}
	if !parseCoordinator.ShouldRunBackgroundRefresh() {
		parseT.Fatal("expected background refresh allowed when online+active")
	}
	if !parseCoordinator.ShouldFlushOutbox() {
		parseT.Fatal("expected outbox flush allowed when online+tunnel+active")
	}
}

// TestApplyStateNoopTransition verifies hooks are not re-emitted for no-op transitions.
func TestApplyStateNoopTransition(parseT *testing.T) {
	parseCoordinator := BuildConnectivityCoordinator(ConnectivityState{
		IsOnline:      true,
		IsTunnelReady: true,
		IsAppActive:   true,
	})
	parseHookCalls := 0
	parseCoordinator.SetHook("hook-1", func(parseState ConnectivityState) {
		_ = parseState
		parseHookCalls++
	})
	parseCoordinator.ApplyState(parseCoordinator.GetState())
	if parseHookCalls != 0 {
		parseT.Fatalf("expected no hook calls for no-op transition, got %d", parseHookCalls)
	}
}
