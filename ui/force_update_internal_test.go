//go:build !(js && wasm)

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestUseForceUpdateRendersAndReturnsCallable(t *testing.T) {
	parseComponent := func(struct{}) ui.Node {
		parseRerender := ui.UseForceUpdate()
		if parseRerender == nil {
			t.Fatal("UseForceUpdate returned nil")
		}
		return html.Div(html.Props{ID: "ok"}, html.Text("x"))
	}
	if _, parseErr := ui.RenderToString(ui.CreateElement(parseComponent, struct{}{})); parseErr != nil {
		t.Fatalf("render with UseForceUpdate: %v", parseErr)
	}
}
