package shorthand

import (
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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

func TestTagAndSplitArgsSupportPropsSlicesAndChildren(parseT *testing.T) {
	parseNode := Tag("section",
		FromProps(Props{Class: "base", Data: map[string]string{"kind": "panel"}}),
		[]PropOption{ClassStr("override"), Attr("data-extra", "yes")},
		"alpha",
		Span(ClassStr("value"), "beta"),
		[]any{Text("gamma")},
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
		{name: "Aside", node: Aside(), tag: "aside"},
		{name: "Blockquote", node: Blockquote(), tag: "blockquote"},
		{name: "Body", node: Body(), tag: "body"},
		{name: "Button", node: Button(), tag: "button"},
		{name: "Br", node: Br(), tag: "br"},
		{name: "Code", node: Code(), tag: "code"},
		{name: "Details", node: Details(), tag: "details"},
		{name: "Dialog", node: Dialog(), tag: "dialog"},
		{name: "Div", node: Div(), tag: "div"},
		{name: "Em", node: Em(), tag: "em"},
		{name: "Fieldset", node: Fieldset(), tag: "fieldset"},
		{name: "Footer", node: Footer(), tag: "footer"},
		{name: "Form", node: Form(), tag: "form"},
		{name: "H1", node: H1(), tag: "h1"},
		{name: "H2", node: H2(), tag: "h2"},
		{name: "H3", node: H3(), tag: "h3"},
		{name: "H4", node: H4(), tag: "h4"},
		{name: "H5", node: H5(), tag: "h5"},
		{name: "H6", node: H6(), tag: "h6"},
		{name: "Head", node: Head(), tag: "head"},
		{name: "Header", node: Header(), tag: "header"},
		{name: "Hr", node: Hr(), tag: "hr"},
		{name: "Html", node: Html(), tag: "html"},
		{name: "Iframe", node: Iframe(), tag: "iframe"},
		{name: "Img", node: Img(), tag: "img"},
		{name: "Input", node: Input(), tag: "input"},
		{name: "Label", node: Label(), tag: "label"},
		{name: "Legend", node: Legend(), tag: "legend"},
		{name: "Li", node: Li(), tag: "li"},
		{name: "Main", node: Main(), tag: "main"},
		{name: "Mark", node: Mark(), tag: "mark"},
		{name: "Meta", node: Meta(), tag: "meta"},
		{name: "Nav", node: Nav(), tag: "nav"},
		{name: "NoScript", node: NoScript(), tag: "noscript"},
		{name: "Option", node: Option(), tag: "option"},
		{name: "P", node: P(), tag: "p"},
		{name: "Pre", node: Pre(), tag: "pre"},
		{name: "Script", node: Script(), tag: "script"},
		{name: "Section", node: Section(), tag: "section"},
		{name: "Select", node: Select(), tag: "select"},
		{name: "Small", node: Small(), tag: "small"},
		{name: "Span", node: Span(), tag: "span"},
		{name: "Strong", node: Strong(), tag: "strong"},
		{name: "Summary", node: Summary(), tag: "summary"},
		{name: "Table", node: Table(), tag: "table"},
		{name: "Tbody", node: Tbody(), tag: "tbody"},
		{name: "Td", node: Td(), tag: "td"},
		{name: "Textarea", node: Textarea(), tag: "textarea"},
		{name: "Th", node: Th(), tag: "th"},
		{name: "Thead", node: Thead(), tag: "thead"},
		{name: "Time", node: Time(), tag: "time"},
		{name: "Tr", node: Tr(), tag: "tr"},
		{name: "Ul", node: Ul(), tag: "ul"},
		{name: "Ol", node: Ol(), tag: "ol"},
		{name: "Tfoot", node: Tfoot(), tag: "tfoot"},
		{name: "Caption", node: Caption(), tag: "caption"},
		{name: "Colgroup", node: Colgroup(), tag: "colgroup"},
		{name: "Col", node: Col(), tag: "col"},
		{name: "Video", node: Video(), tag: "video"},
		{name: "Audio", node: Audio(), tag: "audio"},
		{name: "Source", node: Source(), tag: "source"},
		{name: "Track", node: Track(), tag: "track"},
		{name: "Canvas", node: Canvas(), tag: "canvas"},
		{name: "Optgroup", node: Optgroup(), tag: "optgroup"},
		{name: "Datalist", node: Datalist(), tag: "datalist"},
		{name: "Output", node: Output(), tag: "output"},
		{name: "Progress", node: Progress(), tag: "progress"},
		{name: "Meter", node: Meter(), tag: "meter"},
		{name: "Figure", node: Figure(), tag: "figure"},
		{name: "Figcaption", node: Figcaption(), tag: "figcaption"},
		{name: "Picture", node: Picture(), tag: "picture"},
		{name: "Abbr", node: Abbr(), tag: "abbr"},
		{name: "Kbd", node: Kbd(), tag: "kbd"},
		{name: "Sub", node: Sub(), tag: "sub"},
		{name: "Sup", node: Sup(), tag: "sup"},
		{name: "Del", node: Del(), tag: "del"},
		{name: "Ins", node: Ins(), tag: "ins"},
		{name: "B", node: B(), tag: "b"},
		{name: "I", node: I(), tag: "i"},
		{name: "U", node: U(), tag: "u"},
		{name: "Svg", node: Svg(), tag: "svg"},
		{name: "Path", node: Path(), tag: "path"},
		{name: "Circle", node: Circle(), tag: "circle"},
		{name: "Rect", node: Rect(), tag: "rect"},
		{name: "G", node: G(), tag: "g"},
		{name: "Line", node: Line(), tag: "line"},
		{name: "Polyline", node: Polyline(), tag: "polyline"},
		{name: "Polygon", node: Polygon(), tag: "polygon"},
		{name: "Defs", node: Defs(), tag: "defs"},
		{name: "Use", node: Use(), tag: "use"},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			if parseTt.node == nil || parseTt.node.Type != parseTt.tag {
				parseT2.Fatalf("expected %s tag, got %#v", parseTt.tag, parseTt.node)
			}
		})
	}

	// HiddenInput is a non-varargs convenience builder.
	parseHidden := HiddenInput("csrf", "tok")
	if parseHidden == nil || parseHidden.Type != "input" ||
		parseHidden.Props["type"] != "hidden" ||
		parseHidden.Props["name"] != "csrf" ||
		parseHidden.Props["value"] != "tok" {
		parseT.Fatalf("expected HiddenInput props, got %#v", parseHidden)
	}
}

func TestExpandedElementBuildersRenderExactHTML(parseT *testing.T) {
	parseTests := []struct {
		name string
		node ui.Node
		want string
	}{
		{name: "Ol", node: Ol(ClassStr("x"), Li("item")), want: `<ol class="x"><li>item</li></ol>`},
		{name: "Tfoot", node: Tfoot(ClassStr("x"), Tr(Td("total"))), want: `<tfoot class="x"><tr><td>total</td></tr></tfoot>`},
		{name: "Caption", node: Caption(ClassStr("x"), "caption"), want: `<caption class="x">caption</caption>`},
		{name: "Colgroup", node: Colgroup(ClassStr("x"), Col(Attr("span", "2"))), want: `<colgroup class="x"><col span="2"></colgroup>`},
		{name: "Col", node: Col(ClassStr("x")), want: `<col class="x">`},
		{name: "Video", node: Video(ClassStr("x"), Source(Src("/movie.webm"), Type("video/webm"))), want: `<video class="x"><source src="/movie.webm" type="video/webm"></video>`},
		{name: "Audio", node: Audio(ClassStr("x"), Source(Src("/audio.ogg"))), want: `<audio class="x"><source src="/audio.ogg"></audio>`},
		{name: "Source", node: Source(ClassStr("x"), Src("/clip.mp4")), want: `<source class="x" src="/clip.mp4">`},
		{name: "Track", node: Track(ClassStr("x"), Attr("kind", "captions"), Src("/captions.vtt")), want: `<track class="x" kind="captions" src="/captions.vtt">`},
		{name: "Canvas", node: Canvas(ClassStr("x"), Width("300"), Height("150"), "fallback"), want: `<canvas class="x" height="150" width="300">fallback</canvas>`},
		{name: "Optgroup", node: Optgroup(ClassStr("x"), Attr("label", "Group"), Option(Value("one"), "One")), want: `<optgroup class="x" label="Group"><option value="one">One</option></optgroup>`},
		{name: "Datalist", node: Datalist(ClassStr("x"), Option(Value("one"), "One")), want: `<datalist class="x"><option value="one">One</option></datalist>`},
		{name: "Output", node: Output(ClassStr("x"), For("field"), "10"), want: `<output class="x" for="field">10</output>`},
		{name: "Progress", node: Progress(ClassStr("x"), Max("10"), Attr("value", 3), "3"), want: `<progress class="x" max="10" value="3">3</progress>`},
		{name: "Meter", node: Meter(ClassStr("x"), Min("0"), Max("10"), Attr("value", 5), "5"), want: `<meter class="x" max="10" min="0" value="5">5</meter>`},
		{name: "Figure", node: Figure(ClassStr("x"), Figcaption("Caption")), want: `<figure class="x"><figcaption>Caption</figcaption></figure>`},
		{name: "Picture", node: Picture(ClassStr("x"), Source(Src("/hero.webp")), Img(Src("/hero.png"), Alt("Hero"))), want: `<picture class="x"><source src="/hero.webp"><img alt="Hero" src="/hero.png"></picture>`},
		{name: "Abbr", node: Abbr(ClassStr("x"), "abbr"), want: `<abbr class="x">abbr</abbr>`},
		{name: "Kbd", node: Kbd(ClassStr("x"), "ctrl"), want: `<kbd class="x">ctrl</kbd>`},
		{name: "Sub", node: Sub(ClassStr("x"), "2"), want: `<sub class="x">2</sub>`},
		{name: "Sup", node: Sup(ClassStr("x"), "2"), want: `<sup class="x">2</sup>`},
		{name: "Del", node: Del(ClassStr("x"), "old"), want: `<del class="x">old</del>`},
		{name: "Ins", node: Ins(ClassStr("x"), "new"), want: `<ins class="x">new</ins>`},
		{name: "B", node: B(ClassStr("x"), "bold"), want: `<b class="x">bold</b>`},
		{name: "I", node: I(ClassStr("x"), "italic"), want: `<i class="x">italic</i>`},
		{name: "U", node: U(ClassStr("x"), "under"), want: `<u class="x">under</u>`},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			parseMarkup, parseErr := ui.RenderToString(parseTt.node)
			if parseErr != nil {
				parseT2.Fatalf("RenderToString returned error: %v", parseErr)
			}
			if parseMarkup != parseTt.want {
				parseT2.Fatalf("unexpected markup\nwant: %s\n got: %s", parseTt.want, parseMarkup)
			}
		})
	}
}

func TestExpandedAttributeReexportsRenderExactHTML(parseT *testing.T) {
	parseNode := Div(
		A(Href("/docs"), Target("_blank"), Rel("noreferrer"), "docs"),
		Img(Src("/hero.png"), Alt("Hero"), Width("320"), Height("180"), Loading("lazy")),
		Input(
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
		),
		Table(Tbody(Tr(Td(ColSpan(2), RowSpan(3), "cell")))),
		Details(Open(), Lang("ar"), Dir("rtl"), Summary("summary")),
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<div><a href="/docs" rel="noreferrer" target="_blank">docs</a><img alt="Hero" height="180" loading="lazy" src="/hero.png" width="320"><input accept="image/*" autocomplete="off" hidden max="10" maxLength="6" min="1" minLength="2" multiple pattern="[0-9]+" required step="2" type="text"><table><tbody><tr><td colSpan="2" rowSpan="3">cell</td></tr></tbody></table><details dir="rtl" lang="ar" open><summary>summary</summary></details></div>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected expanded attribute markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}

func TestClassMapReexportIsSortedAndComposes(parseT *testing.T) {
	parseClasses := ClassMap(map[string]bool{
		"zeta":    true,
		" alpha ": true,
		"hidden":  false,
		"beta":    true,
	})
	if parseClasses != "alpha beta zeta" {
		parseT.Fatalf("expected sorted true-valued classes, got %q", parseClasses)
	}
	if parseEmpty := ClassMap(map[string]bool{"hidden": false}); parseEmpty != "" {
		parseT.Fatalf("expected empty ClassMap, got %q", parseEmpty)
	}

	parseMarkup, parseErr := ui.RenderToString(Div(ClassStr(ClassNames("base", parseClasses)), "x"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}
	if parseMarkup != `<div class="base alpha beta zeta">x</div>` {
		parseT.Fatalf("unexpected ClassMap markup: %s", parseMarkup)
	}
}

func TestEventHandlerReexportsEmitNativePropsAndPassive(parseT *testing.T) {
	parsePassive, parseOk := Passive(func() {}).(runtime.PassiveEventHandler)
	if !parseOk {
		parseT.Fatalf("expected Passive to return runtime passive handler, got %#v", Passive(func() {}))
	}
	if parsePassive.Handler == nil {
		parseT.Fatal("expected Passive to preserve callback payload")
	}

	// On* options route through ui.UseEvent, which is a hook: on the wasm build
	// calling them outside a mounted component panics by contract, and this test
	// environment has no DOM to mount into. The emission logic under test is
	// platform-independent, so the native build carries the coverage.
	if goruntime.GOOS == "js" {
		parseT.Skip("On* options are hooks on the wasm build; covered natively")
	}
	parseNode := Div(
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
	)

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

func TestSVGHelperReexportsRenderExactHTML(parseT *testing.T) {
	parseNode := Svg(
		Attr("viewBox", "0 0 16 16"),
		G(Attr("fill", "none"), Path(Attr("d", "M0 0h16v16H0z"))),
		Rect(Width("16"), Height("16")),
		Circle(Attr("cx", 8), Attr("cy", 8), Attr("r", 4)),
		Line(Attr("x1", 0), Attr("y1", 0), Attr("x2", 16), Attr("y2", 16)),
		Polyline(Attr("points", "0,0 8,8 16,0")),
		Polygon(Attr("points", "0,16 8,0 16,16")),
		Defs(Use(Attr("href", "#shape"))),
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg"><g fill="none"><path d="M0 0h16v16H0z"></path></g><rect height="16" width="16"></rect><circle cx="8" cy="8" r="4"></circle><line x1="0" x2="16" y1="0" y2="16"></line><polyline points="0,0 8,8 16,0"></polyline><polygon points="0,16 8,0 16,16"></polygon><defs><use href="#shape"></use></defs></svg>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected SVG markup\nwant: %s\n got: %s", want, parseMarkup)
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
	if parseGot := Children("a", []any{"b", Text("c")}); len(parseGot) != 3 {
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
	// String keys ride the typed fast-lane Key field.
	if parseNode := WithKey(Text("x"), "k1"); parseNode == nil || parseNode.Key != "k1" {
		parseT.Fatalf("expected key application, got %#v", parseNode)
	}
	if Switch("warn", Case("ok", Text("ok")), Case("warn", Text("warn")), Default(Text("fallback"))).TextContent != "warn" {
		parseT.Fatal("expected switch helper to match value")
	}

	parseProps := PropsOf(
		ID("demo"),
		ClassStr("panel"),
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
		Attrs(map[string]any{"data-raw": "ok"}),
	)
	parseProps = WithProps(parseProps, ClassStr("override"))
	parseElem := Button(FromProps(parseProps), "save")
	if parseElem.Props["class"] != "override" || parseElem.Props["id"] != "demo" || parseElem.Props["htmlFor"] != "field" || parseElem.Props["rows"] != 4 {
		parseT.Fatalf("expected prop options to delegate, got %#v", parseElem.Props)
	}
	if parseElem.Props["data-mode"] != "demo" || parseElem.Props["data-kind"] != "field" || parseElem.Props["aria-label"] != "Field" || parseElem.Props["aria-describedby"] != "copy" {
		parseT.Fatalf("expected data and aria props, got %#v", parseElem.Props)
	}
	// Event-handler PropOption coverage lives in the native-only
	// TestHelperReexportsEventHandlerOptions: on the WASM build handler
	// options register hooks and require component context.

	parseItems := []int{1, 2, 3}
	if parseMapped := Map(parseItems, func(parseV int) ui.Node { return Textf("%d", parseV) }); len(parseMapped) != 3 {
		parseT.Fatalf("expected Map delegation, got %#v", parseMapped)
	}
	if parseKeyed := MapKeyed(parseItems, func(parseV2 int) any { return parseV2 }, func(parseV3 int) ui.Node { return Textf("%d", parseV3) }); len(parseKeyed) != 3 || parseKeyed[0].Props["key"] != 1 {
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
	if Maybe(nil, func(parseV6 string) ui.Node { return Text(parseV6) }) != nil {
		parseT.Fatal("expected Maybe nil case")
	}
	if Maybe(&parseValue, func(parseV7 string) ui.Node { return Text(strings.ToUpper(parseV7)) }).TextContent != "READY" {
		parseT.Fatal("expected Maybe value case")
	}
	if OrElse(nil, "fallback") != "fallback" || *Coalesce(nil, &parseValue) != "ready" {
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
	parseWaitForCondition(parseT, 2*time.Second, func() bool { return parseDebouncedCount == 1 })
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
	parseWaitForCondition(parseT, 2*time.Second, func() bool { return parseThrottledCount >= 2 })
	if parseThrottledCount != 2 {
		parseT.Fatalf("expected immediate plus trailing throttled callbacks, got %d", parseThrottledCount)
	}
}

func TestShorthandHelpersRenderExactHTMLString(parseT *testing.T) {
	parseNode := Div(
		ClassStr("panel"),
		Attr("data-mode", "demo"),
		"hello",
		Span(ClassStr("accent"), "world"),
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
		return Li(ClassStr("item"), parseValue)
	})
	parseArgs := []any{ClassStr("items")}
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
