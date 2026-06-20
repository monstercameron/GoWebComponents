//go:build !(js && wasm)

package html_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestBindSetsValueFromState(t *testing.T) {
	parseComponent := func(struct{}) ui.Node {
		parseName := ui.UseState("hello")
		return html.Input(html.PropsOf(html.Bind(parseName), html.ID("field")))
	}
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(parseComponent, struct{}{}))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `value="hello"`) {
		t.Fatalf("Bind did not set value from state:\n%s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `id="field"`) {
		t.Fatalf("other props should still apply:\n%s", parseMarkup)
	}
}
