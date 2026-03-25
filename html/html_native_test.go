package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestNativeToRuntimePropsOmitsZeroValues(t *testing.T) {
	if props := toRuntimeProps(Props{}); props != nil {
		t.Fatalf("expected zero-value props to encode as nil, got %#v", props)
	}
}

func TestNativeToRuntimePropsIncludesFieldsAndRawOverrides(t *testing.T) {
	encoded := toRuntimeProps(Props{
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
		OnClick:   ui.RawHandler("click"),
		OnInput:   ui.RawHandler("input"),
		OnChange:  ui.RawHandler("change"),
		OnSubmit:  ui.RawHandler("submit"),
		OnKeyDown: ui.RawHandler("keydown"),
		OnKeyUp:   ui.RawHandler("keyup"),
		OnMouseUp: ui.RawHandler("mouseup"),
		OnFocus:   ui.RawHandler("focus"),
		OnBlur:    ui.RawHandler("blur"),
	})

	checks := map[string]interface{}{
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

	for key, want := range checks {
		if got := encoded[key]; got != want {
			t.Fatalf("expected %s=%#v, got %#v", key, want, got)
		}
	}

	style, ok := encoded["style"].(map[string]string)
	if !ok || style["display"] != "grid" {
		t.Fatalf("expected style map, got %#v", encoded["style"])
	}
}

func TestNativeToInterfacesHandlesEmptyAndPreservesOrder(t *testing.T) {
	if values := toInterfaces(nil); values != nil {
		t.Fatalf("expected nil for empty children, got %#v", values)
	}

	children := []ui.Node{Text("alpha"), Span(Props{}, Text("beta"))}
	values := toInterfaces(children)
	if len(values) != 2 {
		t.Fatalf("expected two interface children, got %#v", values)
	}
	if node, ok := values[0].(*ui.Element); !ok || node.TextContent != "alpha" {
		t.Fatalf("expected first child preserved, got %#v", values[0])
	}
	if node, ok := values[1].(*ui.Element); !ok || node.Type != "span" {
		t.Fatalf("expected second child preserved, got %#v", values[1])
	}
}

func TestNativeTagAndLinkBuildersPreserveChildren(t *testing.T) {
	node := Tag("section", Props{Class: "shell"}, Text("alpha"), Span(Props{}, Text("beta")))
	if node == nil || node.Type != "section" {
		t.Fatalf("expected section node, got %#v", node)
	}
	if node.Props["class"] != "shell" {
		t.Fatalf("expected class prop, got %#v", node.Props)
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected two children, got %#v", node.Children)
	}

	link := Link(Props{Rel: "stylesheet", Href: "/app.css"})
	if link == nil || link.Type != "link" {
		t.Fatalf("expected link node, got %#v", link)
	}
	if link.Props["rel"] != "stylesheet" || link.Props["href"] != "/app.css" {
		t.Fatalf("expected link props, got %#v", link.Props)
	}
}

func TestNativeCustomElementWithoutExtraChannels(t *testing.T) {
	node := CustomElement("demo-card", CustomElementProps{Props: Props{Class: "shell"}}, Text("child"))
	if node == nil || node.Type != "demo-card" {
		t.Fatalf("expected custom element, got %#v", node)
	}
	if node.Props["class"] != "shell" {
		t.Fatalf("expected class prop, got %#v", node.Props)
	}
	child, ok := node.Children[0].(*ui.Element)
	if len(node.Children) != 1 || !ok || child.TextContent != "child" {
		t.Fatalf("expected child preservation, got %#v", node.Children)
	}
}

func TestNativeTagWrappersExposeExpectedElementTypes(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node == nil || tt.node.Type != tt.tag {
				t.Fatalf("expected tag %q, got %#v", tt.tag, tt.node)
			}
		})
	}
}

func TestNativeConvenienceHelpers(t *testing.T) {
	hidden := HiddenInput("csrf", "token")
	if hidden.Type != "input" || hidden.Props["type"] != "hidden" || hidden.Props["name"] != "csrf" || hidden.Props["value"] != "token" {
		t.Fatalf("expected hidden input props, got %#v", hidden)
	}

	fragment := Fragment(Text("one"), Text("two"))
	if fragment == nil || fragment.Type != "FRAGMENT" || len(fragment.Children) != 2 {
		t.Fatalf("expected fragment children, got %#v", fragment)
	}

	resourceHints := []struct {
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
	for _, tt := range resourceHints {
		if tt.node.Type != "link" || tt.node.Props["rel"] != tt.rel {
			t.Fatalf("expected rel %q, got %#v", tt.rel, tt.node)
		}
		if tt.as != nil && tt.node.Props["as"] != tt.as {
			t.Fatalf("expected as %v, got %#v", tt.as, tt.node.Props["as"])
		}
	}
}
