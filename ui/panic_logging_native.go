//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// actionableUnsupportedOnServerPanic is a core package helper.
func actionableUnsupportedOnServerPanic(parseName string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" {
		parseTrimmed = "API"
	}
	return runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
		Source:  "ui",
		Subject: "ui." + parseTrimmed,
		Message: unsupportedOnServerMessage(parseTrimmed),
		Path:    "ui." + parseTrimmed,
	})
}

// unsupportedOnServerMessage is a core package helper.
func unsupportedOnServerMessage(parseName string) string {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" {
		parseTrimmed = "API"
	}
	return fmt.Sprintf("ui.%s is not available on non-js/wasm builds in the current SSR slice", parseTrimmed)
}
