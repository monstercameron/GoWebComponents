//go:build !js

package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type namedInputFn func(string)

func TestTypedInputConstructorsAcceptEachShape(parseT *testing.T) {
	var parseNil func(string)
	parseProps := PropsOf(
		OnInput(func() {}),
		OnChange(func(string) {}),
	)
	if parseProps.OnInput.Value() == nil || parseProps.OnChange.Value() == nil {
		parseT.Fatal("plain func shapes must register handlers")
	}
	if PropsOf(OnInput(func(ui.Event) {})).OnInput.Value() == nil {
		parseT.Fatal("func(ui.Event) must register a handler")
	}
	parseNamed := PropsOf(OnInput(namedInputFn(func(string) {}))).OnInput.Value()
	if _, isPlain := parseNamed.(func(string)); !isPlain {
		parseT.Fatalf("named func type must normalize to func(string), got %T", parseNamed)
	}
	if PropsOf(OnInput(parseNil)).OnInput.Value() != nil {
		parseT.Fatal("a nil func must register no handler")
	}
	parseHandler := ui.WrapHandler(func() {})
	parseProps = PropsOf(OnInputHandler(parseHandler), OnChangeHandler(parseHandler))
	if parseProps.OnInput.Value() == nil || parseProps.OnChange.Value() == nil {
		parseT.Fatal("handler pass-throughs must register the handler")
	}
}
