package ui

import "github.com/monstercameron/GoWebComponents/v4/internal/runtime"

// WASMStackFrame describes one wasm/browser panic frame (function, source file, line) for
// stack-correlation mapping. It is the public alias of the runtime frame type.
type WASMStackFrame = runtime.WASMStackFrame

// SetWASMStackFrameMapper installs a mapper that translates an observed wasm/browser panic frame
// back to a higher-signal application Go source location. The runtime threads every panic frame
// through it before the in-page error overlay and the devtools panel render it, so a panic shows
// your Go function/file/line instead of an opaque wasm offset.
//
// This is the public C4 "stack-correlation workaround" — the supported bridge until the Go
// toolchain emits browser-consumable source maps for js/wasm natively. Install one at startup
// (sourced from your build's symbol dump or a hand-maintained table for hot paths).
func SetWASMStackFrameMapper(parseMapper func(WASMStackFrame) (WASMStackFrame, bool)) {
	runtime.SetWASMStackFrameMapper(parseMapper)
}

// ResetWASMStackFrameMapper clears the installed wasm stack-frame mapper.
func ResetWASMStackFrameMapper() {
	runtime.ResetWASMStackFrameMapper()
}
