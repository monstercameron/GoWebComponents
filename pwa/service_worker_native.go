//go:build !js || !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// RegisterServiceWorker is a non-browser stub that always returns an unavailable error.
func RegisterServiceWorker(parseServiceCtx context.Context, parseServiceOptions ServiceWorkerOptions) (ServiceWorkerRegistration, error) {
	_ = parseServiceCtx
	return ServiceWorkerRegistration{}, serviceWorkerUnavailable("RegisterServiceWorker", parseServiceOptions.URL)
}

func serviceWorkerUnavailable(parseServiceOp string, parseServiceTarget string) error {
	return &interop.Error{Op: parseServiceOp, Target: parseServiceTarget, Code: interop.CodeUnavailable, Err: errors.New("service workers are unavailable in this build")}
}
