//go:build !js || !wasm

package interop_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

// TestRequestAnimationFrameNative verifies that the native stub for
// RequestAnimationFrame returns a non-nil cancel func and never invokes the
// callback synchronously (there is no display pipeline on native builds).
func TestRequestAnimationFrameNative(t *testing.T) {
	parseCalled := false
	parseCancel := interop.RequestAnimationFrame(func(parseTimestampMillis float64) {
		parseCalled = true
	})

	// The cancel func must be non-nil so callers can unconditionally call it on
	// component unmount without a nil-check.
	if parseCancel == nil {
		t.Fatal("RequestAnimationFrame returned nil cancel func on native")
	}

	// The callback must not have been called synchronously.
	if parseCalled {
		t.Fatal("RequestAnimationFrame invoked the callback synchronously on native; expected no-op")
	}

	// Cancel should be callable without panicking.
	parseCancel()
}
