package pwa

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestRegisterServiceWorkerReportsUnavailableOnNativeBuilds(t *testing.T) {
	_, err := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js"})
	if !interop.IsCode(err, interop.CodeUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestServiceWorkerRegistrationRegisterSyncReportsUnavailableOnNativeBuilds(t *testing.T) {
	err := (ServiceWorkerRegistration{}).RegisterSync(context.Background(), "offline-demo-replay")
	if !interop.IsCode(err, interop.CodeUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
