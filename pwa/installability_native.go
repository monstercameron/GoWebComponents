//go:build !js || !wasm

package pwa

import (
	"errors"

	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// ObserveInstallability is a non-browser stub that always returns an unavailable error.
func ObserveInstallability(parseInstallOptions InstallabilityOptions) (InstallabilityManager, error) {
	_ = parseInstallOptions
	return InstallabilityManager{}, installabilityUnavailable("ObserveInstallability", "window")
}

func installabilityUnavailable(parseInstallOp string, parseInstallTarget string) error {
	return &interop.Error{Op: parseInstallOp, Target: parseInstallTarget, Code: interop.CodeUnavailable, Err: errors.New("installability helpers are unavailable in this build")}
}
