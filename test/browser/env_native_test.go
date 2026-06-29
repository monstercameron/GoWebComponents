//go:build !js || !wasm

package browser

import "testing"

func TestInstallNativeReportsUnavailable(t *testing.T) {
	parseCalls := 0
	parsePrev := nativeBrowserInstallFatal
	nativeBrowserInstallFatal = func(parseTb testing.TB) {
		parseTb.Helper()
		parseCalls++
	}
	t.Cleanup(func() {
		nativeBrowserInstallFatal = parsePrev
	})

	parseEnv := Install(t, Options{Path: "/demo", HashRouting: true})
	if parseEnv != nil {
		t.Fatalf("Install() = %#v, want nil native environment", parseEnv)
	}
	if parseCalls != 1 {
		t.Fatalf("native fatal calls = %d, want 1", parseCalls)
	}
}

func TestNativeBrowserInstallFatalNilTBPanicsBeforeFatal(t *testing.T) {
	defer func() {
		if parseRecovered := recover(); parseRecovered == nil {
			t.Fatal("nativeBrowserInstallFatal(nil) did not panic")
		}
	}()
	nativeBrowserInstallFatal(nil)
}
