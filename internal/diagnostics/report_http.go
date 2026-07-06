//go:build !js

package diagnostics

import "net/http"

// WriteHTTPError emits the report and writes it as a plain-text HTTP error response.
//
// This lives behind !js so browser/wasm builds never link net/http: that import
// chain (http → crypto/tls → x509 → dnsmessage) costs roughly a megabyte of
// wasm for a helper only servers can call.
// NOTE (security, #60): this currently writes the FULL Formatted() report — stack
// frames + absolute build paths (which disclose the OS username / server
// filesystem layout) + runtime detail — to the HTTP client. That is a deliberate,
// test-pinned behavior (report_test.go TestWriteHTTPErrorWritesStructuredBody) and
// is fine for a trusted/dev diagnostics endpoint but a disclosure risk on a
// public error path. The client-safe subset is available as Report.FormattedPublic();
// the recommended fix is a gated verbose mode (default client-safe, opt-in full)
// so both consumers are served without silently flipping the pinned default.
func WriteHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	Emit(parseReport)
	parseW.Header().Set("Content-Type", "text/plain; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	_, _ = parseW.Write([]byte(parseReport.Formatted()))
}
