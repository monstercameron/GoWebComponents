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
func SetWASMStackFrameMapper(parseMapper WASMStackFrameMapper) {
	wasmStackFrameMapper.mu.Lock()
	defer wasmStackFrameMapper.mu.Unlock()
	wasmStackFrameMapper.mapper = parseMapper
}

// ResetWASMStackFrameMapper clears the current wasm stack-frame mapper.
func ResetWASMStackFrameMapper() {
	SetWASMStackFrameMapper(nil)
}

// translateWASMStackFrame is a core package helper.
func translateWASMStackFrame(parseFrame panicFrame) panicFrame {
	wasmStackFrameMapper.mu.RLock()
	parseMapper := wasmStackFrameMapper.mapper
	wasmStackFrameMapper.mu.RUnlock()
	if parseMapper == nil {
		return parseFrame
	}
	parseMapped, parseOk := parseMapper(WASMStackFrame{
		Function: parseFrame.Function,
		File:     parseFrame.File,
		Line:     parseFrame.Line,
	})
	if !parseOk {
		return parseFrame
	}
	if parseMapped.Function == "" {
		parseMapped.Function = parseFrame.Function
	}
	if parseMapped.File == "" {
		parseMapped.File = parseFrame.File
	}
	if parseMapped.Line == 0 {
		parseMapped.Line = parseFrame.Line
	}
	return panicFrame{
		Function: parseMapped.Function,
		File:     parseMapped.File,
		Line:     parseMapped.Line,
	}
}
