//go:build !js || !wasm

package ui

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// actionableUnsupportedOnServerPanic is a core package helper.
func actionableUnsupportedOnServerPanic(parseAPIName string) string {
	parseAPINameTrimmed := strings.TrimSpace(parseAPIName)
	if parseAPINameTrimmed == "" {
		parseAPINameTrimmed = "API"
	}
	return runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
		Source:  "ui",
		Subject: "ui." + parseAPINameTrimmed,
		Message: unsupportedOnServerMessage(parseAPINameTrimmed),
		Path:    "ui." + parseAPINameTrimmed,
	})
}

// unsupportedOnServerMessage is a core package helper.
func unsupportedOnServerMessage(parseAPIName string) string {
	parseAPINameTrimmed := strings.TrimSpace(parseAPIName)
	if parseAPINameTrimmed == "" {
		parseAPINameTrimmed = "API"
	}
	return fmt.Sprintf("ui.%s is not available on non-js/wasm builds in the current SSR slice", parseAPINameTrimmed)
}
