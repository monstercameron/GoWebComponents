package diagnostics

import (
	"net/http"

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

// WriteHTTPError writes the report as an HTTP error response with the given status code.
func WriteHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	internaldiagnostics.WriteHTTPError(parseW, parseStatus, parseReport)
}
