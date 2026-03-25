package diagnostics

import (
	"net/http"

	internaldiagnostics "github.com/monstercameron/GoWebComponents/internal/diagnostics"
)

type Report = internaldiagnostics.Report

type Options = internaldiagnostics.Options

// NewReport builds a diagnostics Report from the given options.
func NewReport(options Options) Report {
	return internaldiagnostics.Build(options)
}

// Emit dispatches a diagnostics report to all registered listeners.
func Emit(report Report) {
	internaldiagnostics.Emit(report)
}

// WriteHTTPError writes the report as an HTTP error response with the given status code.
func WriteHTTPError(w http.ResponseWriter, status int, report Report) {
	internaldiagnostics.WriteHTTPError(w, status, report)
}
