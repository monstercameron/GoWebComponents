//go:build js && wasm

package interop

import (
	"context"
	"syscall/js"
	"testing"
)

// TestAwaitResolvesPromise proves Await bridges a resolved JS Promise into Go.
func TestAwaitResolvesPromise(parseT *testing.T) {
	parsePromise := Value{raw: js.Global().Get("Promise").Call("resolve", "hello")}
	parseResult, parseErr := parsePromise.Await(context.Background())
	if parseErr != nil {
		parseT.Fatalf("Await returned error: %v", parseErr)
	}
	if parseGot := parseResult.String(); parseGot != "hello" {
		parseT.Fatalf("Await resolved to %q, want hello", parseGot)
	}
}

// TestAwaitRejectionReturnsError proves a rejected promise surfaces as a
// CodePromiseRejected error, not a panic or a silent zero value.
func TestAwaitRejectionReturnsError(parseT *testing.T) {
	parseErrObj := js.Global().Get("Error").New("boom")
	parsePromise := Value{raw: js.Global().Get("Promise").Call("reject", parseErrObj)}
	_, parseErr := parsePromise.Await(context.Background())
	if parseErr == nil {
		parseT.Fatal("expected rejection error")
	}
	var parseTyped *Error
	if !asInteropError(parseErr, &parseTyped) || parseTyped.Code != CodePromiseRejected {
		parseT.Fatalf("expected CodePromiseRejected, got %v", parseErr)
	}
}

// TestAwaitNonPromisePassthrough proves a synchronous (non-thenable) value is
// returned unchanged, so Await is safe to call on already-resolved data.
func TestAwaitNonPromisePassthrough(parseT *testing.T) {
	parseValue := Value{raw: js.ValueOf(42)}
	parseResult, parseErr := parseValue.Await(context.Background())
	if parseErr != nil {
		parseT.Fatalf("Await returned error: %v", parseErr)
	}
	if parseGot := parseResult.Int(); parseGot != 42 {
		parseT.Fatalf("Await passthrough = %d, want 42", parseGot)
	}
}

// TestAwaitContextCancelStopsWaiting proves a canceled context unblocks Await on
// a promise that never settles, returning the context error.
func TestAwaitContextCancelStopsWaiting(parseT *testing.T) {
	parseExecutor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		// Never call resolve/reject: the promise stays pending forever.
		return nil
	})
	defer parseExecutor.Release()
	parsePending := Value{raw: js.Global().Get("Promise").New(parseExecutor)}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel() // already canceled

	_, parseErr := parsePending.Await(parseCtx)
	if parseErr == nil {
		parseT.Fatal("expected context-cancel error from a never-settling promise")
	}
}

// TestAwaitUnavailableOnZeroValue proves Await on a value with no underlying JS
// handle reports unavailable.
func TestAwaitUnavailableOnZeroValue(parseT *testing.T) {
	var parseZero Value
	if _, parseErr := parseZero.Await(context.Background()); parseErr == nil {
		parseT.Fatal("expected unavailable error on zero Value")
	}
}

func asInteropError(parseErr error, parseTarget **Error) bool {
	for parseErr != nil {
		if parseTyped, parseOk := parseErr.(*Error); parseOk {
			*parseTarget = parseTyped
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if parseU, parseOk := parseErr.(unwrapper); parseOk {
			parseErr = parseU.Unwrap()
			continue
		}
		break
	}
	return false
}
