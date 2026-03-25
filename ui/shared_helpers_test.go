package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestContextProviderHelperExtraction(t *testing.T) {
	context := CreateContext("light")
	child := Text("child")
	props := ContextProviderProps[string]{
		Value:    "dark",
		Child:    child,
		Children: []Node{Text("extra")},
	}

	if value, ok := extractContextProviderValue(props); !ok || value.(string) != "dark" {
		t.Fatalf("extractContextProviderValue(struct) = %#v, %t; want dark,true", value, ok)
	}
	if value, ok := extractContextProviderValue(&props); !ok || value.(string) != "dark" {
		t.Fatalf("extractContextProviderValue(pointer) = %#v, %t; want dark,true", value, ok)
	}
	if value, ok := extractContextProviderValue(map[string]interface{}{"value": "map-dark"}); !ok || value.(string) != "map-dark" {
		t.Fatalf("extractContextProviderValue(map) = %#v, %t; want map-dark,true", value, ok)
	}
	if value, ok := extractContextProviderValue(42); !ok || value.(int) != 42 {
		t.Fatalf("extractContextProviderValue(non-struct) = %#v, %t; want 42,true", value, ok)
	}

	children := extractContextProviderChildren(props)
	if len(children) != 2 {
		t.Fatalf("extractContextProviderChildren(struct) len = %d, want 2", len(children))
	}
	mapChildren := extractContextProviderChildren(map[string]interface{}{"child": child, "children": []interface{}{Text("map")}})
	if len(mapChildren) != 2 {
		t.Fatalf("extractContextProviderChildren(map) len = %d, want 2", len(mapChildren))
	}
	if createContextProviderElement((*ContextProvider[string])(nil), props) != nil {
		t.Fatal("createContextProviderElement(nil provider) should return nil")
	}
	if node := createContextProviderElement(context.Provider, props); node == nil {
		t.Fatal("createContextProviderElement() should create a provider node")
	}
	if got := castContextValue[string]("value"); got != "value" {
		t.Fatalf("castContextValue() = %q, want value", got)
	}
	if got := castContextValue[int]("wrong"); got != 0 {
		t.Fatalf("castContextValue(mismatch) = %d, want 0", got)
	}
}

func TestErrorBoundaryHelperExtraction(t *testing.T) {
	fallback := Text("fallback")
	errorFallback := func(err error, reset func()) Node {
		return Text(err.Error())
	}
	onError := func(error) {}
	structProps := ErrorBoundaryProps{
		Fallback:      fallback,
		ErrorFallback: errorFallback,
		OnError:       onError,
		ResetKeys:     []interface{}{"v1"},
		Child:         Text("child"),
		Children:      []Node{Text("child-2")},
	}
	mapProps := map[string]interface{}{
		"fallback":      fallback,
		"errorFallback": errorFallback,
		"onError":       onError,
		"resetKeys":     []interface{}{"v2"},
		"child":         Text("child"),
		"children":      []interface{}{Text("child-2")},
	}

	if got, ok := extractErrorBoundaryFallback(structProps); !ok || got != fallback {
		t.Fatalf("extractErrorBoundaryFallback(struct) = %#v, %t; want fallback,true", got, ok)
	}
	if got, ok := extractErrorBoundaryFallback(mapProps); !ok || got != fallback {
		t.Fatalf("extractErrorBoundaryFallback(map) = %#v, %t; want fallback,true", got, ok)
	}
	if got, ok := extractErrorBoundaryErrorFallback(structProps); !ok || got == nil {
		t.Fatalf("extractErrorBoundaryErrorFallback(struct) = %T, %t; want function,true", got, ok)
	}
	if got, ok := extractErrorBoundaryOnError(mapProps); !ok || got == nil {
		t.Fatalf("extractErrorBoundaryOnError(map) = %T, %t; want function,true", got, ok)
	}
	if keys := extractErrorBoundaryResetKeys(structProps); len(keys) != 1 || keys[0] != "v1" {
		t.Fatalf("extractErrorBoundaryResetKeys(struct) = %#v, want [v1]", keys)
	}
	if children := extractErrorBoundaryChildren(mapProps); len(children) != 2 {
		t.Fatalf("extractErrorBoundaryChildren(map) len = %d, want 2", len(children))
	}
	if node := createErrorBoundaryElement(ErrorBoundary, structProps); node == nil {
		t.Fatal("createErrorBoundaryElement() should create a boundary node")
	}
	if createErrorBoundaryElement((*errorBoundaryComponent)(nil), structProps) != nil {
		t.Fatal("createErrorBoundaryElement(nil boundary) should return nil")
	}
	if !dereferenceStructValue(&structProps).IsValid() || dereferenceStructValue((*ErrorBoundaryProps)(nil)).IsValid() {
		t.Fatal("dereferenceStructValue() returned unexpected validity state")
	}
	if _, ok := mapBoundaryNode(map[string]interface{}{}, "Fallback"); ok {
		t.Fatal("mapBoundaryNode() should not find missing nodes")
	}
	if _, ok := mapBoundaryFallback(map[string]interface{}{}, "ErrorFallback"); ok {
		t.Fatal("mapBoundaryFallback() should not find missing functions")
	}
	if _, ok := mapBoundaryOnError(map[string]interface{}{}, "OnError"); ok {
		t.Fatal("mapBoundaryOnError() should not find missing functions")
	}
}

func TestBranchingAndHotReloadFallbackHelpers(t *testing.T) {
	if Component(nativeZeroArgComponent) == nil {
		t.Fatal("Component() should delegate to CreateElement")
	}
	if got := If(false, nil); got != nil {
		t.Fatalf("If(false,nil) = %#v, want nil", got)
	}
	if got := Match().Default(nil); got != nil {
		t.Fatalf("Match().Default(nil) = %#v, want nil", got)
	}
	key := hotReloadBoundaryKey([]interface{}{func() {}})
	if !strings.HasPrefix(key, "__gwc_hotreload_boundary__:") || !strings.Contains(key, "func(") {
		t.Fatalf("hotReloadBoundaryKey(fallback) = %q, want stringified fallback key", key)
	}
	node := ErrorBoundaryProps{
		ErrorFallback: func(err error, reset func()) Node { return Text("caught:" + err.Error()) },
		Child: CreateElement(func() Node {
			panic(errors.New("boom"))
		}),
	}
	markup, err := RenderToString(CreateElement(ErrorBoundary, node))
	if err != nil {
		t.Fatalf("RenderToString(ErrorBoundary) error = %v", err)
	}
	if markup != "caught:boom" {
		t.Fatalf("ErrorBoundary fallback markup = %q, want caught:boom", markup)
	}
}
