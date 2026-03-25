package html

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type sugarStringer string

func (s sugarStringer) String() string {
	return string(s)
}

func TestChildrenNormalizesMixedInputs(t *testing.T) {
	children := Children(
		"alpha",
		[]string{"beta", "gamma"},
		Span(Props{}, Text("delta")),
		[]interface{}{nil, sugarStringer("epsilon"), 42, func() string { return "zeta" }},
	)

	if len(children) != 7 {
		t.Fatalf("expected 7 normalized children, got %d", len(children))
	}

	if children[0].TextContent != "alpha" {
		t.Fatalf("expected first text child alpha, got %#v", children[0])
	}
	if children[1].TextContent != "beta" {
		t.Fatalf("expected second text child beta, got %#v", children[1])
	}
	if children[2].TextContent != "gamma" {
		t.Fatalf("expected third text child gamma, got %#v", children[2])
	}
	if children[3].Type != "span" {
		t.Fatalf("expected preserved span child, got %#v", children[3].Type)
	}
	if children[4].TextContent != "epsilon" {
		t.Fatalf("expected stringer child epsilon, got %#v", children[4])
	}
	if children[5].TextContent != "42" {
		t.Fatalf("expected numeric child to stringify, got %#v", children[5])
	}
	if children[6].Type != runtime.ReactiveTextNodeType {
		t.Fatalf("expected reactive text child, got %#v", children[6].Type)
	}
}

func TestChildrenWorksWithExistingBuilderExpansion(t *testing.T) {
	node := Div(Props{}, Children("hello", []interface{}{" ", Textf("%s", "world")})...)
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("expected SSR render to succeed, got %v", err)
	}
	if markup != "<div>hello world</div>" {
		t.Fatalf("expected normalized child expansion markup, got %q", markup)
	}
}

func TestTextSupportsFormattedAndReactiveContent(t *testing.T) {
	if Text(nil) != nil {
		t.Fatal("expected Text(nil) to return nil")
	}
	original := Span(Props{}, Text("passthrough"))
	if Text(original) != original {
		t.Fatal("expected Text(ui.Node) to return the original node")
	}
	if Text(true).TextContent != "true" {
		t.Fatalf("expected bool text conversion, got %#v", Text(true))
	}
	if Text(uint16(9)).TextContent != "9" {
		t.Fatalf("expected numeric text conversion, got %#v", Text(uint16(9)))
	}

	formatted := Textf("count:%d", 7)
	if formatted == nil || formatted.TextContent != "count:7" {
		t.Fatalf("expected formatted text node, got %#v", formatted)
	}

	if TextIf(false, "hidden") != nil {
		t.Fatal("expected TextIf(false, ...) to return nil")
	}
	if textIf := TextIf(true, "shown"); textIf == nil || textIf.TextContent != "shown" {
		t.Fatalf("expected TextIf(true, ...) to return a text node, got %#v", textIf)
	}

	reactive := Text(func() string { return "live" })
	if reactive == nil || reactive.Type != runtime.ReactiveTextNodeType {
		t.Fatalf("expected reactive text node, got %#v", reactive)
	}

	markup, err := ui.RenderToString(Div(Props{}, reactive))
	if err != nil {
		t.Fatalf("expected SSR reactive text render, got %v", err)
	}
	if markup != "<div>live</div>" {
		t.Fatalf("expected reactive SSR markup, got %q", markup)
	}
}

func TestChildrenAndClassHelpersHandleEmptyAndArrayInputs(t *testing.T) {
	if Children() != nil {
		t.Fatal("expected Children() to return nil")
	}
	if Children(nil, []interface{}{nil}) != nil {
		t.Fatal("expected Children with only nil values to return nil")
	}

	children := Children([2]int{5, 6})
	if len(children) != 2 || children[0].TextContent != "5" || children[1].TextContent != "6" {
		t.Fatalf("expected reflected array children, got %#v", children)
	}

	if ClassNames() != "" {
		t.Fatal("expected empty ClassNames result")
	}
	classes := ClassNames([2]string{"alpha", "beta"}, 7)
	if classes != "alpha beta 7" {
		t.Fatalf("expected reflected class fragments, got %q", classes)
	}
}

func TestClassHelpersNormalizeWhitespaceAndConditionals(t *testing.T) {
	className := ClassNames(" alpha  beta ", When(false, "hidden"), When(true, "active"), []string{"gamma delta"}, sugarStringer("epsilon"))
	if className != "alpha beta active gamma delta epsilon" {
		t.Fatalf("expected normalized class string, got %q", className)
	}
	if When(false, "missing") != "" {
		t.Fatal("expected false When helper to return empty string")
	}
}

func TestConditionalHelpersSelectExpectedNodes(t *testing.T) {
	trueNode := Span(Props{}, Text("true"))
	falseNode := Span(Props{}, Text("false"))

	if If(false, trueNode) != nil {
		t.Fatal("expected If(false, ...) to return nil")
	}
	if If(true, trueNode) != trueNode {
		t.Fatal("expected If(true, ...) to return provided node")
	}
	if IfElse(true, trueNode, falseNode) != trueNode {
		t.Fatal("expected IfElse(true, ...) to return true branch")
	}
	if IfElse(false, trueNode, falseNode) != falseNode {
		t.Fatal("expected IfElse(false, ...) to return false branch")
	}
	if Unless(true, falseNode) != nil {
		t.Fatal("expected Unless(true, ...) to return nil")
	}
	if Unless(false, falseNode) != falseNode {
		t.Fatal("expected Unless(false, ...) to return provided node")
	}
}

func TestMapPreservesOrder(t *testing.T) {
	nodes := Map([]int{3, 1, 4}, func(value int) ui.Node {
		return Li(Props{}, Text(fmt.Sprintf("item-%d", value)))
	})

	markup, err := ui.RenderToString(Ul(Props{}, nodes...))
	if err != nil {
		t.Fatalf("expected mapped list SSR render, got %v", err)
	}
	if !strings.Contains(markup, "<li>item-3</li><li>item-1</li><li>item-4</li>") {
		t.Fatalf("expected mapped order to be preserved, got %q", markup)
	}
}

func TestSecondPassCollectionHelpers(t *testing.T) {
	if Map([]int{}, func(value int) ui.Node { return Textf("%d", value) }) != nil {
		t.Fatal("expected empty Map result to be nil")
	}
	if MapKeyed([]int{}, func(value int) interface{} { return value }, func(value int) ui.Node { return Textf("%d", value) }) != nil {
		t.Fatal("expected empty MapKeyed result to be nil")
	}
	if FlatMap([]int{}, func(value int) []ui.Node { return []ui.Node{Textf("%d", value)} }) != nil {
		t.Fatal("expected empty FlatMap result to be nil")
	}
	if FilterMap([]int{}, func(value int) (ui.Node, bool) { return Textf("%d", value), true }) != nil {
		t.Fatal("expected empty FilterMap result to be nil")
	}
	if Join(Text("|")) != nil {
		t.Fatal("expected Join with no nodes to return nil")
	}
	if Join(Text("|"), nil, nil) != nil {
		t.Fatal("expected Join with only nil nodes to return nil")
	}
	if WithKey(nil, "missing") != nil {
		t.Fatal("expected WithKey(nil, ...) to return nil")
	}

	keyed := MapKeyed([]string{"alpha", "beta"}, func(value string) interface{} { return "k:" + value }, func(value string) ui.Node {
		return Li(Props{}, Text(value))
	})
	if len(keyed) != 2 || keyed[0].Props["key"] != "k:alpha" || keyed[1].Props["key"] != "k:beta" {
		t.Fatalf("expected keyed nodes, got %#v", keyed)
	}

	flatMapped := FlatMap([]int{1, 2, 3}, func(value int) []ui.Node {
		if value%2 == 0 {
			return []ui.Node{Textf("even-%d", value), Text("!")}
		}
		return nil
	})
	if len(flatMapped) != 2 || flatMapped[0].TextContent != "even-2" || flatMapped[1].TextContent != "!" {
		t.Fatalf("expected flat-mapped nodes, got %#v", flatMapped)
	}

	filtered := FilterMap([]int{1, 2, 3, 4}, func(value int) (ui.Node, bool) {
		if value%2 == 0 {
			return Textf("%d", value), true
		}
		return nil, false
	})
	if len(filtered) != 2 || filtered[0].TextContent != "2" || filtered[1].TextContent != "4" {
		t.Fatalf("expected filtered nodes, got %#v", filtered)
	}

	joined := Join(Text("|"), Text("a"), nil, Text("b"), Text("c"))
	markup, err := ui.RenderToString(Div(Props{}, joined...))
	if err != nil {
		t.Fatalf("expected joined render, got %v", err)
	}
	if markup != "<div>a|b|c</div>" {
		t.Fatalf("expected joined markup, got %q", markup)
	}
}

func TestSecondPassOptionalAndSwitchHelpers(t *testing.T) {
	value := "ready"
	if Maybe(nil, func(v string) ui.Node { return Text(v) }) != nil {
		t.Fatal("expected Maybe(nil, ...) to return nil")
	}
	maybe := Maybe(&value, func(v string) ui.Node { return Text(strings.ToUpper(v)) })
	if maybe == nil || maybe.TextContent != "READY" {
		t.Fatalf("expected Maybe to render value, got %#v", maybe)
	}

	if got := OrElse(nil, "fallback"); got != "fallback" {
		t.Fatalf("expected OrElse fallback, got %q", got)
	}
	if got := OrElse(&value, "fallback"); got != "ready" {
		t.Fatalf("expected OrElse value, got %q", got)
	}

	alt := "alt"
	if got := Coalesce(nil, &alt, &value); got == nil || *got != "alt" {
		t.Fatalf("expected first non-nil pointer, got %#v", got)
	}
	if got := Coalesce[string](nil, nil); got != nil {
		t.Fatalf("expected nil coalesce result, got %#v", got)
	}

	switched := Switch("warn",
		Case("ok", Text("ok")),
		Case("warn", Text("warn")),
		Default(Text("fallback")),
	)
	if switched == nil || switched.TextContent != "warn" {
		t.Fatalf("expected matching switch branch, got %#v", switched)
	}

	fallback := Switch("missing", Case("ok", Text("ok")), Default(Text("fallback")))
	if fallback == nil || fallback.TextContent != "fallback" {
		t.Fatalf("expected default switch branch, got %#v", fallback)
	}
	if Switch("missing", Case("ok", Text("ok"))) != nil {
		t.Fatal("expected switch without default to return nil")
	}
}

func TestPropsOfAndOptionHelpers(t *testing.T) {
	var skipped PropOption
	props := PropsOf(
		skipped,
		Class("alpha"),
		Class(ClassNames("alpha", "beta")),
		ID("demo"),
		For("demo-input"),
		Name("save-button"),
		Title("hello"),
		Type("button"),
		Value("save"),
		Placeholder("unused"),
		Href("/settings"),
		Src("/hero.png"),
		Role("button"),
		Rows(4),
		TabIndex(3),
		DisabledIf(true),
		ReadOnlyIf(false),
		SelectedIf(true),
		Required(),
		Style(map[string]string{"color": "red"}),
		Style(map[string]string{"display": "flex"}),
		Data("mode", "demo"),
		Dataset(map[string]string{"state": "ready"}),
		Aria("label", "Demo"),
		AriaSet(map[string]string{"describedby": "copy"}),
		Attr("tabIndex", 9),
		Attrs(map[string]interface{}{"data-extra": "yes"}),
	)

	elem := Button(props, Text("Save"))
	if elem.Props["id"] != "demo" {
		t.Fatalf("expected id prop, got %#v", elem.Props["id"])
	}
	if elem.Props["htmlFor"] != "demo-input" {
		t.Fatalf("expected htmlFor prop, got %#v", elem.Props["htmlFor"])
	}
	if elem.Props["name"] != "save-button" {
		t.Fatalf("expected name prop, got %#v", elem.Props["name"])
	}
	if elem.Props["class"] != "alpha beta" {
		t.Fatalf("expected class prop, got %#v", elem.Props["class"])
	}
	if elem.Props["role"] != "button" {
		t.Fatalf("expected role prop, got %#v", elem.Props["role"])
	}
	if elem.Props["rows"] != 4 {
		t.Fatalf("expected rows prop, got %#v", elem.Props["rows"])
	}
	if elem.Props["tabIndex"] != 9 {
		t.Fatalf("expected raw tabIndex override to win, got %#v", elem.Props["tabIndex"])
	}
	if elem.Props["disabled"] != true {
		t.Fatalf("expected disabled prop, got %#v", elem.Props["disabled"])
	}
	if _, ok := elem.Props["readOnly"]; ok {
		t.Fatalf("expected readonly false to be omitted, got %#v", elem.Props["readOnly"])
	}
	if elem.Props["selected"] != true {
		t.Fatalf("expected selected prop, got %#v", elem.Props["selected"])
	}
	if elem.Props["required"] != true {
		t.Fatalf("expected required prop, got %#v", elem.Props["required"])
	}
	if elem.Props["data-mode"] != "demo" || elem.Props["data-state"] != "ready" {
		t.Fatalf("expected merged data props, got %#v", elem.Props)
	}
	if elem.Props["aria-label"] != "Demo" || elem.Props["aria-describedby"] != "copy" {
		t.Fatalf("expected merged aria props, got %#v", elem.Props)
	}
	if elem.Props["data-extra"] != "yes" {
		t.Fatalf("expected raw attrs to merge, got %#v", elem.Props["data-extra"])
	}
	if style, ok := elem.Props["style"].(map[string]string); !ok || style["color"] != "red" || style["display"] != "flex" {
		t.Fatalf("expected merged style map, got %#v", elem.Props["style"])
	}
	if elem.Props["href"] != "/settings" || elem.Props["src"] != "/hero.png" {
		t.Fatalf("expected string option props, got %#v", elem.Props)
	}
	if elem.Props["value"] != "save" {
		t.Fatalf("expected value prop, got %#v", elem.Props["value"])
	}

	base := Props{
		Class: "base",
		Style: map[string]string{"display": "grid"},
		Data:  map[string]string{"base": "yes"},
		Aria:  map[string]string{"live": "polite"},
		Raw:   map[string]interface{}{"data-base": "ok"},
	}
	merged := WithProps(base, Checked(false), AutoFocus(false), Class("override"), Attr("data-extra", "value"))
	if merged.Class != "override" || merged.Checked || merged.AutoFocus {
		t.Fatalf("expected WithProps overrides, got %#v", merged)
	}
	merged.Style["display"] = "flex"
	merged.Data["base"] = "changed"
	merged.Aria["live"] = "assertive"
	merged.Raw["data-base"] = "changed"
	if base.Style["display"] != "grid" || base.Data["base"] != "yes" || base.Aria["live"] != "polite" || base.Raw["data-base"] != "ok" {
		t.Fatalf("expected WithProps to clone nested maps, base=%#v merged=%#v", base, merged)
	}
}

func TestEventOptionHelpersWrapHandlers(t *testing.T) {
	if goRuntime.GOOS == "js" && goRuntime.GOARCH == "wasm" {
		t.Skip("event helpers depend on hook context on js/wasm")
	}

	clicks := 0
	inputs := 0
	changes := 0
	submits := 0
	keydowns := 0
	keyups := 0
	mouseups := 0
	focuses := 0
	blurs := 0
	props := PropsOf(
		OnClick(func() { clicks++ }),
		OnInput(func(event ui.InputEvent) { inputs += len(event.GetValue()) }),
		OnChange(func(event ui.ChangeEvent) { changes += len(event.GetValue()) }),
		OnSubmit(func(event ui.FormEvent) { submits += len(event.GetValue()) }),
		OnKeyDown(func(event ui.KeyboardEvent) { keydowns += len(event.GetKey()) }),
		OnKeyUp(func(event ui.KeyboardEvent) { keyups += len(event.GetKey()) }),
		OnMouseUp(func(event ui.MouseEvent) { mouseups += event.GetKeyCode() + 1 }),
		OnFocus(func(event ui.Event) { focuses += len(event.GetValue()) }),
		OnBlur(func(event ui.Event) { blurs += len(event.GetValue()) }),
	)

	if props.OnClick.Value() == nil {
		t.Fatal("expected OnClick helper to produce a handler")
	}
	if props.OnInput.Value() == nil || props.OnChange.Value() == nil || props.OnSubmit.Value() == nil || props.OnKeyDown.Value() == nil || props.OnFocus.Value() == nil || props.OnBlur.Value() == nil {
		t.Fatal("expected all event helpers to produce handlers")
	}
	if props.OnKeyUp.Value() == nil {
		t.Fatal("expected OnKeyUp helper to produce a handler")
	}
	if props.OnMouseUp.Value() == nil {
		t.Fatal("expected OnMouseUp helper to produce a handler")
	}

	node := Button(props, Text("Press"))
	if node.Props["onclick"] == nil {
		t.Fatal("expected onclick prop to be emitted")
	}
	if node.Props["oninput"] == nil || node.Props["onchange"] == nil || node.Props["onsubmit"] == nil || node.Props["onkeydown"] == nil || node.Props["onfocus"] == nil || node.Props["onblur"] == nil || node.Props["onkeyup"] == nil || node.Props["onmouseup"] == nil {
		t.Fatal("expected all event props to be emitted")
	}

	if clicks != 0 || inputs != 0 || changes != 0 || submits != 0 || keydowns != 0 || keyups != 0 || mouseups != 0 || focuses != 0 || blurs != 0 {
		t.Fatalf("expected handlers not to execute during props assembly, got clicks=%d inputs=%d changes=%d submits=%d keydowns=%d keyups=%d mouseups=%d focuses=%d blurs=%d", clicks, inputs, changes, submits, keydowns, keyups, mouseups, focuses, blurs)
	}
}

func TestPreventAndStopWrapCallbacks(t *testing.T) {
	prevented := 0
	stopped := 0
	preventWrapped, ok := Prevent(func() { prevented++ }).(func(ui.Event))
	if !ok {
		t.Fatal("expected Prevent to return a ui.Event wrapper")
	}
	stopWrapped, ok := Stop(func(event ui.KeyboardEvent) { stopped += len(event.GetKey()) + 1 }).(func(ui.Event))
	if !ok {
		t.Fatal("expected Stop to return a ui.Event wrapper")
	}

	preventWrapped(ui.Event{})
	stopWrapped(ui.Event{})

	if prevented != 1 {
		t.Fatalf("expected Prevent wrapper to invoke callback once, got %d", prevented)
	}
	if stopped != 1 {
		t.Fatalf("expected Stop wrapper to invoke typed callback once, got %d", stopped)
	}

	nested, ok := Prevent(Stop(func() { prevented++ })).(func(ui.Event))
	if !ok {
		t.Fatal("expected nested wrappers to return a ui.Event wrapper")
	}
	nested(ui.Event{})
	if prevented != 2 {
		t.Fatalf("expected nested wrapper to preserve callback execution, got %d", prevented)
	}
}

func TestTemporalWrappers(t *testing.T) {
	passthrough := func() {}
	if reflect.ValueOf(Debounce(0, passthrough)).Pointer() != reflect.ValueOf(passthrough).Pointer() {
		t.Fatal("expected Debounce(<=0, ...) to return original callback")
	}
	if reflect.ValueOf(Throttle(0, passthrough)).Pointer() != reflect.ValueOf(passthrough).Pointer() {
		t.Fatal("expected Throttle(<=0, ...) to return original callback")
	}

	debouncedCount := 0
	debounced, ok := Debounce(20*time.Millisecond, func() { debouncedCount++ }).(func(ui.Event))
	if !ok {
		t.Fatal("expected Debounce to return a ui.Event wrapper")
	}
	debounced(ui.Event{})
	debounced(ui.Event{})
	debounced(ui.Event{})
	time.Sleep(50 * time.Millisecond)
	if debouncedCount != 1 {
		t.Fatalf("expected debounced callback once, got %d", debouncedCount)
	}

	throttledCount := 0
	throttled, ok := Throttle(25*time.Millisecond, func(event ui.KeyboardEvent) { throttledCount += len(event.GetKey()) + 1 }).(func(ui.Event))
	if !ok {
		t.Fatal("expected Throttle to return a ui.Event wrapper")
	}
	throttled(ui.Event{})
	throttled(ui.Event{})
	time.Sleep(10 * time.Millisecond)
	if throttledCount != 1 {
		t.Fatalf("expected immediate throttled callback, got %d", throttledCount)
	}
	throttled(ui.Event{})
	time.Sleep(40 * time.Millisecond)
	if throttledCount != 2 {
		t.Fatalf("expected trailing throttled callback, got %d", throttledCount)
	}
}

func TestInternalHandlerAndMapHelpersEdgeCases(t *testing.T) {
	if handler := toHandler(nil); handler.Value() != nil {
		t.Fatalf("expected nil callback to produce empty handler, got %#v", handler)
	}
	raw := ui.WrapHandler("raw")
	if handler := toHandler(raw); handler.Value() != raw.Value() {
		t.Fatalf("expected ui.Handler passthrough, got %#v", handler)
	}

	invoked := 0
	invokeEventCallback(nil, ui.Event{})
	invokeEventCallback("not-a-func", ui.Event{})
	invokeEventCallback(func(a, b int) { invoked++ }, ui.Event{})
	invokeEventCallback(func(v int) { invoked += v }, ui.Event{})
	invokeEventCallback(func() { invoked++ }, ui.Event{})
	if invoked != 1 {
		t.Fatalf("expected only zero-arg callback to run, got %d", invoked)
	}

	if cloneStringMap(nil) != nil {
		t.Fatal("expected nil string map clone for nil input")
	}
	if mergeStringMap(map[string]string{"a": "1"}, nil)["a"] != "1" {
		t.Fatal("expected mergeStringMap with nil values to preserve destination")
	}
	if cloneAnyMap(nil) != nil {
		t.Fatal("expected nil any map clone for nil input")
	}
	if mergeAnyMap(map[string]interface{}{"a": 1}, nil)["a"] != 1 {
		t.Fatal("expected mergeAnyMap with nil values to preserve destination")
	}

	original := Props{
		Style: map[string]string{"display": "grid"},
		Data:  map[string]string{"mode": "demo"},
		Aria:  map[string]string{"label": "demo"},
		Raw:   map[string]interface{}{"data-extra": "ok"},
	}
	cloned := cloneProps(original)
	cloned.Style["display"] = "flex"
	cloned.Data["mode"] = "changed"
	cloned.Aria["label"] = "changed"
	cloned.Raw["data-extra"] = "changed"
	if original.Style["display"] != "grid" || original.Data["mode"] != "demo" || original.Aria["label"] != "demo" || original.Raw["data-extra"] != "ok" {
		t.Fatalf("expected cloneProps to preserve the original maps, original=%#v clone=%#v", original, cloned)
	}
	if base := cloneProps(Props{}); base.Style != nil || base.Data != nil || base.Aria != nil || base.Raw != nil {
		t.Fatalf("expected zero clone props to keep nil maps, got %#v", base)
	}
}

func TestVoidTagsRemainSimpleWithPropsOf(t *testing.T) {
	brMarkup, err := ui.RenderToString(Br(PropsOf(Class("gap"))))
	if err != nil {
		t.Fatalf("expected br render, got %v", err)
	}
	if brMarkup != `<br class="gap">` {
		t.Fatalf("expected br markup, got %q", brMarkup)
	}

	hrMarkup, err := ui.RenderToString(Hr(PropsOf(ID("rule"))))
	if err != nil {
		t.Fatalf("expected hr render, got %v", err)
	}
	if hrMarkup != `<hr id="rule">` {
		t.Fatalf("expected hr markup, got %q", hrMarkup)
	}

	imgMarkup, err := ui.RenderToString(Img(PropsOf(Src("/logo.png"), Attr("loading", "lazy"))))
	if err != nil {
		t.Fatalf("expected img render, got %v", err)
	}
	if imgMarkup != `<img loading="lazy" src="/logo.png">` {
		t.Fatalf("expected img markup, got %q", imgMarkup)
	}

	inputMarkup, err := ui.RenderToString(Input(PropsOf(Type("text"), Value("hello"))))
	if err != nil {
		t.Fatalf("expected input render, got %v", err)
	}
	if inputMarkup != `<input type="text" value="hello">` {
		t.Fatalf("expected input markup, got %q", inputMarkup)
	}
}

func TestShorthandParityMatchesExplicitBuilders(t *testing.T) {
	explicit := Div(Props{Class: "panel"}, Text("hello"), Span(Props{}, Text("world")))
	shorthand := Div(PropsOf(Class("panel")), Children("hello", Span(Props{}, Text("world")))...)

	explicitMarkup, err := ui.RenderToString(explicit)
	if err != nil {
		t.Fatalf("expected explicit markup render, got %v", err)
	}
	shorthandMarkup, err := ui.RenderToString(shorthand)
	if err != nil {
		t.Fatalf("expected shorthand markup render, got %v", err)
	}
	if explicitMarkup != shorthandMarkup {
		t.Fatalf("expected shorthand output parity, explicit=%q shorthand=%q", explicitMarkup, shorthandMarkup)
	}
}

func TestSugarHelpersRenderExactHTMLString(t *testing.T) {
	node := Div(
		PropsOf(Class("panel"), Attr("data-mode", "demo")),
		Children(
			"hello",
			If(true, Span(PropsOf(Class("accent")), Text("world"))),
			Unless(true, Text("hidden")),
			TextIf(true, "!"),
		)...,
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("expected sugar markup render, got %v", err)
	}

	const want = `<div class="panel" data-mode="demo">hello<span class="accent">world</span>!</div>`
	if markup != want {
		t.Fatalf("unexpected sugar markup\nwant: %s\n got: %s", want, markup)
	}
}
