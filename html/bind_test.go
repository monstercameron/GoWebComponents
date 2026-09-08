//go:build !(js && wasm)

package html_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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

// TestBindToBindsSignal proves the V4 fine-grained Signal[string] two-way binds
// through BindTo: a Signal satisfies html.Binding and drives the input's value.
func TestBindToBindsSignal(t *testing.T) {
	parseName := state.NewKeyedSignal("test:bind:signal", "world")
	parseMarkup, parseErr := ui.RenderToString(
		html.Input(html.PropsOf(html.BindTo(parseName), html.ID("sigfield"))),
	)
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `value="world"`) {
		t.Fatalf("BindTo did not bind the signal value:\n%s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `id="sigfield"`) {
		t.Fatalf("other props should still apply:\n%s", parseMarkup)
	}
}

// TestBindFuncSetsValueFromGetter proves BindFunc binds through an explicit getter
// (for sources that are not a single handle).
func TestBindFuncSetsValueFromGetter(t *testing.T) {
	parseModel := struct{ Field string }{Field: "from-getter"}
	parseMarkup, parseErr := ui.RenderToString(
		html.Input(html.PropsOf(html.BindFunc(
			func() string { return parseModel.Field },
			func(parseV string) { parseModel.Field = parseV },
		))),
	)
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `value="from-getter"`) {
		t.Fatalf("BindFunc did not bind from the getter:\n%s", parseMarkup)
	}
}

// TestBindToNilTargetRendersNoValue proves a nil target is a safe no-op.
func TestBindToNilTargetRendersNoValue(t *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(html.Input(html.PropsOf(html.BindTo(nil), html.ID("empty"))))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	if strings.Contains(parseMarkup, `value=`) {
		t.Fatalf("nil target should not set a value:\n%s", parseMarkup)
	}
}
