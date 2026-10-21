//go:build !(js && wasm)

package html

import "testing"

// TestParseInlineStyleKeepsSemicolonsInsideValues pins that a ';' inside a url()
// or quoted string does not split the declaration.
func TestParseInlineStyleKeepsSemicolonsInsideValues(parseT *testing.T) {
	parseStyle := parseInlineStyle(`background:url(data:image/png;base64,AAA); content:"a;b"; color:red`)
	if parseStyle["background"] != "url(data:image/png;base64,AAA)" {
		parseT.Fatalf("background = %q", parseStyle["background"])
	}
	if parseStyle["content"] != `"a;b"` || parseStyle["color"] != "red" || len(parseStyle) != 3 {
		parseT.Fatalf("style = %#v", parseStyle)
	}
}
