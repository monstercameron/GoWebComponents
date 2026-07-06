//go:build !js

package diagnostics

import "net/http"

// WriteHTTPError emits the report to the server log and writes the CLIENT-SAFE
// subset (Report.FormattedPublic — summary/code/next/docs only) as a plain-text
// HTTP error response. It deliberately does NOT send stack frames, absolute build
// paths (which disclose the OS username / server filesystem layout), or the
// where/path/error/runtime detail to the client — that lives in the server log
// via Emit. Use WriteHTTPErrorVerbose only for a trusted/dev diagnostics endpoint
// where the full report in the response body is intended (#60).
//
// This lives behind !js so browser/wasm builds never link net/http: that import
// chain (http → crypto/tls → x509 → dnsmessage) costs roughly a megabyte of
// wasm for a helper only servers can call.
func WriteHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	writeHTTPError(parseW, parseStatus, parseReport, false)
}

// WriteHTTPErrorVerbose is the opt-in full-disclosure variant: it writes the
// complete Formatted() report (including stack frames and build paths) to the
// HTTP client. Use it ONLY on trusted/local diagnostics endpoints; never on a
// public error path.
func WriteHTTPErrorVerbose(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	writeHTTPError(parseW, parseStatus, parseReport, true)
}

func writeHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report, parseVerbose bool) {
	Emit(parseReport) // full report always goes to the server log (stderr)
	parseW.Header().Set("Content-Type", "text/plain; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	parseBody := parseReport.FormattedPublic()
	if parseVerbose {
		parseBody = parseReport.Formatted()
	}
	_, _ = parseW.Write([]byte(parseBody))
}
