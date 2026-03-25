package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestNativeToRuntimePropsOmitsZeroValues(parseT *testing.T) {
	if parseProps := toRuntimeProps(Props{}); parseProps != nil {
		parseT.Fatalf("expected zero-value props to encode as nil, got %#v", parseProps)
	}
}

func TestNativeToRuntimePropsIncludesFieldsAndRawOverrides(parseT *testing.T) {
	parseEncoded := toRuntimeProps(Props{
		ID:           "field-id",
		Class:        "field shell",
		Key:          "node-1",
		Slot:         "actions",
		Title:        "Title",
		Type:         "email",
		Name:         "email",
		Value:        "cam@example.test",
		Placeholder:  "Email",
		Accept:       "image/*",
		Href:         "/settings",
		Src:          "/logo.png",
		Alt:          "logo",
		For:          "target-id",
		Role:         "button",
		Target:       "_blank",
		Rel:          "noreferrer",
		As:           "fetch",
		Action:       "/submit",
		Method:       "post",
		EncType:      "multipart/form-data",
		AutoComplete: "on",
		Min:          "1",
		Max:          "10",
		Step:         "2",
		Rows:         3,
		Cols:         4,
		TabIndex:     7,
		Checked:      true,
		Disabled:     true,
		Selected:     true,
		Required:     true,
		ReadOnly:     true,
		Hidden:       true,
		Multiple:     true,
		AutoFocus:    true,
		Style:        map[string]string{"display": "grid"},
		Data:         map[string]string{"mode": "demo"},
		Aria:         map[string]string{"label": "Email"},
		Raw: map[string]interface{}{
			"class":    "raw-class",
			"tabIndex": 9,
			"data-raw": "yes",
		},
		OnClick:   ui.WrapHandler("click"),
		OnInput:   ui.WrapHandler("input"),
		OnChange:  ui.WrapHandler("change"),
		OnSubmit:  ui.WrapHandler("submit"),
		OnKeyDown: ui.WrapHandler("keydown"),
		OnKeyUp:   ui.WrapHandler("keyup"),
		OnMouseUp: ui.WrapHandler("mouseup"),
		OnFocus:   ui.WrapHandler("focus"),
		OnBlur:    ui.WrapHandler("blur"),
	})

	parseChecks := map[string]interface{}{
		"id":           "field-id",
		"class":        "raw-class",
		"key":          "node-1",
		"slot":         "actions",
		"title":        "Title",
		"type":         "email",
		"name":         "email",
		"value":        "cam@example.test",
		"placeholder":  "Email",
		"accept":       "image/*",
		"href":         "/settings",
		"src":          "/logo.png",
		"alt":          "logo",
		"htmlFor":      "target-id",
		"role":         "button",
		"target":       "_blank",
		"rel":          "noreferrer",
		"as":           "fetch",
		"action":       "/submit",
		"method":       "post",
		"enctype":      "multipart/form-data",
		"autocomplete": "on",
		"min":          "1",
		"max":          "10",
		"step":         "2",
		"rows":         3,
		"cols":         4,
		"tabIndex":     9,
		"checked":      true,
		"disabled":     true,
		"selected":     true,
		"required":     true,
		"readOnly":     true,
		"hidden":       true,
		"multiple":     true,
		"autofocus":    true,
		"data-mode":    "demo",
		"data-raw":     "yes",
		"aria-label":   "Email",
		"onclick":      "click",
		"oninput":      "input",
		"onchange":     "change",
		"onsubmit":     "submit",
		"onkeydown":    "keydown",
		"onkeyup":      "keyup",
		"onmouseup":    "mouseup",
		"onfocus":      "focus",
		"onblur":       "blur",
	}

	for parseKey, parseWant := range parseChecks {
		if parseGot := parseEncoded[parseKey]; parseGot != parseWant {
			parseT.Fatalf("expected %s=%#v, got %#v", parseKey, parseWant, parseGot)
		}
	}

	parseStyle, parseOk := parseEncoded["style"].(map[string]string)
	if !parseOk || parseStyle["display"] != "grid" {
		parseT.Fatalf("expected style map, got %#v", parseEncoded["style"])
	}
}

func TestNativeToInterfacesHandlesEmptyAndPreservesOrder(parseT *testing.T) {
	if parseValues := toInterfaces(nil); parseValues != nil {
		parseT.Fatalf("expected nil for empty children, got %#v", parseValues)
	}

	parseChildren := []ui.Node{Text("alpha"), Span(Props{}, Text("beta"))}
	parseValues2 := toInterfaces(parseChildren)
	if len(parseValues2) != 2 {
		parseT.Fatalf("expected two interface children, got %#v", parseValues2)
	}
	if parseNode, parseOk := parseValues2[0].(*ui.Element); !parseOk || parseNode.TextContent != "alpha" {
		parseT.Fatalf("expected first child preserved, got %#v", parseValues2[0])
	}
	if parseNode2, parseOk2 := parseValues2[1].(*ui.Element); !parseOk2 || parseNode2.Type != "span" {
		parseT.Fatalf("expected second child preserved, got %#v", parseValues2[1])
	}
}

func TestNativeTagAndLinkBuildersPreserveChildren(parseT *testing.T) {
	parseNode := Tag("section", Props{Class: "shell"}, Text("alpha"), Span(Props{}, Text("beta")))
	if parseNode == nil || parseNode.Type != "section" {
		parseT.Fatalf("expected section node, got %#v", parseNode)
	}
	if parseNode.Props["class"] != "shell" {
		parseT.Fatalf("expected class prop, got %#v", parseNode.Props)
	}
	if len(parseNode.Children) != 2 {
		parseT.Fatalf("expected two children, got %#v", parseNode.Children)
	}

	parseLink := Link(Props{Rel: "stylesheet", Href: "/app.css"})
	if parseLink == nil || parseLink.Type != "link" {
		parseT.Fatalf("expected link node, got %#v", parseLink)
	}
	if parseLink.Props["rel"] != "stylesheet" || parseLink.Props["href"] != "/app.css" {
		parseT.Fatalf("expected link props, got %#v", parseLink.Props)
	}
}

func TestNativeCustomElementWithoutExtraChannels(parseT *testing.T) {
	parseNode := CustomElement("demo-card", CustomElementProps{Props: Props{Class: "shell"}}, Text("child"))
	if parseNode == nil || parseNode.Type != "demo-card" {
		parseT.Fatalf("expected custom element, got %#v", parseNode)
	}
	if parseNode.Props["class"] != "shell" {
		parseT.Fatalf("expected class prop, got %#v", parseNode.Props)
	}
	parseChild, parseOk := parseNode.Children[0].(*ui.Element)
	if len(parseNode.Children) != 1 || !parseOk || parseChild.TextContent != "child" {
		parseT.Fatalf("expected child preservation, got %#v", parseNode.Children)
	}
}

func TestNativeTagWrappersExposeExpectedElementTypes(parseT *testing.T) {
	parseTests := []struct {
		name string
		node ui.Node
		tag  string
	}{
		{name: "A", node: A(Props{}), tag: "a"},
		{name: "Article", node: Article(Props{}), tag: "article"},
		{name: "Aside", node: Aside(Props{}), tag: "aside"},
		{name: "Blockquote", node: Blockquote(Props{}), tag: "blockquote"},
		{name: "Body", node: Body(Props{}), tag: "body"},
		{name: "Br", node: Br(Props{}), tag: "br"},
		{name: "Button", node: Button(Props{}), tag: "button"},
		{name: "Code", node: Code(Props{}), tag: "code"},
		{name: "Details", node: Details(Props{}), tag: "details"},
		{name: "Dialog", node: Dialog(Props{}), tag: "dialog"},
		{name: "Div", node: Div(Props{}), tag: "div"},
		{name: "Em", node: Em(Props{}), tag: "em"},
		{name: "Fieldset", node: Fieldset(Props{}), tag: "fieldset"},
		{name: "Footer", node: Footer(Props{}), tag: "footer"},
		{name: "Form", node: Form(Props{}), tag: "form"},
		{name: "H1", node: H1(Props{}), tag: "h1"},
		{name: "H2", node: H2(Props{}), tag: "h2"},
		{name: "H3", node: H3(Props{}), tag: "h3"},
		{name: "H4", node: H4(Props{}), tag: "h4"},
		{name: "H5", node: H5(Props{}), tag: "h5"},
		{name: "H6", node: H6(Props{}), tag: "h6"},
		{name: "Head", node: Head(Props{}), tag: "head"},
		{name: "Header", node: Header(Props{}), tag: "header"},
		{name: "Html", node: Html(Props{}), tag: "html"},
		{name: "Hr", node: Hr(Props{}), tag: "hr"},
		{name: "Img", node: Img(Props{}), tag: "img"},
		{name: "Input", node: Input(Props{}), tag: "input"},
		{name: "Label", node: Label(Props{}), tag: "label"},
		{name: "Legend", node: Legend(Props{}), tag: "legend"},
		{name: "Li", node: Li(Props{}), tag: "li"},
		{name: "Main", node: Main(Props{}), tag: "main"},
		{name: "Mark", node: Mark(Props{}), tag: "mark"},
		{name: "Meta", node: Meta(Props{}), tag: "meta"},
		{name: "Nav", node: Nav(Props{}), tag: "nav"},
		{name: "NoScript", node: NoScript(Props{}), tag: "noscript"},
		{name: "Option", node: Option(Props{}), tag: "option"},
		{name: "P", node: P(Props{}), tag: "p"},
		{name: "Pre", node: Pre(Props{}), tag: "pre"},
		{name: "Script", node: Script(Props{}), tag: "script"},
		{name: "Section", node: Section(Props{}), tag: "section"},
		{name: "Select", node: Select(Props{}), tag: "select"},
		{name: "Small", node: Small(Props{}), tag: "small"},
		{name: "Span", node: Span(Props{}), tag: "span"},
		{name: "Strong", node: Strong(Props{}), tag: "strong"},
		{name: "Summary", node: Summary(Props{}), tag: "summary"},
		{name: "Table", node: Table(Props{}), tag: "table"},
		{name: "Tbody", node: Tbody(Props{}), tag: "tbody"},
		{name: "Td", node: Td(Props{}), tag: "td"},
		{name: "Th", node: Th(Props{}), tag: "th"},
		{name: "Thead", node: Thead(Props{}), tag: "thead"},
		{name: "Textarea", node: Textarea(Props{}), tag: "textarea"},
		{name: "Time", node: Time(Props{}), tag: "time"},
		{name: "Tr", node: Tr(Props{}), tag: "tr"},
		{name: "Ul", node: Ul(Props{}), tag: "ul"},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			if parseTt.node == nil || parseTt.node.Type != parseTt.tag {
				parseT2.Fatalf("expected tag %q, got %#v", parseTt.tag, parseTt.node)
			}
		})
	}
}

func TestNativeConvenienceHelpers(parseT *testing.T) {
	parseHidden := HiddenInput("csrf", "token")
	if parseHidden.Type != "input" || parseHidden.Props["type"] != "hidden" || parseHidden.Props["name"] != "csrf" || parseHidden.Props["value"] != "token" {
		parseT.Fatalf("expected hidden input props, got %#v", parseHidden)
	}

	parseFragment := Fragment(Text("one"), Text("two"))
	if parseFragment == nil || parseFragment.Type != "FRAGMENT" || len(parseFragment.Children) != 2 {
		parseT.Fatalf("expected fragment children, got %#v", parseFragment)
	}

	parseResourceHints := []struct {
		node ui.Node
		rel  string
		as   interface{}
	}{
		{node: Preload("/bundle.wasm", "fetch"), rel: "preload", as: "fetch"},
		{node: ModulePreload("/chunk.js"), rel: "modulepreload", as: "script"},
		{node: Prefetch("/next.js"), rel: "prefetch", as: nil},
		{node: Preconnect("https://cdn.example.test"), rel: "preconnect", as: nil},
		{node: DNSPrefetch("https://cdn.example.test"), rel: "dns-prefetch", as: nil},
	}
	for _, parseTt := range parseResourceHints {
		if parseTt.node.Type != "link" || parseTt.node.Props["rel"] != parseTt.rel {
			parseT.Fatalf("expected rel %q, got %#v", parseTt.rel, parseTt.node)
		}
		if parseTt.as != nil && parseTt.node.Props["as"] != parseTt.as {
			parseT.Fatalf("expected as %v, got %#v", parseTt.as, parseTt.node.Props["as"])
		}
	}
}
