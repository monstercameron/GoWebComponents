//go:build !js || !wasm

package interop

import (
	"bytes"
	"testing"
)

// v5 P3.1 — the native slice of binary transferables.
//
// The interesting behavior (actual ownership transfer, buffer detachment) is
// browser-only and covered by the wasm suite. What matters here is that the
// type exists off-browser so code building transferables compiles unchanged on
// the native and SSR slices, and that the send paths fail loudly rather than
// pretending to have sent something.

func TestTransferable_RoundTripsBytesNatively(parseT *testing.T) {
	parsePayload := []byte("patch-stream-bytes")

	parseTransferable, parseErr := NewTransferable(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("NewTransferable: %v", parseErr)
	}

	if parseGot := parseTransferable.Len(); parseGot != len(parsePayload) {
		parseT.Errorf("Len = %d, want %d", parseGot, len(parsePayload))
	}
	if parseGot := parseTransferable.Bytes(); !bytes.Equal(parseGot, parsePayload) {
		parseT.Errorf("Bytes = %q, want %q", parseGot, parsePayload)
	}
}

func TestTransferable_EmptyPayloadIsValid(parseT *testing.T) {
	parseTransferable, parseErr := NewTransferable(nil)
	if parseErr != nil {
		parseT.Fatalf("NewTransferable(nil): %v", parseErr)
	}
	if parseTransferable.Len() != 0 {
		parseT.Errorf("empty transferable reported length %d", parseTransferable.Len())
	}
}

// TestTransferable_NeverDetachedNatively: there is no postMessage off-browser,
// so nothing can take ownership away. A native caller must never be told its
// buffer was moved.
func TestTransferable_NeverDetachedNatively(parseT *testing.T) {
	parseTransferable, _ := NewTransferable([]byte("x"))
	if parseTransferable.IsDetached() {
		parseT.Error("a native transferable can never be detached; nothing can transfer it")
	}
}

// TestTransferable_SendPathsAreUnavailableNatively pins that the stubs report
// failure rather than silently succeeding. A send that appears to work
// off-browser would hide a real portability bug until runtime.
func TestTransferable_SendPathsAreUnavailableNatively(parseT *testing.T) {
	parseTransferable, _ := NewTransferable([]byte("x"))

	if parseErr := (Worker{}).PostTransferable("payload", parseTransferable); parseErr == nil {
		parseT.Error("Worker.PostTransferable must report unavailable off-browser")
	}
	if parseErr := (MessagePort{}).PostTransferable("payload", parseTransferable); parseErr == nil {
		parseT.Error("MessagePort.PostTransferable must report unavailable off-browser")
	}
}
