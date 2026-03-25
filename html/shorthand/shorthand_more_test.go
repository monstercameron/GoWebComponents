package shorthand

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestTagAndSplitArgsSupportPropsSlicesAndChildren(t *testing.T) {
	node := Tag("section",
		FromProps(Props{Class: "base", Data: map[string]string{"kind": "panel"}}),
		[]PropOption{Class("override"), Attr("data-extra", "yes")},
		"alpha",
		Span(Class("value"), "beta"),
		[]interface{}{Text("gamma")},
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}
	if !strings.Contains(markup, `<section class="override"`) ||
		!strings.Contains(markup, `data-kind="panel"`) ||
		!strings.Contains(markup, `data-extra="yes"`) ||
		!strings.Contains(markup, `alpha`) ||
		!strings.Contains(markup, `<span class="value">beta</span>`) ||
		!strings.Contains(markup, `gamma`) {
		t.Fatalf("unexpected markup: %s", markup)
	}
}

func TestWrapperTagFunctionsExposeExpectedTypes(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node == nil || tt.node.Type != tt.tag {
				t.Fatalf("expected %s tag, got %#v", tt.tag, tt.node)
			}
		})
	}
}

func TestHelperReexportsCoverPositiveNegativeAndEdgeCases(t *testing.T) {
	if Text("alpha").TextContent != "alpha" {
		t.Fatal("expected Text helper to delegate")
	}
	if Textf("%d", 7).TextContent != "7" {
		t.Fatal("expected Textf helper to delegate")
	}
	if TextIf(false, "hidden") != nil {
		t.Fatal("expected TextIf(false) to return nil")
	}
	if got := Children("a", []interface{}{"b", Text("c")}); len(got) != 3 {
		t.Fatalf("expected normalized children, got %#v", got)
	}
	if When(false, "hidden") != "" {
		t.Fatal("expected false When result")
	}
	if ClassNames("a", []string{"b c"}) != "a b c" {
		t.Fatal("expected ClassNames to flatten values")
	}
	if If(false, Text("x")) != nil || Unless(false, Text("x")) == nil || IfElse(true, Text("a"), Text("b")).TextContent != "a" {
		t.Fatal("expected conditional helpers to delegate")
	}
	if node := WithKey(Text("x"), "k1"); node == nil || node.Props["key"] != "k1" {
		t.Fatalf("expected key application, got %#v", node)
	}
	if Switch("warn", Case("ok", Text("ok")), Case("warn", Text("warn")), Default(Text("fallback"))).TextContent != "warn" {
		t.Fatal("expected switch helper to match value")
	}

	props := PropsOf(
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
	props = WithProps(props, Class("override"))
	elem := Button(FromProps(props), "save")
	if elem.Props["class"] != "override" || elem.Props["id"] != "demo" || elem.Props["htmlFor"] != "field" || elem.Props["rows"] != 4 {
		t.Fatalf("expected prop options to delegate, got %#v", elem.Props)
	}
	if elem.Props["data-mode"] != "demo" || elem.Props["data-kind"] != "field" || elem.Props["aria-label"] != "Field" || elem.Props["aria-describedby"] != "copy" {
		t.Fatalf("expected data and aria props, got %#v", elem.Props)
	}
	if elem.Props["onclick"] == nil || elem.Props["oninput"] == nil || elem.Props["onchange"] == nil || elem.Props["onsubmit"] == nil || elem.Props["onkeydown"] == nil || elem.Props["onkeyup"] == nil || elem.Props["onmouseup"] == nil || elem.Props["onfocus"] == nil || elem.Props["onblur"] == nil {
		t.Fatalf("expected event props, got %#v", elem.Props)
	}

	items := []int{1, 2, 3}
	if mapped := Map(items, func(v int) ui.Node { return Textf("%d", v) }); len(mapped) != 3 {
		t.Fatalf("expected Map delegation, got %#v", mapped)
	}
	if keyed := MapKeyed(items, func(v int) interface{} { return v }, func(v int) ui.Node { return Textf("%d", v) }); len(keyed) != 3 || keyed[0].Props["key"] != 1 {
		t.Fatalf("expected MapKeyed delegation, got %#v", keyed)
	}
	if flat := FlatMap(items, func(v int) []ui.Node {
		if v == 2 {
			return []ui.Node{Text("two")}
		}
		return nil
	}); len(flat) != 1 {
		t.Fatalf("expected FlatMap delegation, got %#v", flat)
	}
	if filtered := FilterMap(items, func(v int) (ui.Node, bool) { return Textf("%d", v), v%2 == 1 }); len(filtered) != 2 {
		t.Fatalf("expected FilterMap delegation, got %#v", filtered)
	}
	if joined := Join(Text("|"), Text("a"), nil, Text("b")); len(joined) != 3 {
		t.Fatalf("expected Join delegation, got %#v", joined)
	}

	value := "ready"
	if Maybe[string](nil, func(v string) ui.Node { return Text(v) }) != nil {
		t.Fatal("expected Maybe nil case")
	}
	if Maybe(&value, func(v string) ui.Node { return Text(strings.ToUpper(v)) }).TextContent != "READY" {
		t.Fatal("expected Maybe value case")
	}
	if OrElse[string](nil, "fallback") != "fallback" || *Coalesce[string](nil, &value) != "ready" {
		t.Fatal("expected OrElse and Coalesce delegation")
	}

	base := func() {}
	if Prevent(base) == nil || Stop(base) == nil {
		t.Fatal("expected event wrapper delegation")
	}
	if Debounce(0, base) == nil || Throttle(0, base) == nil {
		t.Fatal("expected non-positive timing helpers to pass through callbacks")
	}

	fragment := Fragment("a", Text("b"))
	if fragment == nil || fragment.Type != "FRAGMENT" || len(fragment.Children) != 2 {
		t.Fatalf("expected fragment helper, got %#v", fragment)
	}
}

func TestTemporalWrappersExecuteThroughDelegation(t *testing.T) {
	debouncedCount := 0
	debounced, ok := Debounce(15*time.Millisecond, func() { debouncedCount++ }).(func(ui.Event))
	if !ok {
		t.Fatal("expected debounced wrapper")
	}
	debounced(ui.Event{})
	debounced(ui.Event{})
	time.Sleep(35 * time.Millisecond)
	if debouncedCount != 1 {
		t.Fatalf("expected one debounced callback, got %d", debouncedCount)
	}

	throttledCount := 0
	throttled, ok := Throttle(20*time.Millisecond, func() { throttledCount++ }).(func(ui.Event))
	if !ok {
		t.Fatal("expected throttled wrapper")
	}
	throttled(ui.Event{})
	throttled(ui.Event{})
	time.Sleep(35 * time.Millisecond)
	if throttledCount != 2 {
		t.Fatalf("expected immediate plus trailing throttled callbacks, got %d", throttledCount)
	}
}

func TestShorthandHelpersRenderExactHTMLString(t *testing.T) {
	node := Div(
		Class("panel"),
		Attr("data-mode", "demo"),
		"hello",
		Span(Class("accent"), "world"),
		Text("!"),
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}

	const want = `<div class="panel" data-mode="demo">hello<span class="accent">world</span>!</div>`
	if markup != want {
		t.Fatalf("unexpected shorthand markup\nwant: %s\n got: %s", want, markup)
	}
}

func TestShorthandCollectionHelpersRenderExactHTMLString(t *testing.T) {
	items := Map([]string{"alpha", "beta"}, func(value string) ui.Node {
		return Li(Class("item"), value)
	})
	args := []interface{}{Class("items")}
	for _, item := range items {
		args = append(args, item)
	}
	node := Ul(args...)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}

	const want = `<ul class="items"><li class="item">alpha</li><li class="item">beta</li></ul>`
	if markup != want {
		t.Fatalf("unexpected shorthand collection markup\nwant: %s\n got: %s", want, markup)
	}
}
