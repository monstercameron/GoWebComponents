//go:build js && wasm

package interop

import "syscall/js"

// readRawCookies returns the raw document.cookie string from the browser. It
// returns an unavailable error (rather than panicking) when there is no document
// — e.g. inside a Web Worker or any no-DOM wasm host — so callers degrade
// gracefully instead of crashing on a Value.Get against undefined.
func readRawCookies() (string, error) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return "", unavailable("readRawCookies", "document.cookie")
	}
	parseCookie := parseDocument.Get("cookie")
	if parseCookie.Type() != js.TypeString {
		return "", nil
	}
	return parseCookie.String(), nil
}

// writeRawCookie assigns a serialized Set-Cookie string to document.cookie. Like
// readRawCookies it returns an unavailable error when no document is present
// rather than panicking.
func writeRawCookie(parseSerialized string) error {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return unavailable("writeRawCookie", "document.cookie")
	}
	parseDocument.Set("cookie", parseSerialized)
	return nil
}
