package ui

import (
	"errors"
	goruntime "runtime"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// skipHookCallOutsideComponentOnWasm skips tests that invoke Use* hooks at
// test top level: hooks outside a component panic by contract on the wasm
// build (native stubs mask it), and this environment has no DOM to mount
// into. The selection logic under test is platform-independent, so the
// native run carries the coverage.
func skipHookCallOutsideComponentOnWasm(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "js" {
		t.Skip("Use* hooks are illegal outside a component on the wasm build; covered natively")
	}
}

type fakeMediaQuerySource struct {
	parseMatches bool
	parseSubErr  error
	parseHandler func(interop.MediaQueryEvent)
}

func (parseM *fakeMediaQuerySource) Matches() bool {
	return parseM.parseMatches
}

func (parseM *fakeMediaQuerySource) Subscribe(parseHandler func(interop.MediaQueryEvent)) (interop.Subscription, error) {
	parseM.parseHandler = parseHandler
	if parseM.parseSubErr != nil {
		return interop.Subscription{}, parseM.parseSubErr
	}
	return interop.Subscription{}, nil
}

func withFakeMediaQuerySource(t *testing.T, parseSource mediaQuerySource, parseErr error) {
	t.Helper()
	parsePrev := getMediaQuerySource
	getMediaQuerySource = func(parseQuery string) (mediaQuerySource, error) {
		return parseSource, parseErr
	}
	t.Cleanup(func() {
		getMediaQuerySource = parsePrev
	})
}

type fakeBoolState struct {
	parseValues []bool
}

func (parseS *fakeBoolState) Set(parseValue bool) {
	parseS.parseValues = append(parseS.parseValues, parseValue)
}

func TestPreferenceHooksUnavailableDefaults(t *testing.T) {
	skipHookCallOutsideComponentOnWasm(t)
	withFakeMediaQuerySource(t, nil, errors.New("media unavailable"))
	if currentMediaMatch(prefersDarkSchemeQuery) {
		t.Fatal("currentMediaMatch unavailable = true, want false")
	}
	if UsePrefersReducedMotion() {
		t.Fatal("UsePrefersReducedMotion unavailable = true, want false")
	}
	if parseScheme := UsePrefersColorScheme(); parseScheme != ColorSchemeLight {
		t.Fatalf("UsePrefersColorScheme unavailable = %q, want light", parseScheme)
	}
}

func TestPreferenceHooksReadCurrentMatches(t *testing.T) {
	skipHookCallOutsideComponentOnWasm(t)
	parseSource := &fakeMediaQuerySource{parseMatches: true}
	withFakeMediaQuerySource(t, parseSource, nil)
	if !currentMediaMatch(prefersReducedMotionQuery) {
		t.Fatal("currentMediaMatch matched source = false, want true")
	}
	if !UsePrefersReducedMotion() {
		t.Fatal("UsePrefersReducedMotion matched source = false, want true")
	}
	if parseScheme := UsePrefersColorScheme(); parseScheme != ColorSchemeDark {
		t.Fatalf("UsePrefersColorScheme matched source = %q, want dark", parseScheme)
	}
}

func TestBindMediaQueryMatchBranches(t *testing.T) {
	parseSource := &fakeMediaQuerySource{parseMatches: true}
	withFakeMediaQuerySource(t, parseSource, nil)
	parseState := &fakeBoolState{}

	parseCleanup := bindMediaQueryMatch(prefersDarkSchemeQuery, parseState)
	if parseCleanup == nil {
		t.Fatal("cleanup = nil, want no-op or subscription cleanup")
	}
	if len(parseState.parseValues) != 1 || !parseState.parseValues[0] {
		t.Fatalf("initial subscription state = %#v, want true", parseState.parseValues)
	}
	if parseSource.parseHandler == nil {
		t.Fatal("media query handler was not registered")
	}
	parseSource.parseHandler(interop.MediaQueryEvent{Matches: false, Media: prefersDarkSchemeQuery})
	if len(parseState.parseValues) != 2 || parseState.parseValues[1] {
		t.Fatalf("callback state = %#v, want false update", parseState.parseValues)
	}
	parseCleanup()

	withFakeMediaQuerySource(t, &fakeMediaQuerySource{parseMatches: false, parseSubErr: errors.New("subscribe failed")}, nil)
	if parseCleanup := bindMediaQueryMatch(prefersDarkSchemeQuery, &fakeBoolState{}); parseCleanup == nil {
		t.Fatal("subscribe-error cleanup = nil, want no-op cleanup")
	}

	withFakeMediaQuerySource(t, nil, errors.New("media unavailable"))
	if parseCleanup := bindMediaQueryMatch(prefersDarkSchemeQuery, &fakeBoolState{}); parseCleanup == nil {
		t.Fatal("unavailable cleanup = nil, want no-op cleanup")
	}
}
