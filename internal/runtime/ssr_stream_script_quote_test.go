package runtime

import (
	"regexp"
	"testing"
)

// TestStreamBoundaryPatchScriptClosesItsAttributes verifies the streamed replacement script opens with a
// well-formed tag, so a browser executes the body instead of reading it as attribute text.
func TestStreamBoundaryPatchScriptClosesItsAttributes(parseT *testing.T) {
	parseTag := regexp.MustCompile(`^<template data-gwc-stream-boundary="[^"]*">.*</template><script data-gwc-stream-boundary="[^"]*"( nonce="[^"]*")?>\(function\(\)\{`)
	for _, parseNonce := range []string{"", "abc123"} {
		parsePatch := renderSSRStreamBoundaryPatch("gwc-stream-1", "<p>ready</p>", true, parseNonce)
		if !parseTag.MatchString(parsePatch) {
			parseT.Fatalf("nonce %q: script open tag is malformed: %.200s", parseNonce, parsePatch)
		}
	}
}
