//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func TestWASMDOMNodeNullHelpersTreatTypedNilAndNullWrapperAsEquivalent(parseT *testing.T) {
	var parseTypedNil runtime.DOMNode = (*WASMDOMNode)(nil)
	parseNullWrapper := runtime.DOMNode(&WASMDOMNode{value: js.Null()})
	parseConcrete := runtime.DOMNode(&WASMDOMNode{value: js.Global().Get("Object").New()})

	if !runtime.IsDOMNodeNull(parseTypedNil) {
		parseT.Fatal("expected typed nil wasm DOM node to report null")
	}
	if !runtime.IsDOMNodeNull(parseNullWrapper) {
		parseT.Fatal("expected null js.Value wrapper to report null")
	}
	if runtime.IsDOMNodeNull(parseConcrete) {
		parseT.Fatal("expected concrete wasm DOM node to report non-null")
	}
	if !runtime.IsSameDOMNode(parseTypedNil, parseNullWrapper) {
		parseT.Fatal("expected typed nil and null-wrapper wasm DOM nodes to compare equal")
	}
}

func TestWASMDOMNodeEqualsUsesUnderlyingJSIdentity(parseT *testing.T) {
	parseValue := js.Global().Get("Object").New()
	parseSame := &WASMDOMNode{value: parseValue}
	parseAlsoSame := &WASMDOMNode{value: parseValue}
	parseDifferent := &WASMDOMNode{value: js.Global().Get("Object").New()}

	if !parseSame.Equals(parseAlsoSame) {
		parseT.Fatal("expected equal wrappers for the same underlying js value")
	}
	if parseSame.Equals(parseDifferent) {
		parseT.Fatal("expected different js values to compare unequal")
	}
}
