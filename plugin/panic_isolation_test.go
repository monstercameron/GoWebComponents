package plugin

import "testing"

// TestHostIsolatesPanickingPluginCallbacks pins that a third-party plugin whose
// dispatch callbacks panic does NOT crash the host: the guard degrades to allow,
// the cache-key decorator passes the key through, and the panics are observed
// (not silently swallowed) via PluginPanicHandler.
func TestHostIsolatesPanickingPluginCallbacks(parseT *testing.T) {
	parseOldHandler := PluginPanicHandler
	parseObserved := 0
	PluginPanicHandler = func(string, any) { parseObserved++ }
	defer func() { PluginPanicHandler = parseOldHandler }()

	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData}})
	parseErr := parseHost.Register(Define(Manifest{
		ID:       "panicky",
		Version:  "1.0.0",
		Tier:     TierExperimental,
		Requires: []Capability{CapabilityRouter, CapabilityAsyncData},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		if parseErr := parseHost2.AddRouteGuard(func(RouteRequest) GuardDecision {
			panic("guard boom")
		}); parseErr != nil {
			return nil, parseErr
		}
		if parseErr := parseHost2.AddCacheKeyDecorator(func(string) string {
			panic("decorator boom")
		}); parseErr != nil {
			return nil, parseErr
		}
		return nil, nil
	}))
	if parseErr != nil {
		parseT.Fatalf("register: %v", parseErr)
	}

	parseDecision := parseHost.EvaluateRoute(RouteRequest{})
	if parseDecision.Outcome != GuardAllow {
		parseT.Fatalf("expected allow after guard panic, got %#v", parseDecision)
	}
	if parseGot := parseHost.DecorateCacheKey("k"); parseGot != "k" {
		parseT.Fatalf("expected key passthrough after decorator panic, got %q", parseGot)
	}
	if parseObserved < 2 {
		parseT.Fatalf("expected the two panics to be observed, got %d", parseObserved)
	}
}
