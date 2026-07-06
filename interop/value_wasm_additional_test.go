//go:build js && wasm

package interop

import (
	"math"
	"syscall/js"
	"testing"
)

// TestValuePrimitiveAccessorsWasmCoverZeroAndTypedValues verifies the wasm Value helpers handle both zero and typed JS values.
func TestValuePrimitiveAccessorsWasmCoverZeroAndTypedValues(parseT *testing.T) {
	parseZero := Value{}
	if parseZero.Present() {
		parseT.Fatal("expected zero Value to be absent")
	}
	if parseZero.Truthy() {
		parseT.Fatal("expected zero Value to be falsy")
	}
	if !parseZero.IsUndefined() {
		parseT.Fatal("expected zero Value to report undefined")
	}
	if !parseZero.IsNull() {
		parseT.Fatal("expected zero Value to report null for missing raw state")
	}
	if parseZero.String() != "" || parseZero.Int() != 0 || parseZero.Float() != 0 || parseZero.Bool() {
		parseT.Fatalf("expected zero Value accessors to return empty zero-values, got string=%q int=%d float=%f bool=%t", parseZero.String(), parseZero.Int(), parseZero.Float(), parseZero.Bool())
	}
	if parseZero.Get("missing").Present() {
		parseT.Fatal("expected property lookup on zero Value to remain absent")
	}

	parseTruthy := Value{raw: js.ValueOf(true)}
	if !parseTruthy.Present() || !parseTruthy.Truthy() || !parseTruthy.Bool() {
		parseT.Fatal("expected boolean Value helpers to report a present truthy true value")
	}

	parseFalse := Value{raw: js.ValueOf(false)}
	if !parseFalse.Present() || parseFalse.Truthy() || parseFalse.Bool() {
		parseT.Fatal("expected boolean false Value to remain present but falsy")
	}

	parseFloat := Value{raw: js.ValueOf(7.25)}
	if math.Abs(parseFloat.Float()-7.25) > 0.0001 {
		parseT.Fatalf("expected float Value to decode 7.25, got %f", parseFloat.Float())
	}
	if parseFloat.Int() != 7 {
		parseT.Fatalf("expected numeric Value.Int() truncation to match js semantics, got %d", parseFloat.Int())
	}

	parseUndefined := Value{raw: js.Undefined()}
	if !parseUndefined.IsUndefined() || parseUndefined.Present() {
		parseT.Fatal("expected undefined Value to report absent undefined state")
	}

	parseNull := Value{raw: js.Null()}
	if !parseNull.IsNull() || parseNull.Present() {
		parseT.Fatal("expected null Value to report absent null state")
	}
}

// TestValueGetSetDeleteOnNullReceiverDoNotPanic pins #87: property access on a
// null/undefined receiver must return a safe zero/error rather than panicking
// the caller. syscall/js panics on Get/Set/Delete against a null/undefined
// js.Value, so the wrappers guard the receiver before dereferencing it.
func TestValueGetSetDeleteOnNullReceiverDoNotPanic(parseT *testing.T) {
	for parseName, parseRecv := range map[string]Value{
		"null":      {raw: js.Null()},
		"undefined": {raw: js.Undefined()},
	} {
		parseRecv := parseRecv
		parseT.Run(parseName, func(parseT *testing.T) {
			defer func() {
				if parseR := recover(); parseR != nil {
					parseT.Fatalf("Get/Set/Delete on %s receiver panicked: %v", parseName, parseR)
				}
			}()
			if parseRecv.Get("prop").Present() {
				parseT.Fatalf("expected Get on %s receiver to be absent", parseName)
			}
			if parseErr := parseRecv.Set("prop", 1); parseErr == nil {
				parseT.Fatalf("expected Set on %s receiver to return an error", parseName)
			}
			if parseErr := parseRecv.Delete("prop"); parseErr == nil {
				parseT.Fatalf("expected Delete on %s receiver to return an error", parseName)
			}
		})
	}
}

// TestJSErrorTextWasmPrefersMessageSources verifies JavaScript exception text selection prefers explicit detail before lossy fallbacks.
func TestJSErrorTextWasmPrefersMessageSources(parseT *testing.T) {
	if parseGot := jsExceptionError(nil).Error(); parseGot != "javascript exception" {
		parseT.Fatalf("expected nil exception text fallback, got %q", parseGot)
	}
	if parseGot := jsExceptionError("").Error(); parseGot != "javascript exception" {
		parseT.Fatalf("expected empty string exception text fallback, got %q", parseGot)
	}
	if parseGot := jsExceptionError("boom").Error(); parseGot != "boom" {
		parseT.Fatalf("expected string exception text, got %q", parseGot)
	}

	if parseGot := jsErrorText(js.Undefined()); parseGot != "javascript exception" {
		parseT.Fatalf("expected undefined error text fallback, got %q", parseGot)
	}
	if parseGot := jsErrorText(js.ValueOf("  exploded  ")); parseGot != "exploded" {
		parseT.Fatalf("expected trimmed string error text, got %q", parseGot)
	}

	parseWithMessage := js.Global().Get("Object").New()
	parseWithMessage.Set("message", " object boom ")
	if parseGot := jsErrorText(parseWithMessage); parseGot != "object boom" {
		parseT.Fatalf("expected explicit message field to win, got %q", parseGot)
	}

	parseSummary := js.Global().Get("Object").New()
	parseSummary.Set("message", " ")
	parseSummary.Set("kind", "boom")
	if parseGot := jsErrorText(parseSummary); parseGot != "{\"message\":\" \",\"kind\":\"boom\"}" {
		parseT.Fatalf("expected JSON summary fallback, got %q", parseGot)
	}
}

// TestValueSetFunctionWasmReturnsUndefinedForUnserializableResult verifies callback result encoding failures degrade to undefined instead of panicking.
func TestValueSetFunctionWasmReturnsUndefinedForUnserializableResult(parseT *testing.T) {
	parseGlobal, parseErr := GetGlobalThis()
	if parseErr != nil {
		parseT.Fatalf("expected globalThis wrapper, got %v", parseErr)
	}

	parseSubscription, parseErr := parseGlobal.SetFunction("__interopUnsupportedResultProbe", func(parseArgs ...Value) any {
		_ = parseArgs
		return func() {}
	})
	if parseErr != nil {
		parseT.Fatalf("expected function binding to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseResult, parseErr := parseGlobal.Get("__interopUnsupportedResultProbe").Invoke()
	if parseErr != nil {
		parseT.Fatalf("expected unsupported callback result to degrade to undefined, got %v", parseErr)
	}
	if !parseResult.IsUndefined() {
		parseT.Fatalf("expected unsupported callback result to become undefined, got %#v", parseResult)
	}
}
