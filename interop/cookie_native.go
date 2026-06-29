//go:build !js || !wasm

package interop

// readRawCookies is a non-browser stub that returns an unavailable error.
func readRawCookies() (string, error) {
	return "", unavailable("readRawCookies", "document.cookie")
}

// writeRawCookie is a non-browser stub that returns an unavailable error.
func writeRawCookie(parseSerialized string) error {
	_ = parseSerialized
	return unavailable("writeRawCookie", "document.cookie")
}
