package runtime

import "sync"

// WASMStackFrame describes one parsed wasm/browser panic frame before formatting.
type WASMStackFrame struct {
	Function string
	File     string
	Line     int
}

// WASMStackFrameMapper maps one observed wasm/browser panic frame back to a
// higher-signal application-owned location when source-map metadata is available.
type WASMStackFrameMapper func(WASMStackFrame) (WASMStackFrame, bool)

var wasmStackFrameMapper struct {
	mu     sync.RWMutex
	mapper WASMStackFrameMapper
}

// SetWASMStackFrameMapper installs the current wasm stack-frame mapper.
func SetWASMStackFrameMapper(mapper WASMStackFrameMapper) {
	wasmStackFrameMapper.mu.Lock()
	defer wasmStackFrameMapper.mu.Unlock()
	wasmStackFrameMapper.mapper = mapper
}

// ResetWASMStackFrameMapper clears the current wasm stack-frame mapper.
func ResetWASMStackFrameMapper() {
	SetWASMStackFrameMapper(nil)
}

func translateWASMStackFrame(frame panicFrame) panicFrame {
	wasmStackFrameMapper.mu.RLock()
	mapper := wasmStackFrameMapper.mapper
	wasmStackFrameMapper.mu.RUnlock()
	if mapper == nil {
		return frame
	}
	mapped, ok := mapper(WASMStackFrame{
		Function: frame.Function,
		File:     frame.File,
		Line:     frame.Line,
	})
	if !ok {
		return frame
	}
	if mapped.Function == "" {
		mapped.Function = frame.Function
	}
	if mapped.File == "" {
		mapped.File = frame.File
	}
	if mapped.Line == 0 {
		mapped.Line = frame.Line
	}
	return panicFrame{
		Function: mapped.Function,
		File:     mapped.File,
		Line:     mapped.Line,
	}
}
