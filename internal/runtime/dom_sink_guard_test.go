package runtime

import (
	"reflect"
	"strings"
	"testing"
)

// TestDOMAdapterHasNoRawHTMLSink is a security guard. The DOM adapter interface
// is the only DOM surface the reconciler can reach, so it must expose no
// raw-HTML setter (SetInnerHTML / innerHTML / outerHTML). The reconciler builds
// the tree through CreateElement, CreateTextNode, SetTextContent, and
// SetAttribute, which insert untrusted content as text or attribute values and
// never parse it as markup. If a raw-HTML method is reintroduced to the
// interface, this test fails so the addition is reviewed deliberately instead of
// silently reopening an XSS sink on the render path.
func TestDOMAdapterHasNoRawHTMLSink(parseT *testing.T) {
	parseType := reflect.TypeFor[DOMAdapter]()
	for method := range parseType.Methods() {
		parseName := method.Name
		parseLower := strings.ToLower(parseName)
		if strings.Contains(parseLower, "innerhtml") || strings.Contains(parseLower, "outerhtml") ||
			(strings.HasPrefix(parseLower, "set") && strings.Contains(parseLower, "html")) {
			parseT.Fatalf("DOMAdapter exposes raw-HTML sink %q; route untrusted content through the text/attribute APIs instead, or justify the sink and update this guard", parseName)
		}
	}
}
