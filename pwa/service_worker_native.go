//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// RegisterServiceWorker is a non-browser stub that always returns an unavailable error.
func RegisterServiceWorker(ctx context.Context, options ServiceWorkerOptions) (ServiceWorkerRegistration, error) {"RegisterServiceWorker", options.URL)
}

func serviceWorkerUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("service workers are unavailable in this build")}
}
