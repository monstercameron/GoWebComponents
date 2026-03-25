package shorthand

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestTagAndSplitArgsSupportPropsSlicesAndChildren(parseT *testing.T) {
	parseNode := Tag("section",
		FromProps(Props{Class: "base", Data: map[string]string{"kind": "panel"}}),
		[]PropOption{Class("override"), Attr("data-extra", "yes")},
		"alpha",
		Span(Class("value"), "beta"),
		[]interface{}{Text("gamma")},
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `<section class="override"`) ||
		!strings.Contains(parseMarkup, `data-kind="panel"`) ||
		!strings.Contains(parseMarkup, `data-extra="yes"`) ||
		!strings.Contains(parseMarkup, `alpha`) ||
		!strings.Contains(parseMarkup, `<span class="value">beta</span>`) ||
		!strings.Contains(parseMarkup, `gamma`) {
		parseT.Fatalf("unexpected markup: %s", parseMarkup)
	}
}

func TestWrapperTagFunctionsExposeExpectedTypes(parseT *testing.T) {
	parseTests := []struct {
		name string
		node ui.Node
		tag  string
	}{
		{name: "A", node: A(), tag: "a"},
		{name: "Article", node: Article(), tag: "article"},
		{name: "Body", node: Body(), tag: "body"},
		{name: "Button", node: Button(), tag: "button"},
		{name: "Br", node: Br(), tag: "br"},
		{name: "Code", node: Code(), tag: "code"},
		{name: "Details", node: Details(), tag: "details"},
		{name: "Div", node: Div(), tag: "div"},
		{name: "Form", node: Form(), tag: "form"},
		{name: "H1", node: H1(), tag: "h1"},
		{name: "H2", node: H2(), tag: "h2"},
		{name: "H3", node: H3(), tag: "h3"},
		{name: "Head", node: Head(), tag: "head"},
		{name: "Header", node: Header(), tag: "header"},
		{name: "Hr", node: Hr(), tag: "hr"},
		{name: "Html", node: Html(), tag: "html"},
		{name: "Img", node: Img(), tag: "img"},
		{name: "Input", node: Input(), tag: "input"},
		{name: "Label", node: Label(), tag: "label"},
		{name: "Li", node: Li(), tag: "li"},
		{name: "Main", node: Main(), tag: "main"},
		{name: "Mark", node: Mark(), tag: "mark"},
		{name: "Meta", node: Meta(), tag: "meta"},
		{name: "NoScript", node: NoScript(), tag: "noscript"},
		{name: "Option", node: Option(), tag: "option"},
		{name: "P", node: P(), tag: "p"},
		{name: "Pre", node: Pre(), tag: "pre"},
		{name: "Script", node: Script(), tag: "script"},
		{name: "Section", node: Section(), tag: "section"},
		{name: "Select", node: Select(), tag: "select"},
		{name: "Span", node: Span(), tag: "span"},
		{name: "Summary", node: Summary(), tag: "summary"},
		{name: "Table", node: Table(), tag: "table"},
		{name: "Tbody", node: Tbody(), tag: "tbody"},
		{name: "Td", node: Td(), tag: "td"},
		{name: "Th", node: Th(), tag: "th"},
		{name: "Thead", node: Thead(), tag: "thead"},
		{name: "Tr", node: Tr(), tag: "tr"},
		{name: "Ul", node: Ul(), tag: "ul"},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			if parseTt.node == nil || parseTt.node.Type != parseTt.tag {
				parseT2.Fatalf("expected %s tag, got %#v", parseTt.tag, parseTt.node)
			}
		})
	}
}

func TestHelperReexportsCoverPositiveNegativeAndEdgeCases(parseT *testing.T) {
	if Text("alpha").TextContent != "alpha" {
		parseT.Fatal("expected Text helper to delegate")
	}
	if Textf("%d", 7).TextContent != "7" {
		parseT.Fatal("expected Textf helper to delegate")
	}
	if TextIf(false, "hidden") != nil {
		parseT.Fatal("expected TextIf(false) to return nil")
	}
	if parseGot := Children("a", []interface{}{"b", Text("c")}); len(parseGot) != 3 {
		parseT.Fatalf("expected normalized children, got %#v", parseGot)
	}
	if When(false, "hidden") != "" {
		parseT.Fatal("expected false When result")
	}
	if ClassNames("a", []string{"b c"}) != "a b c" {
		parseT.Fatal("expected ClassNames to flatten values")
	}
	if If(false, Text("x")) != nil || Unless(false, Text("x")) == nil || IfElse(true, Text("a"), Text("b")).TextContent != "a" {
		parseT.Fatal("expected conditional helpers to delegate")
	}
	if parseNode := WithKey(Text("x"), "k1"); parseNode == nil || parseNode.Props["key"] != "k1" {
		parseT.Fatalf("expected key application, got %#v", parseNode)
	}
	if Switch("warn", Case("ok", Text("ok")), Case("warn", Text("warn")), Default(Text("fallback"))).TextContent != "warn" {
		parseT.Fatal("expected switch helper to match value")
	}

	parseProps := PropsOf(
		ID("demo"),
		Class("panel"),
		For("field"),
		Name("field"),
		Title("Title"),
		Value("value"),
		Placeholder("placeholder"),
		Type("text"),
		Href("/href"),
		Src("/src"),
		Role("region"),
		Rows(4),
		TabIndex(2),
		Disabled(),
		Checked(),
		Selected(),
		Required(),
		ReadOnly(),
		AutoFocus(),
		DisabledIf(false),
		ReadOnlyIf(false),
		SelectedIf(false),
		Style(map[string]string{"display": "block"}),
		Data("mode", "demo"),
		Dataset(map[string]string{"kind": "field"}),
		Aria("label", "Field"),
		AriaSet(map[string]string{"describedby": "copy"}),
		Attr("data-extra", "yes"),
		Attrs(map[string]interface{}{"data-raw": "ok"}),
		OnClick(func() {}),
		OnInput(func(ui.InputEvent) {}),
		OnChange(func(ui.ChangeEvent) {}),
		OnSubmit(func(ui.FormEvent) {}),
		OnKeyDown(func(ui.KeyboardEvent) {}),
		OnKeyUp(func(ui.KeyboardEvent) {}),
		OnMouseUp(func(ui.MouseEvent) {}),
		OnFocus(func(ui.FocusEvent) {}),
		OnBlur(func(ui.FocusEvent) {}),
	)
	parseProps = WithProps(parseProps, Class("override"))
	parseElem := Button(FromProps(parseProps), "save")
	if parseElem.Props["class"] != "override" || parseElem.Props["id"] != "demo" || parseElem.Props["htmlFor"] != "field" || parseElem.Props["rows"] != 4 {
		parseT.Fatalf("expected prop options to delegate, got %#v", parseElem.Props)
	}
	if parseElem.Props["data-mode"] != "demo" || parseElem.Props["data-kind"] != "field" || parseElem.Props["aria-label"] != "Field" || parseElem.Props["aria-describedby"] != "copy" {
		parseT.Fatalf("expected data and aria props, got %#v", parseElem.Props)
	}
	if parseElem.Props["onclick"] == nil || parseElem.Props["oninput"] == nil || parseElem.Props["onchange"] == nil || parseElem.Props["onsubmit"] == nil || parseElem.Props["onkeydown"] == nil || parseElem.Props["onkeyup"] == nil || parseElem.Props["onmouseup"] == nil || parseElem.Props["onfocus"] == nil || parseElem.Props["onblur"] == nil {
		parseT.Fatalf("expected event props, got %#v", parseElem.Props)
	}

	parseItems := []int{1, 2, 3}
	if parseMapped := Map(parseItems, func(parseV int) ui.Node { return Textf("%d", parseV) }); len(parseMapped) != 3 {
		parseT.Fatalf("expected Map delegation, got %#v", parseMapped)
	}
	if parseKeyed := MapKeyed(parseItems, func(parseV2 int) interface{} { return parseV2 }, func(parseV3 int) ui.Node { return Textf("%d", parseV3) }); len(parseKeyed) != 3 || parseKeyed[0].Props["key"] != 1 {
		parseT.Fatalf("expected MapKeyed delegation, got %#v", parseKeyed)
	}
	if parseFlat := FlatMap(parseItems, func(parseV4 int) []ui.Node {
		if parseV4 == 2 {
			return []ui.Node{Text("two")}
		}
		return nil
	}); len(parseFlat) != 1 {
		parseT.Fatalf("expected FlatMap delegation, got %#v", parseFlat)
	}
	if parseFiltered := FilterMap(parseItems, func(parseV5 int) (ui.Node, bool) { return Textf("%d", parseV5), parseV5%2 == 1 }); len(parseFiltered) != 2 {
		parseT.Fatalf("expected FilterMap delegation, got %#v", parseFiltered)
	}
	if parseJoined := Join(Text("|"), Text("a"), nil, Text("b")); len(parseJoined) != 3 {
		parseT.Fatalf("expected Join delegation, got %#v", parseJoined)
	}

	parseValue := "ready"
	if Maybe[string](nil, func(parseV6 string) ui.Node { return Text(parseV6) }) != nil {
		parseT.Fatal("expected Maybe nil case")
	}
	if Maybe(&parseValue, func(parseV7 string) ui.Node { return Text(strings.ToUpper(parseV7)) }).TextContent != "READY" {
		parseT.Fatal("expected Maybe value case")
	}
	if OrElse[string](nil, "fallback") != "fallback" || *Coalesce[string](nil, &parseValue) != "ready" {
		parseT.Fatal("expected OrElse and Coalesce delegation")
	}

	parseBase := func() {}
	if Prevent(parseBase) == nil || Stop(parseBase) == nil {
		parseT.Fatal("expected event wrapper delegation")
	}
	if Debounce(0, parseBase) == nil || Throttle(0, parseBase) == nil {
		parseT.Fatal("expected non-positive timing helpers to pass through callbacks")
	}

	parseFragment := Fragment("a", Text("b"))
	if parseFragment == nil || parseFragment.Type != "FRAGMENT" || len(parseFragment.Children) != 2 {
		parseT.Fatalf("expected fragment helper, got %#v", parseFragment)
	}
}

func TestTemporalWrappersExecuteThroughDelegation(parseT *testing.T) {
	parseDebouncedCount := 0
	parseDebounced, parseOk := Debounce(15*time.Millisecond, func() { parseDebouncedCount++ }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected debounced wrapper")
	}
	parseDebounced(ui.Event{})
	parseDebounced(ui.Event{})
	time.Sleep(35 * time.Millisecond)
	if parseDebouncedCount != 1 {
		parseT.Fatalf("expected one debounced callback, got %d", parseDebouncedCount)
	}

	parseThrottledCount := 0
	parseThrottled, parseOk := Throttle(20*time.Millisecond, func() { parseThrottledCount++ }).(func(ui.Event))
	if !parseOk {
		parseT.Fatal("expected throttled wrapper")
	}
	parseThrottled(ui.Event{})
	parseThrottled(ui.Event{})
	time.Sleep(35 * time.Millisecond)
	if parseThrottledCount != 2 {
		parseT.Fatalf("expected immediate plus trailing throttled callbacks, got %d", parseThrottledCount)
	}
}

func TestShorthandHelpersRenderExactHTMLString(parseT *testing.T) {
	parseNode := Div(
		Class("panel"),
		Attr("data-mode", "demo"),
		"hello",
		Span(Class("accent"), "world"),
		Text("!"),
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<div class="panel" data-mode="demo">hello<span class="accent">world</span>!</div>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected shorthand markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}

func TestShorthandCollectionHelpersRenderExactHTMLString(parseT *testing.T) {
	parseItems := Map([]string{"alpha", "beta"}, func(parseValue string) ui.Node {
		return Li(Class("item"), parseValue)
	})
	parseArgs := []interface{}{Class("items")}
	for _, parseItem := range parseItems {
		parseArgs = append(parseArgs, parseItem)
	}
	parseNode := Ul(parseArgs...)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<ul class="items"><li class="item">alpha</li><li class="item">beta</li></ul>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected shorthand collection markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}
