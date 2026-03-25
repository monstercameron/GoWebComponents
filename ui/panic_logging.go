package ui

import (
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
