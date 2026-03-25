package ui

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// actionableCreateElementPanic is a core package helper.
func actionableCreateElementPanic(parsePanicSummary string) string {
	return runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
		Source:  "ui",
		Subject: "ui.CreateElement",
		Message: parsePanicSummary,
		Path:    "ui.CreateElement",
	})
}
