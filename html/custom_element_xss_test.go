package html

import "testing"

// TestCustomElementDropsMarkupSinkProperties pins that CustomElement refuses to
// forward markup-injection sink properties (innerHTML / outerHTML /
// insertAdjacentHTML) to the reconciled element. Previously any Properties entry
// was passed through verbatim as `__gwc_prop__:<name>`, so a component could set
// element.innerHTML to attacker-controlled markup — an XSS sink that bypasses the
// framework's normal text-escaping. Safe properties must still pass through.
func TestCustomElementDropsMarkupSinkProperties(parseT *testing.T) {
	parseNode := CustomElement("demo-widget", CustomElementProps{
		Properties: map[string]any{
			"innerHTML":          "<img src=x onerror=alert(1)>",
			"outerHTML":          "<script>alert(1)</script>",
			"insertAdjacentHTML": "<b>x</b>",
			"score":              7,
		},
	})
	if parseNode == nil {
		parseT.Fatal("expected custom element node")
	}
	for _, parseName := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML"} {
		if _, parseOk := parseNode.Props[customElementPropertyPrefix+parseName]; parseOk {
			parseT.Fatalf("markup-sink property %q must not be forwarded to the element", parseName)
		}
	}
	if parseNode.Props[customElementPropertyPrefix+"score"] != 7 {
		parseT.Fatalf("expected safe property score=7 to pass through, got %#v", parseNode.Props[customElementPropertyPrefix+"score"])
	}
}

// TestIsUnsafeCustomElementProperty pins the denylist itself (case- and
// whitespace-insensitive) so the guard cannot silently regress.
func TestIsUnsafeCustomElementProperty(parseT *testing.T) {
	for _, parseBad := range []string{"innerHTML", "innerhtml", " outerHTML ", "INSERTADJACENTHTML"} {
		if !isUnsafeCustomElementProperty(parseBad) {
			parseT.Fatalf("expected %q to be rejected", parseBad)
		}
	}
	for _, parseOk := range []string{"score", "config", "innerText", "textContent"} {
		if isUnsafeCustomElementProperty(parseOk) {
			parseT.Fatalf("expected %q to be allowed", parseOk)
		}
	}
}
