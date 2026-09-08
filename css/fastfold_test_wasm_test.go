//go:build js && wasm

package css

import "strings"

type fastFoldTestSink struct{ parseCSS strings.Builder }

var fastFoldSink *fastFoldTestSink

// Emit records CSS emitted by the fold tests without requiring a browser DOM.
func (parseSink *fastFoldTestSink) Emit(_ string, parseCSS string) {
	parseSink.parseCSS.WriteString(parseCSS)
}

// resetFastFoldTest resets the registry and installs a deterministic Wasm sink.
func resetFastFoldTest() { Reset(); fastFoldSink = &fastFoldTestSink{}; SetSink(fastFoldSink) }

// getFastFoldStyleBlock reads CSS emitted by the installed Wasm test sink.
func getFastFoldStyleBlock() string {
	if fastFoldSink == nil {
		return ""
	}
	return fastFoldSink.parseCSS.String()
}
