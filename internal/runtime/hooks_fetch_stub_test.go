//go:build !js || !wasm
// +build !js !wasm

package runtime

import (
	"reflect"
	"testing"
)

func TestGoUseFetchStub_ReturnsUnsupportedState(t *testing.T) {
	getter, refetch := GoUseFetch("/api/test")

	state := getter()
	if state.Loading {
		t.Fatalf("expected non-loading stub state, got %+v", state)
	}
	if state.Data != nil {
		t.Fatalf("expected nil data for unsupported stub state, got %#v", state.Data)
	}
	if state.Error != "fetch API unavailable in this environment" {
		t.Fatalf("expected unsupported environment error, got %q", state.Error)
	}

	refetch()
	stateAfter := getter()
	if stateAfter != state {
		t.Fatalf("expected no-op refetch to preserve stub state, before=%+v after=%+v", state, stateAfter)
	}
}

func TestGoUseFetchStub_ReusesSharedFunctions(t *testing.T) {
	getter1, refetch1 := GoUseFetch("/api/one")
	getter2, refetch2 := GoUseFetch("/api/two", "ignored")

	if getter1 == nil || getter2 == nil || refetch1 == nil || refetch2 == nil {
		t.Fatal("expected non-nil stub functions")
	}
	if getter1() != getter2() {
		t.Fatalf("expected shared unsupported state across stub calls")
	}
	if reflect.ValueOf(getter1).Pointer() != reflect.ValueOf(getter2).Pointer() {
		t.Fatalf("expected shared getter implementation")
	}
	if reflect.ValueOf(refetch1).Pointer() != reflect.ValueOf(refetch2).Pointer() {
		t.Fatalf("expected shared refetch implementation")
	}
}
