package gwcattrs

import (
	"strings"
	"testing"
)

// TestKeysAreStableLiterals pins the public wire strings so a rename is a deliberate change.
func TestKeysAreStableLiterals(parseT *testing.T) {
	parseWant := map[string]string{
		KeyErrorCode: "gwc.error.code", KeyComponent: "gwc.component", KeyComponentStack: "gwc.component_stack",
		KeyRenderMode: "gwc.render.mode", KeyLane: "gwc.lane", KeyHydrationStrict: "gwc.hydration.strict",
		KeyExceptionType: "exception.type", KeyExceptionMessage: "exception.message",
		KeyExceptionStacktrace: "exception.stacktrace", KeySSRHydrationFailure: "gwc.ssr.hydration.failure",
		KeySSRCorrelationID: "gwc.ssr.correlation_id",
	}
	for parseGot, parseExpected := range parseWant {
		if parseGot != parseExpected {
			parseT.Fatalf("key = %q, want %q", parseGot, parseExpected)
		}
		if strings.TrimSpace(parseGot) == "" {
			parseT.Fatal("empty key")
		}
	}
}
