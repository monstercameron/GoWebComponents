//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// ObserveInstallability is a non-browser stub that always returns an unavailable error.
func ObserveInstallability(options InstallabilityOptions) (InstallabilityManager, error) {"ObserveInstallability", "window")
}

func installabilityUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("installability helpers are unavailable in this build")}
}
