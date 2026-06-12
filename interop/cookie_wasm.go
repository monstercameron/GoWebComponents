//go:build js && wasm
// +build js,wasm

package interop

import "syscall/js"

// readRawCookies returns the raw document.cookie string from the browser.
func readRawCookies() (string, error) {
	return js.Global().Get("document").Get("cookie").String(), nil
}

// writeRawCookie assigns a serialized Set-Cookie string to document.cookie.
func writeRawCookie(parseSerialized string) error {
	js.Global().Get("document").Set("cookie", parseSerialized)
	return nil
}
