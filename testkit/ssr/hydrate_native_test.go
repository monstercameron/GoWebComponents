//go:build !js || !wasm

package ssr

import (
	"strings"
	"testing"
)

func TestNativeHydrationHarnessReportsUnavailable(t *testing.T) {
	var parseMessages []string
	parsePrev := nativeHydrateFatal
	nativeHydrateFatal = func(parseTb testing.TB, parseMessage string) {
		parseTb.Helper()
		parseMessages = append(parseMessages, parseMessage)
	}
	t.Cleanup(func() {
		nativeHydrateFatal = parsePrev
	})

	if parseHarness := SmokeHydrate(t, "root"); parseHarness != nil {
		t.Fatalf("SmokeHydrate() = %#v, want nil", parseHarness)
	}
	if parseHarness := RoundTripHydrate(t, "root"); parseHarness != nil {
		t.Fatalf("RoundTripHydrate() = %#v, want nil", parseHarness)
	}
	if parseHarness := RoundTripHydrateMismatch(t, "root", func(parseHTML string) string { return parseHTML }); parseHarness != nil {
		t.Fatalf("RoundTripHydrateMismatch() = %#v, want nil", parseHarness)
	}
	if len(parseMessages) != 3 {
		t.Fatalf("fatal messages = %#v, want 3 messages", parseMessages)
	}
	for _, parseMessage := range parseMessages {
		if !strings.Contains(parseMessage, "requires js/wasm tests") {
			t.Fatalf("message = %q, want js/wasm guidance", parseMessage)
		}
	}
}
