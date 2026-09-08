package app

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestRenderMessageMarkup keeps formatting as real nodes and rejects executable markup.
func TestRenderMessageMarkup(parseT *testing.T) {
	setMessageMarkupTestRuntime(parseT)
	parseOutput, parseErr := ui.RenderToString(renderMessageMarkup(`<p>Hello <strong>world</strong></p><pre><code>one &lt; two</code></pre><script>alert(1)</script><img src="x" onerror="alert(2)"><a href="javascript:alert(3)">link</a>`))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	for _, parseWant := range []string{"<strong>world</strong>", "<pre><code>one &lt; two</code></pre>", "prose", `data-gwc-code="true"`, `type="button"`, ">Copy</button>"} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Errorf("missing %q in %s", parseWant, parseOutput)
		}
	}
	for _, parseUnsafe := range []string{"<script", "onerror", "javascript:", "innerHTML", "alert("} {
		if strings.Contains(parseOutput, parseUnsafe) {
			parseT.Errorf("unsafe markup %q in %s", parseUnsafe, parseOutput)
		}
	}
}
