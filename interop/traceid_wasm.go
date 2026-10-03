//go:build js && wasm

package interop

import "syscall/js"

// fillRandomBytes fills parseBuf with crypto.getRandomValues, which is capped at 65536 bytes per call.
func fillRandomBytes(parseOp string, parseBuf []byte) error {
	parseCrypto, parseErr := globalProperty(parseOp, "crypto")
	if parseErr != nil {
		return parseErr
	}
	if !parseCrypto.Get("getRandomValues").Truthy() || len(parseBuf) > 65536 {
		return unavailable(parseOp, "crypto.getRandomValues")
	}
	parseArray := js.Global().Get("Uint8Array").New(len(parseBuf))
	parseCrypto.Call("getRandomValues", parseArray)
	js.CopyBytesToGo(parseBuf, parseArray)
	return nil
}
