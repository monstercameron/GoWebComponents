package ui

import (
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
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
