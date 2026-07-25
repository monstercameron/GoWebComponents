//go:build !(js && wasm)

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// fakeMediaSource is a substitutable mediaQuerySource for deterministic tests.
type fakeMediaSource struct {
	matches bool
}

func (parseF fakeMediaSource) Matches() bool { return parseF.matches }
func (parseF fakeMediaSource) Subscribe(func(interop.MediaQueryEvent)) (interop.Subscription, error) {
	return interop.Subscription{}, nil
}

// TestUseMediaQueryReturnsFalseWhenUnavailable proves the native default (no
// media-query source) is a stable false.
func TestUseMediaQueryReturnsFalseWhenUnavailable(parseT *testing.T) {
	if UseMediaQuery("(min-width: 768px)") {
		parseT.Fatal("expected false when media queries are unavailable")
	}
}

// TestUseMediaQueryReadsSource proves UseMediaQuery reflects a substituted media
// source's match value at read time.
func TestUseMediaQueryReadsSource(parseT *testing.T) {
	parsePrev := getMediaQuerySource
	defer func() { getMediaQuerySource = parsePrev }()
	getMediaQuerySource = func(string) (mediaQuerySource, error) {
		return fakeMediaSource{matches: true}, nil
	}
	if !UseMediaQuery("(min-width: 768px)") {
		parseT.Fatal("expected true from a matching media source")
	}
}

// TestUseNetworkStatusReflectsSeam proves UseNetworkStatus seeds from the
// readOnlineStatus seam.
func TestUseNetworkStatusReflectsSeam(parseT *testing.T) {
	parsePrev := readOnlineStatus
	defer func() { readOnlineStatus = parsePrev }()

	readOnlineStatus = func() bool { return false }
	if UseNetworkStatus() {
		parseT.Fatal("expected offline when seam reports false")
	}
	readOnlineStatus = func() bool { return true }
	if !UseNetworkStatus() {
		parseT.Fatal("expected online when seam reports true")
	}
}

// TestDefaultReadOnlineStatusNativeAssumesOnline proves the native seam default.
func TestDefaultReadOnlineStatusNativeAssumesOnline(parseT *testing.T) {
	if !defaultReadOnlineStatus() {
		parseT.Fatal("native build should assume online")
	}
}

// TestViewTransitionNativeCallsApply proves the native fallback runs the callback
// directly.
func TestViewTransitionNativeCallsApply(parseT *testing.T) {
	parseCalled := false
	ViewTransition(func() { parseCalled = true })
	if !parseCalled {
		parseT.Fatal("ViewTransition must call apply on native")
	}
	// nil apply must not panic.
	ViewTransition(nil)
}

// TestUseMountNativeDoesNotPanic proves UseMount is a safe no-op on native (no
// effects run; fn is not invoked).
func TestUseMountNativeDoesNotPanic(parseT *testing.T) {
	parseRan := false
	UseMount(func() func() {
		parseRan = true
		return nil
	})
	if parseRan {
		parseT.Fatal("native effects do not run, so UseMount fn must not execute")
	}
}

// TestElementHooksNativeReturnZeroValues proves the element-scoped hooks degrade
// to safe zero values on native/SSR.
func TestElementHooksNativeReturnZeroValues(parseT *testing.T) {
	var parseRef DOMRef

	if parseRect := UseElementGeometry(parseRef); parseRect != (interop.Rect{}) {
		parseT.Fatalf("expected zero Rect on native, got %+v", parseRect)
	}
	if UseIntersection(parseRef) {
		parseT.Fatal("expected false intersection on native")
	}
	// These must not panic.
	UseAnimationRestart(parseRef, "anim")
	UsePointerEvents(parseRef, PointerHandlers{OnPointerDown: func(Event) {}})
	UseWheel(parseRef, func(Event) {})
}

// TestCurrentMediaMatchNativeFalse proves the shared media read returns false
// natively.
func TestCurrentMediaMatchNativeFalse(parseT *testing.T) {
	if currentMediaMatch("(prefers-color-scheme: dark)") {
		parseT.Fatal("expected false on native")
	}
}
