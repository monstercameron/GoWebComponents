// Fuzzing is unsupported on js/wasm, and the wasm-under-node runner cannot even
// read the seed corpus directory on Windows (O_DIRECTORY unsupported) — the
// seeds execute on the native build.
//go:build !(js && wasm)

package html_test

import (
	"regexp"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

var (
	reMDScriptOrIframe = regexp.MustCompile(`(?i)<\s*(script|iframe)`)
	reMDEventHandler   = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*["']`)
	reMDJSURL          = regexp.MustCompile(`(?i)(href|src)\s*=\s*["']\s*javascript:`)
)

// FuzzRenderMarkdownSanitize fuzzes markdown rendering for XSS: markdown commonly
// embeds raw HTML and link/image URLs, so the rendered output must never contain
// a <script>/<iframe> tag, a quoted on*= handler, or a javascript: href/src — for
// any input, without panicking.
func FuzzRenderMarkdownSanitize(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "# hi", "<script>x()</script>", "[x](javascript:alert(1))",
		"![img](javascript:alert(1))", "<img src=x onerror=alert(1)>",
		"normal **bold** text", "<iframe src=//evil>", "[a](<javascript:1>)",
		"\n\n<script src=//evil></script>\n\n", "<a href='javascript:x'>y</a>",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		parseNodes := html.RenderMarkdown(parseIn)
		parseOut, parseErr := ui.RenderToString(html.Div(html.Props{}, parseNodes...))
		if parseErr != nil {
			return
		}
		if reMDScriptOrIframe.MatchString(parseOut) {
			parseT.Fatalf("markdown leaked a script/iframe tag for %q -> %q", parseIn, parseOut)
		}
		if reMDEventHandler.MatchString(parseOut) {
			parseT.Fatalf("markdown leaked an on*= handler for %q -> %q", parseIn, parseOut)
		}
		if reMDJSURL.MatchString(parseOut) {
			parseT.Fatalf("markdown leaked a javascript: URL for %q -> %q", parseIn, parseOut)
		}
	})
}
