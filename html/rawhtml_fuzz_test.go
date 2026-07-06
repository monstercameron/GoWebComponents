// Fuzzing is unsupported on js/wasm, and the wasm-under-node runner cannot even
// read the seed corpus directory on Windows (O_DIRECTORY unsupported) — the
// seeds execute on the native build.
//go:build !(js && wasm)

package html_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Precise breakout oracles. Escaped text content cannot produce these — a real
// <script>/<iframe> tag or a quoted on*=/javascript: attribute would have to
// survive sanitization as actual markup.
var (
	reScriptOrIframe = regexp.MustCompile(`(?i)<\s*(script|iframe)`)
	reEventHandler   = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*["']`)
	reJSURL          = regexp.MustCompile(`(?i)=\s*["']\s*javascript:`)
)

// FuzzRawHTMLSanitize fuzzes the sanitized RawHTML path: for ANY input markup,
// the rendered output must never contain an executable vector — a <script>/<iframe>
// tag, a quoted on*= event-handler attribute, or a javascript: attribute URL —
// and must never panic.
func FuzzRawHTMLSanitize(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "<p>hi</p>", "<script>x()</script>", "<img src=x onerror=alert(1)>",
		"<a href=\"javascript:alert(1)\">x</a>", "<svg/onload=alert(1)>",
		"<b>bold</b><script src=//evil></script>", "<iframe src=javascript:1>",
		"<p onclick=\"x\">t</p>", "<!-- --><script>", "onerror=0", "<a href='javascript:x'>",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		parseNodes := html.RawHTML(parseIn)
		parseOut, parseErr := ui.RenderToString(html.Div(html.Props{}, parseNodes...))
		if parseErr != nil {
			return
		}
		if reScriptOrIframe.MatchString(parseOut) {
			parseT.Fatalf("sanitized RawHTML leaked a script/iframe tag for %q -> %q", parseIn, parseOut)
		}
		if reEventHandler.MatchString(parseOut) {
			parseT.Fatalf("sanitized RawHTML leaked an on*= handler for %q -> %q", parseIn, parseOut)
		}
		if reJSURL.MatchString(parseOut) {
			parseT.Fatalf("sanitized RawHTML leaked a javascript: URL for %q -> %q", parseIn, parseOut)
		}
		_ = strings.TrimSpace
	})
}
