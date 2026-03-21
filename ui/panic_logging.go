package ui

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func actionableCreateElementPanic(summary string) string {
	return runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
		Source:  "ui",
		Subject: "ui.CreateElement",
		Message: summary,
		Path:    "ui.CreateElement",
	})
}

func actionableUnsupportedOnServerPanic(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "API"
	}
	return runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
		Source:  "ui",
		Subject: "ui." + trimmed,
		Message: unsupportedOnServerMessage(trimmed),
		Path:    "ui." + trimmed,
	})
}

func unsupportedOnServerMessage(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "API"
	}
	return fmt.Sprintf("ui.%s is not available on non-js/wasm builds in the current SSR slice", trimmed)
}
