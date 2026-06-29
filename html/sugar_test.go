package html

import (
	"fmt"
	"reflect"
	goRuntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// parseWaitForCondition polls parseCheck every 2ms until it returns true or parseTimeout elapses.
func parseWaitForCondition(parseT *testing.T, parseTimeout time.Duration, parseCheck func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for condition")
}

type sugarStringer string

func (parseS sugarStringer) String() string {
	return string(parseS)
}

func TestChildrenNormalizesMixedInputs(parseT *testing.T) {
	parseChildren := Children(
		"alpha",
		[]string{"beta", "gamma"},
		Span(Props{}, Text("delta")),
		[]any{nil, sugarStringer("epsilon"), 42, func() string { return "zeta" }},
	)

	if len(parseChildren) != 7 {
		parseT.Fatalf("expected 7 normalized children, got %d", len(parseChildren))
	}

	if parseChildren[0].TextContent != "alpha" {
		parseT.Fatalf("expected first text child alpha, got %#v", parseChildren[0])
	}
	if parseChildren[1].TextContent != "beta" {
		parseT.Fatalf("expected second text child beta, got %#v", parseChildren[1])
	}
	if parseChildren[2].TextContent != "gamma" {
		parseT.Fatalf("expected third text child gamma, got %#v", parseChildren[2])
	}
	if parseChildren[3].Type != "span" {
		parseT.Fatalf("expected preserved span child, got %#v", parseChildren[3].Type)
	}
	if parseChildren[4].TextContent != "epsilon" {
		parseT.Fatalf("expected stringer child epsilon, got %#v", parseChildren[4])
	}
	if parseChildren[5].TextContent != "42" {
		parseT.Fatalf("expected numeric child to stringify, got %#v", parseChildren[5])
	}
	if parseChildren[6].Type != runtime.ReactiveTextNodeType {
		parseT.Fatalf("expected reactive text child, got %#v", parseChildren[6].Type)
	}
}

func TestChildrenWorksWithExistingBuilderExpansion(parseT *testing.T) {
	parseNode := Div(Props{}, Children("hello", []any{" ", Textf("%s", "world")})...)
	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("expected SSR render to succeed, got %v", parseErr)
	}
	if parseMarkup != "<div>hello world</div>" {
		parseT.Fatalf("expected normalized child expansion markup, got %q", parseMarkup)
	}
}

func TestTextSupportsFormattedAndReactiveContent(parseT *testing.T) {
	if Text(nil) != nil {
		parseT.Fatal("expected Text(nil) to return nil")
	}
	parseOriginal := Span(Props{}, Text("passthrough"))
	if Text(parseOriginal) != parseOriginal {
		parseT.Fatal("expected Text(ui.Node) to return the original node")
	}
	if Text(true).TextContent != "true" {
		parseT.Fatalf("expected bool text conversion, got %#v", Text(true))
	}
	if Text(uint16(9)).TextContent != "9" {
		parseT.Fatalf("expected numeric text conversion, got %#v", Text(uint16(9)))
	}

	parseFormatted := Textf("count:%d", 7)
	if parseFormatted == nil || parseFormatted.TextContent != "count:7" {
		parseT.Fatalf("expected formatted text node, got %#v", parseFormatted)
	}

	if TextIf(false, "hidden") != nil {
		parseT.Fatal("expected TextIf(false, ...) to return nil")
	}
	if parseTextIf := TextIf(true, "shown"); parseTextIf == nil || parseTextIf.TextContent != "shown" {
		parseT.Fatalf("expected TextIf(true, ...) to return a text node, got %#v", parseTextIf)
	}

	parseReactive := Text(func() string { return "live" })
	if parseReactive == nil || parseReactive.Type != runtime.ReactiveTextNodeType {
		parseT.Fatalf("expected reactive text node, got %#v", parseReactive)
	}

	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseReactive))
	if parseErr != nil {
		parseT.Fatalf("expected SSR reactive text render, got %v", parseErr)
	}
	if parseMarkup != "<div>live</div>" {
		parseT.Fatalf("expected reactive SSR markup, got %q", parseMarkup)
	}
}

func TestChildrenAndClassHelpersHandleEmptyAndArrayInputs(parseT *testing.T) {
	if Children() != nil {
		parseT.Fatal("expected Children() to return nil")
	}
	if Children(nil, []any{nil}) != nil {
		parseT.Fatal("expected Children with only nil values to return nil")
	}

	parseChildren := Children([2]int{5, 6})
	if len(parseChildren) != 2 || parseChildren[0].TextContent != "5" || parseChildren[1].TextContent != "6" {
		parseT.Fatalf("expected reflected array children, got %#v", parseChildren)
	}

	if ClassNames() != "" {
		parseT.Fatal("expected empty ClassNames result")
	}
	parseClasses := ClassNames([2]string{"alpha", "beta"}, 7)
	if parseClasses != "alpha beta 7" {
		parseT.Fatalf("expected reflected class fragments, got %q", parseClasses)
	}
}

func TestClassHelpersNormalizeWhitespaceAndConditionals(parseT *testing.T) {
	parseClassName := ClassNames(" alpha  beta ", When(false, "hidden"), When(true, "active"), []string{"gamma delta"}, sugarStringer("epsilon"))
	if parseClassName != "alpha beta active gamma delta epsilon" {
		parseT.Fatalf("expected normalized class string, got %q", parseClassName)
	}
	if When(false, "missing") != "" {
		parseT.Fatal("expected false When helper to return empty string")
	}
}

func TestClassMapIsDeterministicAndComposes(parseT *testing.T) {
	parseClasses := ClassMap(map[string]bool{
		"zeta":    true,
		" alpha ": true,
		"hidden":  false,
		"beta":    true,
		"":        true,
	})
	if parseClasses != "alpha beta zeta" {
		parseT.Fatalf("expected sorted true-valued classes, got %q", parseClasses)
	}
	if parseEmpty := ClassMap(map[string]bool{"hidden": false}); parseEmpty != "" {
		parseT.Fatalf("expected empty ClassMap for all-false input, got %q", parseEmpty)
	}
	if parseCombined := ClassNames("base", parseClasses, When(true, "active")); parseCombined != "base alpha beta zeta active" {
		parseT.Fatalf("expected ClassMap to compose inside ClassNames, got %q", parseCombined)
	}
}

func TestConditionalHelpersSelectExpectedNodes(parseT *testing.T) {
	parseTrueNode := Span(Props{}, Text("true"))
	parseFalseNode := Span(Props{}, Text("false"))

	if If(false, parseTrueNode) != nil {
		parseT.Fatal("expected If(false, ...) to return nil")
	}
	if If(true, parseTrueNode) != parseTrueNode {
		parseT.Fatal("expected If(true, ...) to return provided node")
	}
	if IfElse(true, parseTrueNode, parseFalseNode) != parseTrueNode {
		parseT.Fatal("expected IfElse(true, ...) to return true branch")
	}
	if IfElse(false, parseTrueNode, parseFalseNode) != parseFalseNode {
		parseT.Fatal("expected IfElse(false, ...) to return false branch")
	}
	if Unless(true, parseFalseNode) != nil {
		parseT.Fatal("expected Unless(true, ...) to return nil")
	}
	if Unless(false, parseFalseNode) != parseFalseNode {
		parseT.Fatal("expected Unless(false, ...) to return provided node")
	}
}

func TestMapPreservesOrder(parseT *testing.T) {
	parseNodes := Map([]int{3, 1, 4}, func(parseValue int) ui.Node {
		return Li(Props{}, Text(fmt.Sprintf("item-%d", parseValue)))
	})

	parseMarkup, parseErr := ui.RenderToString(Ul(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected mapped list SSR render, got %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "<li>item-3</li><li>item-1</li><li>item-4</li>") {
		parseT.Fatalf("expected mapped order to be preserved, got %q", parseMarkup)
	}
}

func TestSecondPassCollectionHelpers(parseT *testing.T) {
	if Map([]int{}, func(parseValue int) ui.Node { return Textf("%d", parseValue) }) != nil {
		parseT.Fatal("expected empty Map result to be nil")
	}
	if MapKeyed([]int{}, func(parseValue2 int) any { return parseValue2 }, func(parseValue3 int) ui.Node { return Textf("%d", parseValue3) }) != nil {
		parseT.Fatal("expected empty MapKeyed result to be nil")
	}
	if FlatMap([]int{}, func(parseValue4 int) []ui.Node { return []ui.Node{Textf("%d", parseValue4)} }) != nil {
		parseT.Fatal("expected empty FlatMap result to be nil")
	}
	if FilterMap([]int{}, func(parseValue5 int) (ui.Node, bool) { return Textf("%d", parseValue5), true }) != nil {
		parseT.Fatal("expected empty FilterMap result to be nil")
	}
	if Join(Text("|")) != nil {
		parseT.Fatal("expected Join with no nodes to return nil")
	}
	if Join(Text("|"), nil, nil) != nil {
		parseT.Fatal("expected Join with only nil nodes to return nil")
	}
	if WithKey(nil, "missing") != nil {
		parseT.Fatal("expected WithKey(nil, ...) to return nil")
	}

	parseKeyed := MapKeyed([]string{"alpha", "beta"}, func(parseValue6 string) any { return "k:" + parseValue6 }, func(parseValue7 string) ui.Node {
		return Li(Props{}, Text(parseValue7))
	})
	if len(parseKeyed) != 2 || parseKeyed[0].Props["key"] != "k:alpha" || parseKeyed[1].Props["key"] != "k:beta" {
		parseT.Fatalf("expected keyed nodes, got %#v", parseKeyed)
	}

	parseFlatMapped := FlatMap([]int{1, 2, 3}, func(parseValue8 int) []ui.Node {
		if parseValue8%2 == 0 {
			return []ui.Node{Textf("even-%d", parseValue8), Text("!")}
		}
		return nil
	})
	if len(parseFlatMapped) != 2 || parseFlatMapped[0].TextContent != "even-2" || parseFlatMapped[1].TextContent != "!" {
		parseT.Fatalf("expected flat-mapped nodes, got %#v", parseFlatMapped)
	}

	parseFiltered := FilterMap([]int{1, 2, 3, 4}, func(parseValue9 int) (ui.Node, bool) {
		if parseValue9%2 == 0 {
			return Textf("%d", parseValue9), true
		}
		return nil, false
	})
	if len(parseFiltered) != 2 || parseFiltered[0].TextContent != "2" || parseFiltered[1].TextContent != "4" {
		parseT.Fatalf("expected filtered nodes, got %#v", parseFiltered)
	}

	parseJoined := Join(Text("|"), Text("a"), nil, Text("b"), Text("c"))
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseJoined...))
	if parseErr != nil {
		parseT.Fatalf("expected joined render, got %v", parseErr)
	}
	if parseMarkup != "<div>a|b|c</div>" {
		parseT.Fatalf("expected joined markup, got %q", parseMarkup)
	}
}

func TestSecondPassOptionalAndSwitchHelpers(parseT *testing.T) {
	parseValue := "ready"
	if Maybe(nil, func(parseV string) ui.Node { return Text(parseV) }) != nil {
		parseT.Fatal("expected Maybe(nil, ...) to return nil")
	}
	parseMaybe := Maybe(&parseValue, func(parseV2 string) ui.Node { return Text(strings.ToUpper(parseV2)) })
	if parseMaybe == nil || parseMaybe.TextContent != "READY" {
		parseT.Fatalf("expected Maybe to render value, got %#v", parseMaybe)
	}

	if parseGot := OrElse(nil, "fallback"); parseGot != "fallback" {
		parseT.Fatalf("expected OrElse fallback, got %q", parseGot)
	}
	if parseGot2 := OrElse(&parseValue, "fallback"); parseGot2 != "ready" {
		parseT.Fatalf("expected OrElse value, got %q", parseGot2)
	}

	parseAlt := "alt"
	if parseGot3 := Coalesce(nil, &parseAlt, &parseValue); parseGot3 == nil || *parseGot3 != "alt" {
		parseT.Fatalf("expected first non-nil pointer, got %#v", parseGot3)
	}
	if parseGot4 := Coalesce[string](nil, nil); parseGot4 != nil {
		parseT.Fatalf("expected nil coalesce result, got %#v", parseGot4)
	}

	parseSwitched := Switch("warn",
		Case("ok", Text("ok")),
		Case("warn", Text("warn")),
		Default(Text("fallback")),
	)
	if parseSwitched == nil || parseSwitched.TextContent != "warn" {
		parseT.Fatalf("expected matching switch branch, got %#v", parseSwitched)
	}

	parseFallback := Switch("missing", Case("ok", Text("ok")), Default(Text("fallback")))
	if parseFallback == nil || parseFallback.TextContent != "fallback" {
		parseT.Fatalf("expected default switch branch, got %#v", parseFallback)
	}
	if Switch("missing", Case("ok", Text("ok"))) != nil {
		parseT.Fatal("expected switch without default to return nil")
	}
}

func TestPropsOfAndOptionHelpers(parseT *testing.T) {
	var parseSkipped PropOption
	parseProps := PropsOf(
		parseSkipped,
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
		Attrs(map[string]any{"data-extra": "yes"}),
	)

	parseElem := Button(parseProps, Text("Save"))
	if parseElem.Props["id"] != "demo" {
		parseT.Fatalf("expected id prop, got %#v", parseElem.Props["id"])
	}
	if parseElem.Props["htmlFor"] != "demo-input" {
		parseT.Fatalf("expected htmlFor prop, got %#v", parseElem.Props["htmlFor"])
	}
	if parseElem.Props["name"] != "save-button" {
		parseT.Fatalf("expected name prop, got %#v", parseElem.Props["name"])
	}
	if parseElem.Props["class"] != "alpha beta" {
		parseT.Fatalf("expected class prop, got %#v", parseElem.Props["class"])
	}
	if parseElem.Props["role"] != "button" {
		parseT.Fatalf("expected role prop, got %#v", parseElem.Props["role"])
	}
	if parseElem.Props["rows"] != 4 {
		parseT.Fatalf("expected rows prop, got %#v", parseElem.Props["rows"])
	}
	if parseElem.Props["tabIndex"] != 9 {
		parseT.Fatalf("expected raw tabIndex override to win, got %#v", parseElem.Props["tabIndex"])
	}
	if parseElem.Props["disabled"] != true {
		parseT.Fatalf("expected disabled prop, got %#v", parseElem.Props["disabled"])
	}
	if _, parseOk := parseElem.Props["readOnly"]; parseOk {
		parseT.Fatalf("expected readonly false to be omitted, got %#v", parseElem.Props["readOnly"])
	}
	if parseElem.Props["selected"] != true {
		parseT.Fatalf("expected selected prop, got %#v", parseElem.Props["selected"])
	}
	if parseElem.Props["required"] != true {
		parseT.Fatalf("expected required prop, got %#v", parseElem.Props["required"])
	}
	if parseElem.Props["data-mode"] != "demo" || parseElem.Props["data-state"] != "ready" {
		parseT.Fatalf("expected merged data props, got %#v", parseElem.Props)
	}
	if parseElem.Props["aria-label"] != "Demo" || parseElem.Props["aria-describedby"] != "copy" {
		parseT.Fatalf("expected merged aria props, got %#v", parseElem.Props)
	}
	if parseElem.Props["data-extra"] != "yes" {
		parseT.Fatalf("expected raw attrs to merge, got %#v", parseElem.Props["data-extra"])
	}
	if parseStyle, parseOk2 := parseElem.Props["style"].(map[string]string); !parseOk2 || parseStyle["color"] != "red" || parseStyle["display"] != "flex" {
		parseT.Fatalf("expected merged style map, got %#v", parseElem.Props["style"])
	}
	if parseElem.Props["href"] != "/settings" || parseElem.Props["src"] != "/hero.png" {
		parseT.Fatalf("expected string option props, got %#v", parseElem.Props)
	}
	if parseElem.Props["value"] != "save" {
		parseT.Fatalf("expected value prop, got %#v", parseElem.Props["value"])
	}

	parseBase := Props{
		Class: "base",
		Style: map[string]string{"display": "grid"},
		Data:  map[string]string{"base": "yes"},
		Aria:  map[string]string{"live": "polite"},
		Raw:   map[string]any{"data-base": "ok"},
	}
	parseMerged := WithProps(parseBase, Checked(false), AutoFocus(false), Class("override"), Attr("data-extra", "value"))
	if parseMerged.Class != "override" || parseMerged.Checked || parseMerged.AutoFocus {
		parseT.Fatalf("expected WithProps overrides, got %#v", parseMerged)
	}
	parseMerged.Style["display"] = "flex"
	parseMerged.Data["base"] = "changed"
	parseMerged.Aria["live"] = "assertive"
	parseMerged.Raw["data-base"] = "changed"
	if parseBase.Style["display"] != "grid" || parseBase.Data["base"] != "yes" || parseBase.Aria["live"] != "polite" || parseBase.Raw["data-base"] != "ok" {
		parseT.Fatalf("expected WithProps to clone nested maps, base=%#v merged=%#v", parseBase, parseMerged)
	}
}

func TestExpandedAttributeHelpersRenderExactHTML(parseT *testing.T) {
	parseNode := Div(Props{},
		A(PropsOf(Href("/docs"), Target("_blank"), Rel("noreferrer")), Text("docs")),
		Img(PropsOf(Src("/hero.png"), Alt("Hero"), Width("320"), Height("180"), Loading("lazy"))),
		Input(PropsOf(
			Type("text"),
			Accept("image/*"),
			AutoComplete("off"),
			Min("1"),
			Max("10"),
			Step("2"),
			Pattern("[0-9]+"),
			MaxLength(6),
			MinLength(2),
			Multiple(),
			Required(),
			Hidden(),
		)),
		Table(Props{}, Tbody(Props{}, Tr(Props{}, Td(PropsOf(ColSpan(2), RowSpan(3)), Text("cell"))))),
		Details(PropsOf(Open(), Lang("ar"), Dir("rtl")), Summary(Props{}, Text("summary"))),
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("expected expanded attributes to render, got %v", parseErr)
	}

	const want = `<div><a href="/docs" rel="noreferrer" target="_blank">docs</a><img alt="Hero" height="180" loading="lazy" src="/hero.png" width="320"><input accept="image/*" autocomplete="off" hidden max="10" maxLength="6" min="1" minLength="2" multiple pattern="[0-9]+" required step="2" type="text"><table><tbody><tr><td colSpan="2" rowSpan="3">cell</td></tr></tbody></table><details dir="rtl" lang="ar" open><summary>summary</summary></details></div>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected expanded attribute markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}

func TestEventOptionHelpersWrapHandlers(parseT *testing.T) {
	if goRuntime.GOOS == "js" && goRuntime.GOARCH == "wasm" {
		parseT.Skip("event helpers depend on hook context on js/wasm")
	}

	parseClicks := 0
	parseInputs := 0
	parseChanges := 0
	parseSubmits := 0
	parseKeydowns := 0
	parseKeyups := 0
	parseMouseups := 0
	parseFocuses := 0
	parseBlurs := 0
	parseProps := PropsOf(
		OnClick(func() { parseClicks++ }),
		OnInput(func(parseEvent ui.InputEvent) { parseInputs += len(parseEvent.GetValue()) }),
		OnChange(func(parseEvent2 ui.ChangeEvent) { parseChanges += len(parseEvent2.GetValue()) }),
		OnSubmit(func(parseEvent3 ui.FormEvent) { parseSubmits += len(parseEvent3.GetValue()) }),
		OnKeyDown(func(parseEvent4 ui.KeyboardEvent) { parseKeydowns += len(parseEvent4.GetKey()) }),
		OnKeyUp(func(parseEvent5 ui.KeyboardEvent) { parseKeyups += len(parseEvent5.GetKey()) }),
		OnMouseUp(func(parseEvent6 ui.MouseEvent) { parseMouseups += parseEvent6.GetKeyCode() + 1 }),
		OnFocus(func(parseEvent7 ui.Event) { parseFocuses += len(parseEvent7.GetValue()) }),
		OnBlur(func(parseEvent8 ui.Event) { parseBlurs += len(parseEvent8.GetValue()) }),
	)

	if parseProps.OnClick.Value() == nil {
		parseT.Fatal("expected OnClick helper to produce a handler")
	}
	if parseProps.OnInput.Value() == nil || parseProps.OnChange.Value() == nil || parseProps.OnSubmit.Value() == nil || parseProps.OnKeyDown.Value() == nil || parseProps.OnFocus.Value() == nil || parseProps.OnBlur.Value() == nil {
		parseT.Fatal("expected all event helpers to produce handlers")
	}
	if parseProps.OnKeyUp.Value() == nil {
		parseT.Fatal("expected OnKeyUp helper to produce a handler")
	}
	if parseProps.OnMouseUp.Value() == nil {
		parseT.Fatal("expected OnMouseUp helper to produce a handler")
	}

	parseNode := Button(parseProps, Text("Press"))
	if parseNode.Props["onclick"] == nil {
		parseT.Fatal("expected onclick prop to be emitted")
	}
	if parseNode.Props["oninput"] == nil || parseNode.Props["onchange"] == nil || parseNode.Props["onsubmit"] == nil || parseNode.Props["onkeydown"] == nil || parseNode.Props["onfocus"] == nil || parseNode.Props["onblur"] == nil || parseNode.Props["onkeyup"] == nil || parseNode.Props["onmouseup"] == nil {
		parseT.Fatal("expected all event props to be emitted")
	}

	if parseClicks != 0 || parseInputs != 0 || parseChanges != 0 || parseSubmits != 0 || parseKeydowns != 0 || parseKeyups != 0 || parseMouseups != 0 || parseFocuses != 0 || parseBlurs != 0 {
		parseT.Fatalf("expected handlers not to execute during props assembly, got clicks=%d inputs=%d changes=%d submits=%d keydowns=%d keyups=%d mouseups=%d focuses=%d blurs=%d", parseClicks, parseInputs, parseChanges, parseSubmits, parseKeydowns, parseKeyups, parseMouseups, parseFocuses, parseBlurs)
	}
}

func TestExpandedEventOptionHelpersEmitNativePropsAndPassive(parseT *testing.T) {
	if goRuntime.GOOS == "js" && goRuntime.GOARCH == "wasm" {
		parseT.Skip("event helpers depend on hook context on js/wasm")
	}

	parsePassiveCallback := func() {}
	parsePassive, parseOk := Passive(parsePassiveCallback).(runtime.PassiveEventHandler)
	if !parseOk {
		parseT.Fatalf("expected Passive to return a runtime passive handler, got %#v", Passive(parsePassiveCallback))
	}
	if parsePassive.Handler == nil {
		parseT.Fatal("expected Passive to preserve callback payload")
	}

	parseNode := Div(PropsOf(
		OnPointerDown(parsePassive),
		OnPointerMove(func(ui.Event) {}),
		OnPointerUp(func(ui.Event) {}),
		OnTouchStart(func(ui.Event) {}),
		OnTouchMove(func(ui.Event) {}),
		OnTouchEnd(func(ui.Event) {}),
		OnDragStart(func(ui.Event) {}),
		OnDragOver(func(ui.Event) {}),
		OnDrop(func(ui.Event) {}),
		OnDragEnd(func(ui.Event) {}),
		OnMouseDown(func(ui.Event) {}),
		OnMouseEnter(func(ui.Event) {}),
		OnMouseLeave(func(ui.Event) {}),
		OnDoubleClick(func(ui.Event) {}),
		OnContextMenu(func(ui.Event) {}),
		OnWheel(func(ui.Event) {}),
		OnTransitionEnd(func(ui.Event) {}),
		OnAnimationEnd(func(ui.Event) {}),
		OnLoad(func(ui.Event) {}),
		OnError(func(ui.Event) {}),
	))

	for _, parseName := range []string{
		"onpointerdown",
		"onpointermove",
		"onpointerup",
		"ontouchstart",
		"ontouchmove",
		"ontouchend",
		"ondragstart",
		"ondragover",
		"ondrop",
		"ondragend",
		"onmousedown",
		"onmouseenter",
		"onmouseleave",
		"ondblclick",
		"oncontextmenu",
		"onwheel",
		"ontransitionend",
		"onanimationend",
		"onload",
		"onerror",
	} {
		if parseNode.Props[parseName] == nil {
			parseT.Fatalf("expected %s event prop to be emitted, props=%#v", parseName, parseNode.Props)
		}
	}

	if _, parsePassiveOk := parseNode.Props["onpointerdown"].(runtime.PassiveEventHandler); !parsePassiveOk {
		parseT.Fatalf("expected passive pointerdown prop, got %#v", parseNode.Props["onpointerdown"])
	}
}

func TestPreventAndStopWrapCallbacks(parseT *testing.T) {
	parsePrevented := 0
	parseStopped := 0
	parsePreventWrapped, parseOk := Prevent(func() { parsePrevented++ }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected Prevent to return a ui.Event wrapper")
	}
	parseStopWrapped, parseOk := Stop(func(parseEvent ui.KeyboardEvent) { parseStopped += len(parseEvent.GetKey()) + 1 }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected Stop to return a ui.Event wrapper")
	}

	parsePreventWrapped(ui.Event{})
	parseStopWrapped(ui.Event{})

	if parsePrevented != 1 {
		parseT.Fatalf("expected Prevent wrapper to invoke callback once, got %d", parsePrevented)
	}
	if parseStopped != 1 {
		parseT.Fatalf("expected Stop wrapper to invoke typed callback once, got %d", parseStopped)
	}

	parseNested, parseOk := Prevent(Stop(func() { parsePrevented++ })).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected nested wrappers to return a ui.Event wrapper")
	}
	parseNested(ui.Event{})
	if parsePrevented != 2 {
		parseT.Fatalf("expected nested wrapper to preserve callback execution, got %d", parsePrevented)
	}
}

func TestTemporalWrappers(parseT *testing.T) {
	parsePassthrough := func() {}
	if reflect.ValueOf(Debounce(0, parsePassthrough)).Pointer() != reflect.ValueOf(parsePassthrough).Pointer() {
		parseT.Fatal("expected Debounce(<=0, ...) to return original callback")
	}
	if reflect.ValueOf(Throttle(0, parsePassthrough)).Pointer() != reflect.ValueOf(parsePassthrough).Pointer() {
		parseT.Fatal("expected Throttle(<=0, ...) to return original callback")
	}

	parseDebouncedCount := 0
	parseDebounced, parseOk := Debounce(20*time.Millisecond, func() { parseDebouncedCount++ }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected Debounce to return a ui.Event wrapper")
	}
	parseDebounced(ui.Event{})
	parseDebounced(ui.Event{})
	parseDebounced(ui.Event{})
	parseWaitForCondition(parseT, 2*time.Second, func() bool { return parseDebouncedCount == 1 })
	if parseDebouncedCount != 1 {
		parseT.Fatalf("expected debounced callback once, got %d", parseDebouncedCount)
	}

	parseThrottledCount := 0
	parseThrottled, parseOk := Throttle(25*time.Millisecond, func(parseEvent ui.KeyboardEvent) { parseThrottledCount += len(parseEvent.GetKey()) + 1 }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected Throttle to return a ui.Event wrapper")
	}
	parseThrottled(ui.Event{})
	parseThrottled(ui.Event{})
	parseWaitForCondition(parseT, 2*time.Second, func() bool { return parseThrottledCount >= 1 })
	if parseThrottledCount != 1 {
		parseT.Fatalf("expected immediate throttled callback, got %d", parseThrottledCount)
	}
	parseThrottled(ui.Event{})
	parseWaitForCondition(parseT, 2*time.Second, func() bool { return parseThrottledCount >= 2 })
	if parseThrottledCount != 2 {
		parseT.Fatalf("expected trailing throttled callback, got %d", parseThrottledCount)
	}
}

func TestInternalHandlerAndMapHelpersEdgeCases(parseT *testing.T) {
	if parseHandler := toHandler(nil); parseHandler.Value() != nil {
		parseT.Fatalf("expected nil callback to produce empty handler, got %#v", parseHandler)
	}
	parseRaw := ui.WrapHandler("raw")
	if parseHandler2 := toHandler(parseRaw); parseHandler2.Value() != parseRaw.Value() {
		parseT.Fatalf("expected ui.Handler passthrough, got %#v", parseHandler2)
	}

	parseInvoked := 0
	invokeEventCallback(nil, ui.Event{})
	invokeEventCallback("not-a-func", ui.Event{})
	invokeEventCallback(func(parseA, parseB int) { parseInvoked++ }, ui.Event{})
	invokeEventCallback(func(parseV int) { parseInvoked += parseV }, ui.Event{})
	invokeEventCallback(func() { parseInvoked++ }, ui.Event{})
	if parseInvoked != 1 {
		parseT.Fatalf("expected only zero-arg callback to run, got %d", parseInvoked)
	}

	if cloneStringMap(nil) != nil {
		parseT.Fatal("expected nil string map clone for nil input")
	}
	if mergeStringMap(map[string]string{"a": "1"}, nil)["a"] != "1" {
		parseT.Fatal("expected mergeStringMap with nil values to preserve destination")
	}
	if cloneAnyMap(nil) != nil {
		parseT.Fatal("expected nil any map clone for nil input")
	}
	if mergeAnyMap(map[string]any{"a": 1}, nil)["a"] != 1 {
		parseT.Fatal("expected mergeAnyMap with nil values to preserve destination")
	}

	parseOriginal := Props{
		Style: map[string]string{"display": "grid"},
		Data:  map[string]string{"mode": "demo"},
		Aria:  map[string]string{"label": "demo"},
		Raw:   map[string]any{"data-extra": "ok"},
	}
	parseCloned := cloneProps(parseOriginal)
	parseCloned.Style["display"] = "flex"
	parseCloned.Data["mode"] = "changed"
	parseCloned.Aria["label"] = "changed"
	parseCloned.Raw["data-extra"] = "changed"
	if parseOriginal.Style["display"] != "grid" || parseOriginal.Data["mode"] != "demo" || parseOriginal.Aria["label"] != "demo" || parseOriginal.Raw["data-extra"] != "ok" {
		parseT.Fatalf("expected cloneProps to preserve the original maps, original=%#v clone=%#v", parseOriginal, parseCloned)
	}
	if parseBase := cloneProps(Props{}); parseBase.Style != nil || parseBase.Data != nil || parseBase.Aria != nil || parseBase.Raw != nil {
		parseT.Fatalf("expected zero clone props to keep nil maps, got %#v", parseBase)
	}
}

func TestVoidTagsRemainSimpleWithPropsOf(parseT *testing.T) {
	parseBrMarkup, parseErr := ui.RenderToString(Br(PropsOf(Class("gap"))))
	if parseErr != nil {
		parseT.Fatalf("expected br render, got %v", parseErr)
	}
	if parseBrMarkup != `<br class="gap">` {
		parseT.Fatalf("expected br markup, got %q", parseBrMarkup)
	}

	parseHrMarkup, parseErr := ui.RenderToString(Hr(PropsOf(ID("rule"))))
	if parseErr != nil {
		parseT.Fatalf("expected hr render, got %v", parseErr)
	}
	if parseHrMarkup != `<hr id="rule">` {
		parseT.Fatalf("expected hr markup, got %q", parseHrMarkup)
	}

	parseImgMarkup, parseErr := ui.RenderToString(Img(PropsOf(Src("/logo.png"), Attr("loading", "lazy"))))
	if parseErr != nil {
		parseT.Fatalf("expected img render, got %v", parseErr)
	}
	if parseImgMarkup != `<img loading="lazy" src="/logo.png">` {
		parseT.Fatalf("expected img markup, got %q", parseImgMarkup)
	}

	parseInputMarkup, parseErr := ui.RenderToString(Input(PropsOf(Type("text"), Value("hello"))))
	if parseErr != nil {
		parseT.Fatalf("expected input render, got %v", parseErr)
	}
	if parseInputMarkup != `<input type="text" value="hello">` {
		parseT.Fatalf("expected input markup, got %q", parseInputMarkup)
	}
}

func TestSVGHelperInjectsXMLNSWithoutMutatingRawMap(parseT *testing.T) {
	parseRaw := map[string]any{"viewBox": "0 0 10 10"}
	parseMarkup, parseErr := ui.RenderToString(Svg(Props{Raw: parseRaw}, Circle(Props{Raw: map[string]any{"cx": 5, "cy": 5, "r": 4}})))
	if parseErr != nil {
		parseT.Fatalf("expected svg render, got %v", parseErr)
	}
	const want = `<svg viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg"><circle cx="5" cy="5" r="4"></circle></svg>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected svg markup\nwant: %s\n got: %s", want, parseMarkup)
	}
	if _, parseMutated := parseRaw["xmlns"]; parseMutated {
		parseT.Fatalf("expected Svg helper not to mutate caller Raw map, got %#v", parseRaw)
	}

	parseCustomMarkup, parseErr := ui.RenderToString(Svg(Props{Raw: map[string]any{"xmlns": "urn:custom"}}))
	if parseErr != nil {
		parseT.Fatalf("expected custom svg render, got %v", parseErr)
	}
	if parseCustomMarkup != `<svg xmlns="urn:custom"></svg>` {
		parseT.Fatalf("expected custom xmlns to be preserved, got %q", parseCustomMarkup)
	}
}

func TestShorthandParityMatchesExplicitBuilders(parseT *testing.T) {
	parseExplicit := Div(Props{Class: "panel"}, Text("hello"), Span(Props{}, Text("world")))
	parseShorthand := Div(PropsOf(Class("panel")), Children("hello", Span(Props{}, Text("world")))...)

	parseExplicitMarkup, parseErr := ui.RenderToString(parseExplicit)
	if parseErr != nil {
		parseT.Fatalf("expected explicit markup render, got %v", parseErr)
	}
	parseShorthandMarkup, parseErr := ui.RenderToString(parseShorthand)
	if parseErr != nil {
		parseT.Fatalf("expected shorthand markup render, got %v", parseErr)
	}
	if parseExplicitMarkup != parseShorthandMarkup {
		parseT.Fatalf("expected shorthand output parity, explicit=%q shorthand=%q", parseExplicitMarkup, parseShorthandMarkup)
	}
}

func TestSugarHelpersRenderExactHTMLString(parseT *testing.T) {
	parseNode := Div(
		PropsOf(Class("panel"), Attr("data-mode", "demo")),
		Children(
			"hello",
			If(true, Span(PropsOf(Class("accent")), Text("world"))),
			Unless(true, Text("hidden")),
			TextIf(true, "!"),
		)...,
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("expected sugar markup render, got %v", parseErr)
	}

	const want = `<div class="panel" data-mode="demo">hello<span class="accent">world</span>!</div>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected sugar markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}
