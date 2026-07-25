package diagnostics

import (
	internaldiagnostics "github.com/monstercameron/GoWebComponents/v5/internal/diagnostics"
)

type Report = internaldiagnostics.Report

type Options = internaldiagnostics.Options

// NewReport builds a diagnostics Report from the given options.
func NewReport(parseOptions Options) Report {
	return internaldiagnostics.Build(parseOptions)
}

// Emit writes a formatted diagnostics report to stderr. (There is no listener/
// sink registry — the report is rendered synchronously to os.Stderr.)
func Emit(parseReport Report) {
	internaldiagnostics.Emit(parseReport)
}
