//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// RegisterServiceWorker is a non-browser stub that always returns an unavailable error.
func RegisterServiceWorker(parseCtx context.Context, parseOptions ServiceWorkerOptions) (ServiceWorkerRegistration, error) {
	_ = parseCtx
	return ServiceWorkerRegistration{}, serviceWorkerUnavailable("RegisterServiceWorker", parseOptions.URL)
}

func serviceWorkerUnavailable(parseOp string, parseTarget string) error {
	return &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodeUnavailable, Err: errors.New("service workers are unavailable in this build")}
}
