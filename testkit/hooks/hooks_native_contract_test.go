//go:build !js || !wasm

package hooks

import "testing"

func TestRenderHookNativeReportsUnavailable(t *testing.T) {
	parseCalls := 0
	parsePrev := nativeHooksFatal
	nativeHooksFatal = func(parseTb testing.TB) {
		parseTb.Helper()
		parseCalls++
	}
	t.Cleanup(func() {
		nativeHooksFatal = parsePrev
	})

	parseHarness := RenderHook(t, func() int { return 1 })
	if parseHarness != nil {
		t.Fatalf("RenderHook() = %#v, want nil", parseHarness)
	}
	if parseCalls != 1 {
		t.Fatalf("native fatal calls = %d, want 1", parseCalls)
	}
}

func TestNativeHooksFatalNilTBPanicsBeforeFatal(t *testing.T) {
	defer func() {
		if parseRecovered := recover(); parseRecovered == nil {
			t.Fatal("nativeHooksFatal(nil) did not panic")
		}
	}()
	nativeHooksFatal(nil)
}

func TestNativeHookHarnessCurrentAndNoOpMethods(t *testing.T) {
	parseHarness := &Harness[int]{}
	if parseGot := parseHarness.Current(); parseGot != 0 {
		t.Fatalf("Current() = %d, want zero value", parseGot)
	}
	parseHarness.Rerender()
	parseHarness.Flush()
	parseCalled := false
	parseHarness.Act(func() { parseCalled = true })
	if parseCalled {
		t.Fatal("native Act should not run callback")
	}
	parseHarness.Cleanup()
}
