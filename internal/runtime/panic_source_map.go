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
func SetWASMStackFrameMapper(parseFrameMapper WASMStackFrameMapper) {
	wasmStackFrameMapper.mu.Lock()
	defer wasmStackFrameMapper.mu.Unlock()
	wasmStackFrameMapper.mapper = parseFrameMapper
}

// ResetWASMStackFrameMapper clears the current wasm stack-frame mapper.
func ResetWASMStackFrameMapper() {
	SetWASMStackFrameMapper(nil)
}

// translateWASMStackFrame is a core package helper.
func translateWASMStackFrame(parsePanicFrame panicFrame) panicFrame {
	wasmStackFrameMapper.mu.RLock()
	parseFrameMapper := wasmStackFrameMapper.mapper
	wasmStackFrameMapper.mu.RUnlock()
	if parseFrameMapper == nil {
		return parsePanicFrame
	}
	parseMappedFrame, parseMappedOK := parseFrameMapper(WASMStackFrame(parsePanicFrame))
	if !parseMappedOK {
		return parsePanicFrame
	}
	if parseMappedFrame.Function == "" {
		parseMappedFrame.Function = parsePanicFrame.Function
	}
	if parseMappedFrame.File == "" {
		parseMappedFrame.File = parsePanicFrame.File
	}
	if parseMappedFrame.Line == 0 {
		parseMappedFrame.Line = parsePanicFrame.Line
	}
	return panicFrame(parseMappedFrame)
}
