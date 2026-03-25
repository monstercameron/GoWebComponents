//go:build !js || !wasm
// +build !js !wasm

package runtime

import (
	"reflect"
	"testing"
)

func TestGoUseFetchStub_ReturnsUnsupportedState(parseT *testing.T) {
	parseGetter, parseRefetch := GoUseFetch("/api/test")

	parseState := parseGetter()
	if parseState.Loading {
		parseT.Fatalf("expected non-loading stub state, got %+v", parseState)
	}
	if parseState.Data != nil {
		parseT.Fatalf("expected nil data for unsupported stub state, got %#v", parseState.Data)
	}
	if parseState.Error != "fetch API unavailable in this environment" {
		parseT.Fatalf("expected unsupported environment error, got %q", parseState.Error)
	}

	parseRefetch()
	parseStateAfter := parseGetter()
	if parseStateAfter != parseState {
		parseT.Fatalf("expected no-op refetch to preserve stub state, before=%+v after=%+v", parseState, parseStateAfter)
	}
}

func TestGoUseFetchStub_ReusesSharedFunctions(parseT *testing.T) {
	parseGetter1, parseRefetch1 := GoUseFetch("/api/one")
	parseGetter2, parseRefetch2 := GoUseFetch("/api/two", "ignored")

	if parseGetter1 == nil || parseGetter2 == nil || parseRefetch1 == nil || parseRefetch2 == nil {
		parseT.Fatal("expected non-nil stub functions")
	}
	if parseGetter1() != parseGetter2() {
		parseT.Fatalf("expected shared unsupported state across stub calls")
	}
	if reflect.ValueOf(parseGetter1).Pointer() != reflect.ValueOf(parseGetter2).Pointer() {
		parseT.Fatalf("expected shared getter implementation")
	}
	if reflect.ValueOf(parseRefetch1).Pointer() != reflect.ValueOf(parseRefetch2).Pointer() {
		parseT.Fatalf("expected shared refetch implementation")
	}
}
