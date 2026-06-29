//go:build !js

package diagnostics

import "net/http"

// WriteHTTPError emits the report and writes it as a plain-text HTTP error response.
//
// This lives behind !js so browser/wasm builds never link net/http: that import
// chain (http → crypto/tls → x509 → dnsmessage) costs roughly a megabyte of
// wasm for a helper only servers can call.
func WriteHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	Emit(parseReport)
	parseW.Header().Set("Content-Type", "text/plain; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	_, _ = parseW.Write([]byte(parseReport.Formatted()))
}
