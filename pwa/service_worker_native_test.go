//go:build !js || !wasm

package pwa

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

func TestRegisterServiceWorkerReportsUnavailableOnNativeBuilds(parseT *testing.T) {
	_, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js"})
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable error, got %v", parseErr)
	}
}

func TestServiceWorkerRegistrationRegisterSyncReportsUnavailableOnNativeBuilds(parseT *testing.T) {
	parseErr := (ServiceWorkerRegistration{}).RegisterSync(context.Background(), "offline-demo-replay")
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable error, got %v", parseErr)
	}
}
