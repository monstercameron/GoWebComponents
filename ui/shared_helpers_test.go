package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestContextProviderHelperExtraction(parseT *testing.T) {
	parseContext := CreateContext("light")
	parseChild := Text("child")
	parseProps := ContextProviderProps[string]{
		Value:    "dark",
		Child:    parseChild,
		Children: []Node{Text("extra")},
	}

	if parseValue, parseOk := extractContextProviderValue(parseProps); !parseOk || parseValue.(string) != "dark" {
		parseT.Fatalf("extractContextProviderValue(struct) = %#v, %t; want dark,true", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := extractContextProviderValue(&parseProps); !parseOk2 || parseValue2.(string) != "dark" {
		parseT.Fatalf("extractContextProviderValue(pointer) = %#v, %t; want dark,true", parseValue2, parseOk2)
	}
	if parseValue3, parseOk3 := extractContextProviderValue(map[string]interface{}{"value": "map-dark"}); !parseOk3 || parseValue3.(string) != "map-dark" {
		parseT.Fatalf("extractContextProviderValue(map) = %#v, %t; want map-dark,true", parseValue3, parseOk3)
	}
	if parseValue4, parseOk4 := extractContextProviderValue(42); !parseOk4 || parseValue4.(int) != 42 {
		parseT.Fatalf("extractContextProviderValue(non-struct) = %#v, %t; want 42,true", parseValue4, parseOk4)
	}

	parseChildren := extractContextProviderChildren(parseProps)
	if len(parseChildren) != 2 {
		parseT.Fatalf("extractContextProviderChildren(struct) len = %d, want 2", len(parseChildren))
	}
	parseMapChildren := extractContextProviderChildren(map[string]interface{}{"child": parseChild, "children": []interface{}{Text("map")}})
	if len(parseMapChildren) != 2 {
		parseT.Fatalf("extractContextProviderChildren(map) len = %d, want 2", len(parseMapChildren))
	}
	if createContextProviderElement((*ContextProvider[string])(nil), parseProps) != nil {
		parseT.Fatal("createContextProviderElement(nil provider) should return nil")
	}
	if parseNode := createContextProviderElement(parseContext.Provider, parseProps); parseNode == nil {
		parseT.Fatal("createContextProviderElement() should create a provider node")
	}
	if parseGot := castContextValue[string]("value"); parseGot != "value" {
		parseT.Fatalf("castContextValue() = %q, want value", parseGot)
	}
	if parseGot2 := castContextValue[int]("wrong"); parseGot2 != 0 {
		parseT.Fatalf("castContextValue(mismatch) = %d, want 0", parseGot2)
	}
}

func TestErrorBoundaryHelperExtraction(parseT *testing.T) {
	parseFallback := Text("fallback")
	parseErrorFallback := func(parseErr error, reset func()) Node {
		return Text(parseErr.Error())
	}
	parseOnError := func(error) {}
	parseStructProps := ErrorBoundaryProps{
		Fallback:      parseFallback,
		ErrorFallback: parseErrorFallback,
		OnError:       parseOnError,
		ResetKeys:     []interface{}{"v1"},
		Child:         Text("child"),
		Children:      []Node{Text("child-2")},
	}
	parseMapProps := map[string]interface{}{
		"fallback":      parseFallback,
		"errorFallback": parseErrorFallback,
		"onError":       parseOnError,
		"resetKeys":     []interface{}{"v2"},
		"child":         Text("child"),
		"children":      []interface{}{Text("child-2")},
	}

	if parseGot, parseOk := extractErrorBoundaryFallback(parseStructProps); !parseOk || parseGot != parseFallback {
		parseT.Fatalf("extractErrorBoundaryFallback(struct) = %#v, %t; want fallback,true", parseGot, parseOk)
	}
	if parseGot2, parseOk2 := extractErrorBoundaryFallback(parseMapProps); !parseOk2 || parseGot2 != parseFallback {
		parseT.Fatalf("extractErrorBoundaryFallback(map) = %#v, %t; want fallback,true", parseGot2, parseOk2)
	}
	if parseGot3, parseOk3 := extractErrorBoundaryErrorFallback(parseStructProps); !parseOk3 || parseGot3 == nil {
		parseT.Fatalf("extractErrorBoundaryErrorFallback(struct) = %T, %t; want function,true", parseGot3, parseOk3)
	}
	if parseGot4, parseOk4 := extractErrorBoundaryOnError(parseMapProps); !parseOk4 || parseGot4 == nil {
		parseT.Fatalf("extractErrorBoundaryOnError(map) = %T, %t; want function,true", parseGot4, parseOk4)
	}
	if parseKeys := extractErrorBoundaryResetKeys(parseStructProps); len(parseKeys) != 1 || parseKeys[0] != "v1" {
		parseT.Fatalf("extractErrorBoundaryResetKeys(struct) = %#v, want [v1]", parseKeys)
	}
	if parseChildren := extractErrorBoundaryChildren(parseMapProps); len(parseChildren) != 2 {
		parseT.Fatalf("extractErrorBoundaryChildren(map) len = %d, want 2", len(parseChildren))
	}
	if parseNode := createErrorBoundaryElement(ErrorBoundary, parseStructProps); parseNode == nil {
		parseT.Fatal("createErrorBoundaryElement() should create a boundary node")
	}
	if createErrorBoundaryElement((*errorBoundaryComponent)(nil), parseStructProps) != nil {
		parseT.Fatal("createErrorBoundaryElement(nil boundary) should return nil")
	}
	if !dereferenceStructValue(&parseStructProps).IsValid() || dereferenceStructValue((*ErrorBoundaryProps)(nil)).IsValid() {
		parseT.Fatal("dereferenceStructValue() returned unexpected validity state")
	}
	if _, parseOk5 := mapBoundaryNode(map[string]interface{}{}, "Fallback"); parseOk5 {
		parseT.Fatal("mapBoundaryNode() should not find missing nodes")
	}
	if _, parseOk6 := mapBoundaryFallback(map[string]interface{}{}, "ErrorFallback"); parseOk6 {
		parseT.Fatal("mapBoundaryFallback() should not find missing functions")
	}
	if _, parseOk7 := mapBoundaryOnError(map[string]interface{}{}, "OnError"); parseOk7 {
		parseT.Fatal("mapBoundaryOnError() should not find missing functions")
	}
}

func TestBranchingAndHotReloadFallbackHelpers(parseT *testing.T) {
	if Component(func() Node { return Text("helper") }) == nil {
		parseT.Fatal("Component() should delegate to CreateElement")
	}
	if parseGot := If(false, nil); parseGot != nil {
		parseT.Fatalf("If(false,nil) = %#v, want nil", parseGot)
	}
	if parseGot2 := Match().Default(nil); parseGot2 != nil {
		parseT.Fatalf("Match().Default(nil) = %#v, want nil", parseGot2)
	}
	parseKey := hotReloadBoundaryKey([]interface{}{func() {}})
	if !strings.HasPrefix(parseKey, "__gwc_hotreload_boundary__:") || !strings.Contains(parseKey, "func(") {
		parseT.Fatalf("hotReloadBoundaryKey(fallback) = %q, want stringified fallback key", parseKey)
	}
	parseNode := ErrorBoundaryProps{
		ErrorFallback: func(parseErr2 error, reset func()) Node { return Text("caught:" + parseErr2.Error()) },
		Child: CreateElement(func() Node {
			panic(errors.New("boom"))
		}),
	}
	parseMarkup, parseErr := RenderToString(CreateElement(ErrorBoundary, parseNode))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ErrorBoundary) error = %v", parseErr)
	}
	if parseMarkup != "caught:boom" {
		parseT.Fatalf("ErrorBoundary fallback markup = %q, want caught:boom", parseMarkup)
	}
}
