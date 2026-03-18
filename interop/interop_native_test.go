//go:build !js || !wasm
// +build !js !wasm

package interop

import "testing"

func TestNativeInteropConstructorsReportUnavailable(t *testing.T) {
	checks := []struct {
		name string
		err  error
	}{
		{name: "LocalStorage", err: func() error { _, err := LocalStorage(); return err }()},
		{name: "SessionStorage", err: func() error { _, err := SessionStorage(); return err }()},
		{name: "WindowLocation", err: func() error { _, err := WindowLocation(); return err }()},
		{name: "WindowHistory", err: func() error { _, err := WindowHistory(); return err }()},
		{name: "NavigatorClipboard", err: func() error { _, err := NavigatorClipboard(); return err }()},
		{name: "WindowEvents", err: func() error { _, err := WindowEvents(); return err }()},
		{name: "DocumentEvents", err: func() error { _, err := DocumentEvents(); return err }()},
		{name: "MatchMedia", err: func() error { _, err := MatchMedia("(prefers-color-scheme: dark)"); return err }()},
		{name: "ImportModule", err: func() error { _, err := ImportModule(nil, "/demo.js"); return err }()},
	}
	for _, check := range checks {
		if !IsCode(check.err, CodeUnavailable) {
			t.Fatalf("%s: expected unavailable error, got %v", check.name, check.err)
		}
	}
}
