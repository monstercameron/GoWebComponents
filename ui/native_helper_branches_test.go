//go:build !js || !wasm

package ui

import (
	"strings"
	"testing"
	"time"

	internalruntime "github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestNativeTraceHelpersParseAndNormalize verifies traceparent parsing and ID normalization helpers.
func TestNativeTraceHelpersParseAndNormalize(parseT *testing.T) {
	if _, parseOk := parseTraceparent("bad-header"); parseOk {
		parseT.Fatal("expected invalid traceparent to be rejected")
	}
	if _, parseOk := parseTraceparent("GG-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"); parseOk {
		parseT.Fatal("expected invalid traceparent version to be rejected")
	}
	if _, parseOk := parseTraceparent("00-00000000000000000000000000000000-00f067aa0ba902b7-01"); parseOk {
		parseT.Fatal("expected zero trace id to be rejected")
	}
	parseContext, parseOk := parseTraceparent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if !parseOk {
		parseT.Fatal("expected valid traceparent to parse")
	}
	if parseContext.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" || parseContext.ParentSpanID != "00f067aa0ba902b7" || parseContext.Flags != "01" {
		parseT.Fatalf("unexpected parsed trace context %#v", parseContext)
	}

	if parseGot := normalizeToTraceID("00f067aa0ba902b7"); parseGot != strings.Repeat("0", 16)+"00f067aa0ba902b7" {
		parseT.Fatalf("unexpected normalized 16-char trace id %q", parseGot)
	}
	if parseGot := normalizeToTraceID("abcdefabcdefabcdefabcdef"); parseGot != "00000000abcdefabcdefabcdefabcdef" {
		parseT.Fatalf("unexpected normalized 24-char trace id %q", parseGot)
	}
	if parseGot := normalizeToTraceID("4BF92F3577B34DA6A3CE929D0E0E4736"); parseGot != "4bf92f3577b34da6a3ce929d0e0e4736" {
		parseT.Fatalf("expected uppercase trace id to normalize to lowercase, got %q", parseGot)
	}
	if parseGot := normalizeToTraceID("not-hex"); parseGot != "not-hex" {
		parseT.Fatalf("expected invalid trace id to pass through unchanged, got %q", parseGot)
	}

	parseTraceID := generateTraceID()
	parseSpanID := generateSpanID()
	if len(parseTraceID) != 32 || !isLowercaseHex(parseTraceID) {
		parseT.Fatalf("expected 32-char lowercase hex trace id, got %q", parseTraceID)
	}
	if len(parseSpanID) != 16 || !isLowercaseHex(parseSpanID) {
		parseT.Fatalf("expected 16-char lowercase hex span id, got %q", parseSpanID)
	}
}

// TestNativePanicAndHydrationHelpers verifies native unsupported-API messaging and hydration observations.
func TestNativePanicAndHydrationHelpers(parseT *testing.T) {
	if parseMessage := unsupportedOnServerMessage(""); !strings.Contains(parseMessage, "ui.API") {
		parseT.Fatalf("expected default unsupported message to reference ui.API, got %q", parseMessage)
	}
	if parseMessage := actionableUnsupportedOnServerPanic("UseContext"); !strings.Contains(parseMessage, "ui.UseContext") {
		parseT.Fatalf("expected actionable panic text to reference ui.UseContext, got %q", parseMessage)
	}

	parseFinishedAt := time.Now().UTC()
	parseStartedAt := parseFinishedAt.Add(-25 * time.Millisecond)
	parseSuccess := newSSRHydrationObservation(internalruntime.HydrationMetrics{
		CorrelationID:        "corr-1",
		StartedAt:            parseStartedAt,
		FinishedAt:           parseFinishedAt,
		Duration:             25 * time.Millisecond,
		DurationNs:           int64(25 * time.Millisecond),
		ExistingDOMNodeCount: 8,
		FallbackCount:        1,
		MismatchCount:        2,
		DiscardedNodeCount:   3,
		Strict:               true,
	})
	if parseSuccess.Phase != "finish" || parseSuccess.Domain != "runtime" || parseSuccess.Hydration == nil || parseSuccess.Hydration.MismatchCount != 2 {
		parseT.Fatalf("unexpected hydration success observation %#v", parseSuccess)
	}
	parseFailure := newSSRHydrationObservation(internalruntime.HydrationMetrics{
		CorrelationID: "corr-2",
		FinishedAt:    parseFinishedAt,
		Failed:        true,
		Failure:       "mismatch",
	})
	if parseFailure.Phase != "error" || parseFailure.Hydration == nil || parseFailure.Hydration.Failure != "mismatch" {
		parseT.Fatalf("unexpected hydration failure observation %#v", parseFailure)
	}
}

// TestNativeSSRTransferEnvelopeHelpers verifies direct envelope encoding, decoding, and legacy detection branches.
func TestNativeSSRTransferEnvelopeHelpers(parseT *testing.T) {
	parseEnvelope, parseErr := encodeSSRPayloadEnvelope(time.Unix(0, 42).UTC(), SSRPayloadOptions{Encoding: SSRPayloadEncodingTimeUnixNano, Scope: SSRPayloadScopeSubtree, Target: "cart"})
	if parseErr != nil {
		parseT.Fatalf("encodeSSRPayloadEnvelope(time-unix-nano): %v", parseErr)
	}
	if parseEnvelope.Encoding != SSRPayloadEncodingTimeUnixNano || parseEnvelope.Scope != SSRPayloadScopeSubtree || parseEnvelope.Target != "cart" {
		parseT.Fatalf("unexpected encoded time envelope %#v", parseEnvelope)
	}
	parseStamp, parseErr := decodeSSRPayloadEnvelope[time.Time](parseEnvelope)
	if parseErr != nil || parseStamp.UnixNano() != 42 {
		parseT.Fatalf("decodeSSRPayloadEnvelope(time-unix-nano) = %v, %v; want unix nano 42", parseStamp, parseErr)
	}

	parseCBOR, parseErr := encodeSSRPayloadEnvelope(map[string]int{"count": 3}, SSRPayloadOptions{Encoding: SSRPayloadEncodingCBOR})
	if parseErr != nil {
		parseT.Fatalf("encodeSSRPayloadEnvelope(cbor): %v", parseErr)
	}
	parseDecodedMap, parseErr := decodeSSRPayloadEnvelope[map[string]int](parseCBOR)
	if parseErr != nil || parseDecodedMap["count"] != 3 {
		parseT.Fatalf("decodeSSRPayloadEnvelope(cbor) = %#v, %v; want count=3", parseDecodedMap, parseErr)
	}

	parseLegacyEnvelope, isParseLegacy, parseErr := envelopeFromBootstrapData(map[string]any{"hello": "world"})
	if parseErr != nil || !isParseLegacy || parseLegacyEnvelope.Encoding != SSRPayloadEncodingJSON {
		parseT.Fatalf("unexpected legacy envelope result envelope=%#v legacy=%v err=%v", parseLegacyEnvelope, isParseLegacy, parseErr)
	}
	parseDirectEnvelope, isParseLegacy, parseErr := envelopeFromBootstrapData(SSRPayloadEnvelope{Version: CurrentSSRBootstrapVersion, Encoding: SSRPayloadEncodingText, Text: "atlas"})
	if parseErr != nil || isParseLegacy || parseDirectEnvelope.Text != "atlas" {
		parseT.Fatalf("unexpected direct envelope result envelope=%#v legacy=%v err=%v", parseDirectEnvelope, isParseLegacy, parseErr)
	}

	if _, parseErr2 := decodeSSRPayloadEnvelope[time.Time](SSRPayloadEnvelope{Encoding: SSRPayloadEncodingTimeUnixNano, Text: "bad"}); parseErr2 == nil {
		parseT.Fatal("expected invalid unix-nano payload text to fail decoding")
	}
	if _, parseErr3 := decodeSSRPayloadEnvelope[string](SSRPayloadEnvelope{Encoding: SSRPayloadEncodingCBOR, Binary: []byte("not-cbor")}); parseErr3 == nil {
		parseT.Fatal("expected invalid cbor payload to fail decoding")
	}
}

// TestNativeUseContextAndAsyncBoundaryBranches verifies context resolution and fallback branches in the native SSR slice.
func TestNativeUseContextAndAsyncBoundaryBranches(parseT *testing.T) {
	// UseContext resolves through the runtime's SSR hook fiber on native
	// builds (it used to panic UnsupportedOnServer despite full runtime
	// support): a provider value reaches the consuming component.
	parseTheme := CreateContext("light")
	parseConsumer := func() Node {
		return Text(UseContext(parseTheme))
	}
	parseContextMarkup, parseContextErr := RenderToString(CreateElement(parseTheme.Provider, ContextProviderProps[string]{
		Value: "dark",
		Child: CreateElement(parseConsumer),
	}))
	if parseContextErr != nil {
		parseT.Fatalf("unexpected UseContext SSR error: %v", parseContextErr)
	}
	if parseContextMarkup != "dark" {
		parseT.Fatalf("expected provider value to resolve through native SSR, got %q", parseContextMarkup)
	}

	// Outside a render it still panics (hook outside component context).
	func() {
		defer func() {
			if recover() == nil {
				parseT.Fatal("expected UseContext outside a render to panic")
			}
		}()
		_ = UseContext(parseTheme)
	}()

	parseErrorFallback := Text("error-fallback")
	parseErrorMarkup, parseErrorRenderErr := RenderToString(AsyncBoundary(AsyncBoundaryProps{
		Error:         errNativeHelperTestMessage("boom"),
		ErrorFallback: func(error) Node { return parseErrorFallback },
	}))
	if parseErrorRenderErr != nil {
		parseT.Fatalf("unexpected AsyncBoundary error fallback render error: %v", parseErrorRenderErr)
	}
	if parseErrorMarkup != "error-fallback" {
		parseT.Fatalf("expected AsyncBoundary error fallback markup, got %q", parseErrorMarkup)
	}
	parseTimeoutFallback := Text("timeout-fallback")
	parseTimeoutMarkup, parseTimeoutRenderErr := RenderToString(AsyncBoundary(AsyncBoundaryProps{Pending: true, Timeout: time.Second, TimeoutFallback: parseTimeoutFallback, Fallback: Text("fallback")}))
	if parseTimeoutRenderErr != nil {
		parseT.Fatalf("unexpected AsyncBoundary timeout render error: %v", parseTimeoutRenderErr)
	}
	if parseTimeoutMarkup != "timeout-fallback" {
		parseT.Fatalf("expected AsyncBoundary timeout fallback markup, got %q", parseTimeoutMarkup)
	}
	UseEffect(func() func() { return nil })
}

type errNativeHelperTestMessage string

// Error returns the string form used by native helper tests.
func (parseE errNativeHelperTestMessage) Error() string { return string(parseE) }
