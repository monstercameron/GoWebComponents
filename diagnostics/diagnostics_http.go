//go:build !js

package diagnostics

import (
	"net/http"

	internaldiagnostics "github.com/monstercameron/GoWebComponents/internal/diagnostics"
)

// WriteHTTPError writes the report as an HTTP error response with the given
// status code.  Server-only (!js) so wasm builds never link net/http.
func WriteHTTPError(parseW http.ResponseWriter, parseStatus int, parseReport Report) {
	internaldiagnostics.WriteHTTPError(parseW, parseStatus, parseReport)
}
