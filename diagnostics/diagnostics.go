package diagnostics

import (
	internaldiagnostics "github.com/monstercameron/GoWebComponents/internal/diagnostics"
)

type Report = internaldiagnostics.Report

type Options = internaldiagnostics.Options

// NewReport builds a diagnostics Report from the given options.
func NewReport(parseOptions Options) Report {
	return internaldiagnostics.Build(parseOptions)
}

// Emit dispatches a diagnostics report to all registered listeners.
func Emit(parseReport Report) {
	internaldiagnostics.Emit(parseReport)
}
