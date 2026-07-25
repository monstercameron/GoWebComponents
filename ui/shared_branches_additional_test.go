//go:build !js || !wasm

package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// TestSharedContextAndBoundaryAdditionalBranches covers remaining shared extraction branches for context and error boundaries.
func TestSharedContextAndBoundaryAdditionalBranches(parseT *testing.T) {
	type parseContextNoValue struct {
		Label string
	}
	type parseBoundaryMapProps struct {
		Label string
	}

	if parseValue, parseOk := extractContextProviderValue((*ContextProviderProps[string])(nil)); parseOk || parseValue != nil {
		parseT.Fatalf("extractContextProviderValue(nil pointer) = %#v, %t; want nil,false", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := extractContextProviderValue(parseContextNoValue{Label: "x"}); parseOk2 || parseValue2 != nil {
		parseT.Fatalf("extractContextProviderValue(no Value field) = %#v, %t; want nil,false", parseValue2, parseOk2)
	}

	parseContext := CreateContext("light")
	parseProviderNode := createContextProviderElement(parseContext.Provider, 42)
	if parseProviderNode == nil {
		parseT.Fatal("createContextProviderElement(non-struct props) should create a node")
	}
	if parseProviderNode.Props["value"] != 42 {
		parseT.Fatalf("expected non-struct provider props to map to value=42, got %#v", parseProviderNode.Props)
	}
	if parseChildren := extractContextProviderChildren((*ContextProviderProps[string])(nil)); parseChildren != nil {
		parseT.Fatalf("extractContextProviderChildren(nil pointer) = %#v, want nil", parseChildren)
	}

	parseFallback := Text("fallback")
	parseErrorFallback := func(parseErr error, parseReset func()) Node { return Text(parseErr.Error()) }
	parseOnError := func(error) {}
	parseMapProps := map[string]any{
		"Fallback":      parseFallback,
		"ErrorFallback": parseErrorFallback,
		"OnError":       parseOnError,
		"ResetKeys":     []any{"v3"},
		"Child":         Text("child"),
		"Children":      []Node{Text("child-2"), nil, Text("child-3")},
	}

	if parseGot, parseOk := extractErrorBoundaryFallback(parseMapProps); !parseOk || parseGot != parseFallback {
		parseT.Fatalf("extractErrorBoundaryFallback(upper map) = %#v, %t; want fallback,true", parseGot, parseOk)
	}
	if parseGot2, parseOk2 := extractErrorBoundaryErrorFallback(parseMapProps); !parseOk2 || parseGot2 == nil {
		parseT.Fatalf("extractErrorBoundaryErrorFallback(upper map) = %T, %t; want function,true", parseGot2, parseOk2)
	}
	if parseGot3, parseOk3 := extractErrorBoundaryOnError(parseMapProps); !parseOk3 || parseGot3 == nil {
		parseT.Fatalf("extractErrorBoundaryOnError(upper map) = %T, %t; want function,true", parseGot3, parseOk3)
	}
	if parseKeys := extractErrorBoundaryResetKeys(parseMapProps); len(parseKeys) != 1 || parseKeys[0] != "v3" {
		parseT.Fatalf("extractErrorBoundaryResetKeys(upper map) = %#v, want [v3]", parseKeys)
	}
	if parseChildren2 := extractErrorBoundaryChildren(parseMapProps); len(parseChildren2) != 3 {
		parseT.Fatalf("extractErrorBoundaryChildren(upper map) len = %d, want 3", len(parseChildren2))
	}
	if parseGot4, parseOk4 := extractErrorBoundaryFallback((*ErrorBoundaryProps)(nil)); parseOk4 || parseGot4 != nil {
		parseT.Fatalf("extractErrorBoundaryFallback(nil pointer) = %#v, %t; want nil,false", parseGot4, parseOk4)
	}
	if parseGot5, parseOk5 := extractErrorBoundaryOnError(parseBoundaryMapProps{Label: "x"}); parseOk5 || parseGot5 != nil {
		parseT.Fatalf("extractErrorBoundaryOnError(no field) returned ok=%t, want nil,false", parseOk5)
	}
	if parseKeys2 := extractErrorBoundaryResetKeys(123); parseKeys2 != nil {
		parseT.Fatalf("extractErrorBoundaryResetKeys(non-struct) = %#v, want nil", parseKeys2)
	}
	if parseValue3 := dereferenceStructValue(123); parseValue3.IsValid() {
		parseT.Fatalf("dereferenceStructValue(non-struct) = %+v, want invalid", parseValue3)
	}
}

// TestSharedBootstrapOverlayAndComponentAdditionalBranches covers remaining lazy-branch, bootstrap, overlay, and component identity helpers.
func TestSharedBootstrapOverlayAndComponentAdditionalBranches(parseT *testing.T) {
	isParseFalseCalled := false
	parseFalseNode := If(false, nil, func() Node {
		isParseFalseCalled = true
		return Text("false")
	})
	if parseFalseNode == nil || parseFalseNode.TextContent != "false" || !isParseFalseCalled {
		parseT.Fatalf("If(false, falseBranch) = %#v called=%t; want false branch node", parseFalseNode, isParseFalseCalled)
	}
	if parseGot := If(true, nil); parseGot != nil {
		parseT.Fatalf("If(true,nil) = %#v, want nil", parseGot)
	}

	isParseDefaultCalled := false
	if parseGot2 := Match().When(true, nil).Default(func() Node {
		isParseDefaultCalled = true
		return Text("default")
	}); parseGot2 != nil || isParseDefaultCalled {
		parseT.Fatalf("Match().When(true,nil).Default() = %#v defaultCalled=%t; want nil,false", parseGot2, isParseDefaultCalled)
	}
	if parseGot3 := Match().Default(func() Node { return Text("default") }); parseGot3 == nil || parseGot3.TextContent != "default" {
		parseT.Fatalf("Match().Default() = %#v, want default node", parseGot3)
	}

	type parseComponentValue struct {
		Label string
	}
	if parsePretty, parseQualified := describeComponentIdentity(nil); parsePretty != "" || parseQualified != "" {
		parseT.Fatalf("describeComponentIdentity(nil) = (%q,%q), want empty strings", parsePretty, parseQualified)
	}
	parseExpectedType := reflect.TypeFor[parseComponentValue]().String()
	if parsePretty2, parseQualified2 := describeComponentIdentity(parseComponentValue{}); parsePretty2 != parseExpectedType || parseQualified2 != parseExpectedType {
		parseT.Fatalf("describeComponentIdentity(non-func) = (%q,%q), want (%q,%q)", parsePretty2, parseQualified2, parseExpectedType, parseExpectedType)
	}

	if _, parseErr := normalizeSSRBootstrapVersion(-1); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported SSR bootstrap version") {
		parseT.Fatalf("normalizeSSRBootstrapVersion(-1) error = %v, want unsupported-version error", parseErr)
	}
	if _, parseErr2 := normalizeSSRBootstrapVersion(CurrentSSRBootstrapVersion + 1); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unsupported SSR bootstrap version") {
		parseT.Fatalf("normalizeSSRBootstrapVersion(high) error = %v, want unsupported-version error", parseErr2)
	}
	if parsePayload, parseErr3 := UnmarshalSSRBootstrap(nil); parseErr3 != nil || parsePayload.Version != CurrentSSRBootstrapVersion {
		parseT.Fatalf("UnmarshalSSRBootstrap(nil) = %+v, %v; want normalized empty payload", parsePayload, parseErr3)
	}
	if parsePayload2, parseErr4 := UnmarshalSSRBootstrapBinary(nil); parseErr4 != nil || parsePayload2.Version != CurrentSSRBootstrapVersion {
		parseT.Fatalf("UnmarshalSSRBootstrapBinary(nil) = %+v, %v; want normalized empty payload", parsePayload2, parseErr4)
	}
	if parseRef, parseErr5 := UnmarshalSSRBootstrapReference(nil); parseErr5 != nil || parseRef.Version != CurrentSSRBootstrapVersion || parseRef.Format != SSRBootstrapFormatJSON {
		parseT.Fatalf("UnmarshalSSRBootstrapReference(nil) = %+v, %v; want normalized empty reference", parseRef, parseErr5)
	}
	if parseScript, parseErr6 := RenderBootstrapReferenceScript(SSRBootstrapReference{URL: "/bootstrap.json"}, ""); parseErr6 != nil || !strings.Contains(parseScript, `"format":"json"`) {
		parseT.Fatalf("RenderBootstrapReferenceScript(default format) = %q, %v; want default json format", parseScript, parseErr6)
	}

	parseManager := newOverlayStackManager()
	parseStack := parseManager.snapshot("missing", overlayManagerRegistration{Kind: ""}, false)
	if parseStack.Depth != -1 || parseStack.Kind != OverlayKindCustom || parseStack.BackdropZIndex != 1000 || parseStack.SurfaceZIndex != 1001 {
		parseT.Fatalf("overlay snapshot without fallback = %+v, want missing/default stack", parseStack)
	}
	if parseGot4 := overlayTopMatchID(nil, func(parseEntry overlayManagerRegistration) bool { return parseEntry.CloseOnEscape }); parseGot4 != "" {
		parseT.Fatalf("overlayTopMatchID(nil) = %q, want empty", parseGot4)
	}
	if normalizeOverlayKind("") != OverlayKindCustom {
		parseT.Fatalf("normalizeOverlayKind(\"\") = %q, want %q", normalizeOverlayKind(""), OverlayKindCustom)
	}
	if normalizeOverlayBaseZIndex(0) != 1000 {
		parseT.Fatalf("normalizeOverlayBaseZIndex(0) = %d, want 1000", normalizeOverlayBaseZIndex(0))
	}

	parseNotified := make(chan struct{}, 1)
	parseManager.notify([]overlaySubscriber{
		{id: 0, notify: func() { parseT.Fatal("subscriber id 0 should not notify") }},
		{id: 1, notify: func() { parseNotified <- struct{}{} }},
	})
	select {
	case <-parseNotified:
	case <-time.After(2 * time.Second):
		parseT.Fatal("expected overlay subscriber notification")
	}

	parseManager.upsert(overlayManagerRegistration{ID: "dialog", Kind: OverlayKindDialog, BaseZIndex: 1000})
	parseUnexpectedNotify := make(chan struct{}, 1)
	parseManager.subscribers = []overlaySubscriber{{id: 1, notify: func() { parseUnexpectedNotify <- struct{}{} }}}
	parseManager.upsert(overlayManagerRegistration{ID: "dialog", Kind: OverlayKindDialog, BaseZIndex: 1000})
	select {
	case <-parseUnexpectedNotify:
		parseT.Fatal("unchanged overlay upsert should not notify subscribers")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestBuildComponentRendererSpecializationsAndFallbacks(parseT *testing.T) {
	parseFunc := func() Node { return Text("identity") }
	parsePretty, parseQualified := describeComponentIdentity(parseFunc)
	parsePrettyAgain, parseQualifiedAgain := describeComponentIdentity(parseFunc)
	if parsePretty == "" || parseQualified == "" || parsePretty != parsePrettyAgain || parseQualified != parseQualifiedAgain {
		parseT.Fatalf("describeComponentIdentity(func) unstable: (%q,%q) then (%q,%q)", parsePretty, parseQualified, parsePrettyAgain, parseQualifiedAgain)
	}

	parseNoArg := buildComponentRenderer(func() Node { return Text("no-arg") })
	if parseGot := parseNoArg(func() Node { return Text("fresh") }, nil); parseGot.TextContent != "fresh" {
		parseT.Fatalf("no-arg renderer = %#v", parseGot)
	}
	assertCreateElementPanic(parseT, func() { _ = parseNoArg("wrong", nil) })

	parseMapRenderer := buildComponentRenderer(func(map[string]any) Node { return Text("unused") })
	parseMapGot := parseMapRenderer(func(parseProps map[string]any) Node {
		return Text(fmt.Sprint(parseProps["label"]))
	}, map[string]any{propsKey: map[string]any{"label": "map"}})
	if parseMapGot.TextContent != "map" {
		parseT.Fatalf("map renderer = %#v", parseMapGot)
	}

	parseAttrsRenderer := buildComponentRenderer(func(runtime.Attrs) Node { return Text("unused") })
	parseAttrsGot := parseAttrsRenderer(func(parseProps runtime.Attrs) Node {
		return Text(fmt.Sprint(parseProps["label"]))
	}, map[string]any{propsKey: runtime.Attrs{"label": "attrs"}})
	if parseAttrsGot.TextContent != "attrs" {
		parseT.Fatalf("attrs renderer = %#v", parseAttrsGot)
	}

	type typedProps struct {
		Label string
	}
	parseTypedComponent := func(parseProps typedProps) Node { return Text(parseProps.Label) }
	parseTypedRenderer := buildComponentRenderer(parseTypedComponent)
	parseTypedGot := parseTypedRenderer(parseTypedComponent, map[string]any{propsKey: typedProps{Label: "typed"}})
	if parseTypedGot == nil || parseTypedGot.TextContent != "typed" {
		parseT.Fatalf("typed renderer = %#v", parseTypedGot)
	}
	parseTypedRendererCached := buildComponentRenderer(func(typedProps) Node { return Text("cached") })
	if reflect.ValueOf(parseTypedRenderer).Pointer() != reflect.ValueOf(parseTypedRendererCached).Pointer() {
		parseT.Fatal("component renderer should be cached by component function type")
	}

	parseNilComponent := func() Node { return nil }
	if parseGot := buildComponentRenderer(parseNilComponent)(parseNilComponent, nil); parseGot != nil {
		parseT.Fatalf("nil component result = %#v", parseGot)
	}
	if parseRenderer := buildComponentRenderer(nil); parseRenderer != nil {
		parseT.Fatalf("nil component renderer type = %T", parseRenderer)
	}

	parseFallbackRenderer := buildComponentRenderer("not-a-function")
	assertCreateElementPanic(parseT, func() { _ = parseFallbackRenderer("not-a-function", nil) })
}

func assertCreateElementPanic(parseT *testing.T, parseFn func()) {
	parseT.Helper()
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil || !strings.Contains(fmt.Sprint(parseRecovered), "ui.CreateElement requires a component function") {
			parseT.Fatalf("expected actionable create element panic, got %#v", parseRecovered)
		}
	}()
	parseFn()
}
