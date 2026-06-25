//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func rawAttrNode(parseName string, parseValue any) ui.Node {
	return html.Tag("div", html.Props{Raw: map[string]any{parseName: parseValue}}, ui.Text("x"))
}

// TestAttributeSerializationSemantics codifies the universal attribute-rendering
// invariants (the framework-agnostic subset of React's
// ReactDOMServerIntegrationAttributes suite): boolean coercion, nil omission,
// numeric stringification, and HTML-escaping of values so a crafted value can
// never break out of the quoted attribute or inject markup.
func TestAttributeSerializationSemantics(t *testing.T) {
	cases := []struct {
		name     string
		attr     string
		value    any
		contains string // substring the output MUST contain
		absent   string // substring the output must NOT contain ("" = skip)
	}{
		{"bool-true-bare", "hidden", true, "<div hidden>", `hidden="`},
		{"bool-false-omitted", "hidden", false, "<div>x</div>", "hidden"},
		{"nil-omitted", "data-x", nil, "<div>x</div>", "data-x"},
		{"int", "tabindex", 3, `tabindex="3"`, ""},
		{"float", "data-x", 1.5, `data-x="1.5"`, ""},
		{"quote-escaped", "title", `a" onmouseover="x`, `title="a&#34; onmouseover=&#34;x"`, ""},
		{"angle-escaped", "title", `a><b`, `title="a&gt;&lt;b"`, ""},
		{"amp-escaped", "title", `a&b`, `title="a&amp;b"`, ""},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(rawAttrNode(parseCase.attr, parseCase.value))
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if !strings.Contains(out, parseCase.contains) {
			t.Errorf("%s: expected %q in %q", parseCase.name, parseCase.contains, out)
		}
		if parseCase.absent != "" && strings.Contains(out, parseCase.absent) {
			t.Errorf("%s: did not expect %q in %q", parseCase.name, parseCase.absent, out)
		}
	}
}

// TestAttributeNameValidationRejectsInjection mirrors React finding #57: an
// attribute name that does not conform to the HTML name production (contains a
// space, angle bracket, quote, equals, or is empty) is dropped entirely rather
// than emitted, so a crafted name cannot inject a second attribute or markup.
func TestAttributeNameValidationRejectsInjection(t *testing.T) {
	badNames := []string{"a b", "a<b", `a"b`, "a>b", "a=b", "a'b", "", "a\tb", "a\nb"}
	for _, parseName := range badNames {
		out, err := ui.RenderToString(rawAttrNode(parseName, "payload"))
		if err != nil {
			t.Fatalf("name %q: render error: %v", parseName, err)
		}
		if strings.Contains(out, "payload") {
			t.Errorf("name %q: invalid attribute name was emitted: %q", parseName, out)
		}
		if out != "<div>x</div>" {
			t.Errorf("name %q: expected bare div, got %q", parseName, out)
		}
	}
}
