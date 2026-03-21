package diagnostics

import (
	"net/http"

	internaldiagnostics "github.com/monstercameron/GoWebComponents/internal/diagnostics"
)

type Report = internaldiagnostics.Report

type Options = internaldiagnostics.Options

func Build(options Options) Report {
	return internaldiagnostics.Build(options)
}

func Emit(report Report) {
	internaldiagnostics.Emit(report)
}

func WriteHTTPError(w http.ResponseWriter, status int, report Report) {
	internaldiagnostics.WriteHTTPError(w, status, report)
}
