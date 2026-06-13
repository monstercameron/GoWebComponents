//go:build !js || !wasm

package ssr

import "testing"

func TestNativeHydrationWrappersDelegateToTestkit(t *testing.T) {
	parseCalls := []string{}
	parsePrevSmoke := smokeHydrateBase
	parsePrevRoundTrip := roundTripHydrateBase
	parsePrevMismatch := roundTripHydrateMismatchBase
	smokeHydrateBase = func(parseTb testing.TB, parseRoot any, parseOptions ...HydrationOptions) *HydrationHarness {
		parseCalls = append(parseCalls, "smoke")
		return &HydrationHarness{}
	}
	roundTripHydrateBase = func(parseTb testing.TB, parseRoot any, parseOptions ...HydrationOptions) *HydrationHarness {
		parseCalls = append(parseCalls, "roundtrip")
		return &HydrationHarness{}
	}
	roundTripHydrateMismatchBase = func(parseTb testing.TB, parseRoot any, buildMutate func(string) string, parseOptions ...HydrationOptions) *HydrationHarness {
		if buildMutate("html") != "html-mutated" {
			t.Fatal("mutator was not passed through")
		}
		parseCalls = append(parseCalls, "mismatch")
		return &HydrationHarness{}
	}
	t.Cleanup(func() {
		smokeHydrateBase = parsePrevSmoke
		roundTripHydrateBase = parsePrevRoundTrip
		roundTripHydrateMismatchBase = parsePrevMismatch
	})

	if SmokeHydrate(t, "root") == nil || RoundTripHydrate(t, "root") == nil || RoundTripHydrateMismatch(t, "root", func(parseHTML string) string { return parseHTML + "-mutated" }) == nil {
		t.Fatal("wrapper returned nil harness from fake delegate")
	}
	parseWant := []string{"smoke", "roundtrip", "mismatch"}
	for parseIndex, parseCall := range parseWant {
		if parseCalls[parseIndex] != parseCall {
			t.Fatalf("calls = %#v, want %#v", parseCalls, parseWant)
		}
	}
}
