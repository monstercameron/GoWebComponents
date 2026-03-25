//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// ObserveInstallability is a non-browser stub that always returns an unavailable error.
func ObserveInstallability(parseOptions InstallabilityOptions) (InstallabilityManager, error) {
	_ = parseOptions
	return InstallabilityManager{}, installabilityUnavailable("ObserveInstallability", "window")
}

func installabilityUnavailable(parseOp string, parseTarget string) error {
	return &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodeUnavailable, Err: errors.New("installability helpers are unavailable in this build")}
}
