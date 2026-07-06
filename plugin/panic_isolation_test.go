package plugin

import (
	"strings"
	"testing"
)

// TestSetupEnforcesDeclaredCapabilities pins the #53 per-plugin capability boundary: during
// Setup a plugin may use only the capabilities it declared in Manifest.Requires, even when the
// host enables more. A plugin that reaches for an undeclared (but host-enabled) capability fails
// registration and is rolled back; one that declares it succeeds.
func TestSetupEnforcesDeclaredCapabilities(parseT *testing.T) {
	// Host enables BOTH capabilities, so only the per-plugin check can reject.
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData}})

	// Declares Router only, but uses AsyncData (AddCacheKeyDecorator).
	parseErr := parseHost.Register(Define(Manifest{
		ID:       "under-declared",
		Version:  "1.0.0",
		Tier:     TierExperimental,
		Requires: []Capability{CapabilityRouter},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		return nil, parseHost2.AddCacheKeyDecorator(func(parseKey string) string { return parseKey })
	}))
	if parseErr == nil {
		parseT.Fatal("expected Register to fail: plugin used a capability it did not declare")
	}
	if !strings.Contains(parseErr.Error(), "without declaring it") {
		parseT.Fatalf("unexpected error: %v", parseErr)
	}
	if len(parseHost.Plugins()) != 0 {
		parseT.Fatalf("the rejected plugin must be rolled back, got %d registered", len(parseHost.Plugins()))
	}

	// Declaring AsyncData makes the identical Add* call succeed.
	parseErr2 := parseHost.Register(Define(Manifest{
		ID:       "well-declared",
		Version:  "1.0.0",
		Tier:     TierExperimental,
		Requires: []Capability{CapabilityAsyncData},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		return nil, parseHost2.AddCacheKeyDecorator(func(parseKey string) string { return "ns:" + parseKey })
	}))
	if parseErr2 != nil {
		parseT.Fatalf("well-declared plugin must register: %v", parseErr2)
	}
	if parseGot := parseHost.DecorateCacheKey("k"); parseGot != "ns:k" {
		parseT.Fatalf("declared decorator should apply, got %q", parseGot)
	}

	// After Setup completes, direct host configuration (no active plugin scope) is
	// governed only by the host-wide capability check — the per-plugin gate is off.
	if parseErr3 := parseHost.AddRouteGuard(func(RouteRequest) GuardDecision { return GuardDecision{Outcome: GuardAllow} }); parseErr3 != nil {
		parseT.Fatalf("direct Add* outside Setup must not be gated per-plugin: %v", parseErr3)
	}
}

// TestDecorateCacheKeyRejectsOverlongOutput pins the #53 cache-key cap: a
// decorator that returns a runaway (huge) string is a memory/CPU DoS on the fetch
// hot path, so its output is REJECTED (the pre-decoration key is kept) rather
// than allowed through or truncated. A decorator that stays under the cap still
// applies, and a rejection is surfaced (not silent) via reportCacheKeyOverflow.
func TestDecorateCacheKeyRejectsOverlongOutput(parseT *testing.T) {
	parseOldReport := reportCacheKeyOverflow
	parseRejections := 0
	reportCacheKeyOverflow = func(int) { parseRejections++ }
	defer func() { reportCacheKeyOverflow = parseOldReport }()

	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityAsyncData}})

	// A decorator that balloons the key far past the cap.
	if parseErr := parseHost.AddCacheKeyDecorator(func(parseKey string) string {
		return parseKey + strings.Repeat("x", maxDecoratedCacheKeyLen*4)
	}); parseErr != nil {
		parseT.Fatalf("add overlong decorator: %v", parseErr)
	}
	// A well-behaved decorator registered AFTER the runaway one must still run on
	// the (rejected → original) key, proving rejection does not abort the chain.
	if parseErr := parseHost.AddCacheKeyDecorator(func(parseKey string) string {
		return "ns:" + parseKey
	}); parseErr != nil {
		parseT.Fatalf("add tame decorator: %v", parseErr)
	}

	parseGot := parseHost.DecorateCacheKey("inventory")
	if parseGot != "ns:inventory" {
		parseT.Fatalf("expected overlong output rejected and tame decorator applied, got %q (len %d)", parseGot, len(parseGot))
	}
	if len(parseGot) > maxDecoratedCacheKeyLen {
		parseT.Fatalf("decorated key of %d bytes exceeds the %d-byte cap", len(parseGot), maxDecoratedCacheKeyLen)
	}
	if parseRejections != 1 {
		parseT.Fatalf("expected exactly one surfaced rejection, got %d", parseRejections)
	}

	// A decorator whose output is exactly at the cap is allowed (boundary check).
	parseHost2 := NewHost(HostOptions{Capabilities: []Capability{CapabilityAsyncData}})
	if parseErr := parseHost2.AddCacheKeyDecorator(func(string) string {
		return strings.Repeat("y", maxDecoratedCacheKeyLen)
	}); parseErr != nil {
		parseT.Fatalf("add at-cap decorator: %v", parseErr)
	}
	if parseGot2 := parseHost2.DecorateCacheKey("k"); len(parseGot2) != maxDecoratedCacheKeyLen {
		parseT.Fatalf("at-cap output should pass through unchanged, got len %d", len(parseGot2))
	}
}

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
