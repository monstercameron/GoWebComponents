//go:build js && wasm

package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestDiv_OmitsZeroValueProps(parseT *testing.T) {
	parseElem := Div(Props{})
	if parseElem == nil {
		parseT.Fatal("expected element")
	}
	// Typed fast-lane elements carry no props map at construction time.
	if parseElem.Props != nil {
		parseT.Fatalf("expected fast-lane element without a props map, got %#v", parseElem.Props)
	}
	if len(runtime.EnsureElementProps(parseElem)) != 0 {
		parseT.Fatalf("expected empty materialized props, got %#v", parseElem.Props)
	}
}

func TestInput_KeepsMeaningfulProps(parseT *testing.T) {
	parseElem := Input(Props{
		ID:       "name",
		Value:    "alice",
		Accept:   "image/*",
		Disabled: true,
		Rows:     4,
		Class:    "field",
		Data:     map[string]string{"mode": "demo"},
		Aria:     map[string]string{"label": "Name"},
		Raw:      map[string]interface{}{"tabIndex": 2},
	})
	if parseElem == nil {
		parseT.Fatal("expected element")
	}
	if parseElem.Props["id"] != "name" {
		parseT.Fatalf("expected id prop, got %#v", parseElem.Props["id"])
	}
	if parseElem.Props["value"] != "alice" {
		parseT.Fatalf("expected value prop, got %#v", parseElem.Props["value"])
	}
	if parseElem.Props["accept"] != "image/*" {
		parseT.Fatalf("expected accept prop, got %#v", parseElem.Props["accept"])
	}
	if parseElem.Props["disabled"] != true {
		parseT.Fatalf("expected disabled prop, got %#v", parseElem.Props["disabled"])
	}
	if parseElem.Props["rows"] != 4 {
		parseT.Fatalf("expected rows prop, got %#v", parseElem.Props["rows"])
	}
	if parseElem.Props["data-mode"] != "demo" {
		parseT.Fatalf("expected data attribute, got %#v", parseElem.Props["data-mode"])
	}
	if parseElem.Props["aria-label"] != "Name" {
		parseT.Fatalf("expected aria attribute, got %#v", parseElem.Props["aria-label"])
	}
	if parseElem.Props["tabIndex"] != 2 {
		parseT.Fatalf("expected raw prop override, got %#v", parseElem.Props["tabIndex"])
	}
	if _, parseOk := parseElem.Props["required"]; parseOk {
		parseT.Fatalf("expected zero-value bool prop to be omitted, got %#v", parseElem.Props["required"])
	}
}

func TestHiddenInput_UsesHiddenTypeAndProvidedNameValue(parseT *testing.T) {
	parseElem := HiddenInput("csrf_token", "token-123")
	if parseElem == nil {
		parseT.Fatal("expected element")
	}
	if parseElem.Props["type"] != "hidden" {
		parseT.Fatalf("expected hidden type, got %#v", parseElem.Props["type"])
	}
	if parseElem.Props["name"] != "csrf_token" {
		parseT.Fatalf("expected hidden input name, got %#v", parseElem.Props["name"])
	}
	if parseElem.Props["value"] != "token-123" {
		parseT.Fatalf("expected hidden input value, got %#v", parseElem.Props["value"])
	}
}

func TestFormPropsIncludeEncType(parseT *testing.T) {
	parseElem := Form(Props{Action: "/upload", Method: "post", EncType: "multipart/form-data"})
	if parseElem == nil {
		parseT.Fatal("expected form element")
	}
	if runtime.EnsureElementProps(parseElem)["enctype"] != "multipart/form-data" {
		parseT.Fatalf("expected enctype prop, got %#v", parseElem.Props["enctype"])
	}
}

func TestTagBuildersPreservePublicProps(parseT *testing.T) {
	parseElem := Tag("input", Props{
		Key:          "k1",
		Title:        "Title",
		Type:         "email",
		Name:         "email",
		Placeholder:  "Email",
		Href:         "/home",
		Src:          "/img.png",
		Alt:          "alt",
		For:          "field",
		Role:         "button",
		Target:       "_blank",
		Rel:          "noreferrer",
		As:           "script",
		Action:       "/submit",
		Method:       "post",
		AutoComplete: "on",
		Min:          "1",
		Max:          "10",
		Step:         "2",
		Cols:         3,
		Checked:      true,
		Selected:     true,
		Required:     true,
		ReadOnly:     true,
		Hidden:       true,
		Multiple:     true,
		AutoFocus:    true,
		Style:        map[string]string{"color": "red"},
		OnClick:      ui.WrapHandler("click"),
		OnInput:      ui.WrapHandler("input"),
		OnChange:     ui.WrapHandler("change"),
		OnSubmit:     ui.WrapHandler("submit"),
		OnKeyDown:    ui.WrapHandler("keydown"),
		OnKeyUp:      ui.WrapHandler("keyup"),
		OnMouseUp:    ui.WrapHandler("mouseup"),
		OnFocus:      ui.WrapHandler("focus"),
		OnBlur:       ui.WrapHandler("blur"),
	})
	if parseElem == nil {
		parseT.Fatal("expected element")
	}
	parseProps := parseElem.Props

	parseAssertions := map[string]interface{}{
		"key":          "k1",
		"title":        "Title",
		"type":         "email",
		"name":         "email",
		"placeholder":  "Email",
		"href":         "/home",
		"src":          "/img.png",
		"alt":          "alt",
		"htmlFor":      "field",
		"role":         "button",
		"target":       "_blank",
		"rel":          "noreferrer",
		"as":           "script",
		"action":       "/submit",
		"method":       "post",
		"autocomplete": "on",
		"min":          "1",
		"max":          "10",
		"step":         "2",
		"cols":         3,
		"checked":      true,
		"selected":     true,
		"required":     true,
		"readOnly":     true,
		"hidden":       true,
		"multiple":     true,
		"autofocus":    true,
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

	for parseKey, parseExpected := range parseAssertions {
		if parseProps[parseKey] != parseExpected {
			parseT.Fatalf("expected %s to equal %#v, got %#v", parseKey, parseExpected, parseProps[parseKey])
		}
	}
	if parseStyle, parseOk := parseProps["style"].(map[string]string); !parseOk || parseStyle["color"] != "red" {
		parseT.Fatalf("expected style map to be preserved, got %#v", parseProps["style"])
	}
}

func TestResourceHintHelpersCreateLinkElements(parseT *testing.T) {
	parsePreload := Preload("/static/bin/browser-interop.wasm", "fetch")
	if parsePreload == nil {
		parseT.Fatal("expected preload element")
	}
	if parsePreload.Type != "link" {
		parseT.Fatalf("expected link tag, got %#v", parsePreload.Type)
	}
	parsePreloadProps := runtime.EnsureElementProps(parsePreload)
	if parsePreloadProps["rel"] != "preload" {
		parseT.Fatalf("expected preload rel, got %#v", parsePreloadProps["rel"])
	}
	if parsePreloadProps["href"] != "/static/bin/browser-interop.wasm" {
		parseT.Fatalf("expected preload href, got %#v", parsePreloadProps["href"])
	}
	if parsePreloadProps["as"] != "fetch" {
		parseT.Fatalf("expected preload as attribute, got %#v", parsePreloadProps["as"])
	}

	parseModulePreload := runtime.EnsureElementProps(ModulePreload("/static/modules/browser-interop-lazy-module.js"))
	if parseModulePreload["rel"] != "modulepreload" {
		parseT.Fatalf("expected modulepreload rel, got %#v", parseModulePreload["rel"])
	}
	if parseModulePreload["as"] != "script" {
		parseT.Fatalf("expected modulepreload as script, got %#v", parseModulePreload["as"])
	}

	if parsePrefetch := runtime.EnsureElementProps(Prefetch("/static/modules/next-route.js")); parsePrefetch["rel"] != "prefetch" {
		parseT.Fatalf("expected prefetch rel, got %#v", parsePrefetch["rel"])
	}

	if parsePreconnect := runtime.EnsureElementProps(Preconnect("https://cdn.example.test")); parsePreconnect["rel"] != "preconnect" {
		parseT.Fatalf("expected preconnect rel, got %#v", parsePreconnect["rel"])
	}

	if parseDns := runtime.EnsureElementProps(DNSPrefetch("https://cdn.example.test")); parseDns["rel"] != "dns-prefetch" {
		parseT.Fatalf("expected dns-prefetch rel, got %#v", parseDns["rel"])
	}
}

func TestAccessibilityPropsPreserveSemanticRelationships(parseT *testing.T) {
	parseElem := Dialog(Props{
		ID:   "settings-dialog",
		Role: "dialog",
		Aria: map[string]string{
			"labelledby":  "settings-title",
			"describedby": "settings-description",
			"modal":       "true",
		},
		Raw: map[string]interface{}{"tabIndex": -1},
	})
	if parseElem == nil {
		parseT.Fatal("expected element")
	}
	if parseElem.Type != "dialog" {
		parseT.Fatalf("expected dialog tag, got %#v", parseElem.Type)
	}
	if parseElem.Props["id"] != "settings-dialog" {
		parseT.Fatalf("expected id prop, got %#v", parseElem.Props["id"])
	}
	if parseElem.Props["role"] != "dialog" {
		parseT.Fatalf("expected role prop, got %#v", parseElem.Props["role"])
	}
	if parseElem.Props["aria-labelledby"] != "settings-title" {
		parseT.Fatalf("expected aria-labelledby prop, got %#v", parseElem.Props["aria-labelledby"])
	}
	if parseElem.Props["aria-describedby"] != "settings-description" {
		parseT.Fatalf("expected aria-describedby prop, got %#v", parseElem.Props["aria-describedby"])
	}
	if parseElem.Props["aria-modal"] != "true" {
		parseT.Fatalf("expected aria-modal prop, got %#v", parseElem.Props["aria-modal"])
	}
	if parseElem.Props["tabIndex"] != -1 {
		parseT.Fatalf("expected tabIndex prop, got %#v", parseElem.Props["tabIndex"])
	}

	parseLabel := Label(Props{For: "field-id"}, Text("Field"))
	if parseLabel == nil {
		parseT.Fatal("expected label element")
	}
	if runtime.EnsureElementProps(parseLabel)["htmlFor"] != "field-id" {
		parseT.Fatalf("expected htmlFor prop, got %#v", parseLabel.Props["htmlFor"])
	}
}

func TestTagWrappersExposeExpectedElementTypes(parseT *testing.T) {
	parseTests := []struct {
		name string
		node ui.Node
		tag  string
	}{
		{name: "A", node: A(Props{}), tag: "a"},
		{name: "Article", node: Article(Props{}), tag: "article"},
		{name: "Aside", node: Aside(Props{}), tag: "aside"},
		{name: "Blockquote", node: Blockquote(Props{}), tag: "blockquote"},
		{name: "Br", node: Br(Props{}), tag: "br"},
		{name: "Button", node: Button(Props{}), tag: "button"},
		{name: "Code", node: Code(Props{}), tag: "code"},
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
		{name: "Header", node: Header(Props{}), tag: "header"},
		{name: "Hr", node: Hr(Props{}), tag: "hr"},
		{name: "Img", node: Img(Props{}), tag: "img"},
		{name: "Input", node: Input(Props{}), tag: "input"},
		{name: "Label", node: Label(Props{}), tag: "label"},
		{name: "Legend", node: Legend(Props{}), tag: "legend"},
		{name: "Li", node: Li(Props{}), tag: "li"},
		{name: "Main", node: Main(Props{}), tag: "main"},
		{name: "Nav", node: Nav(Props{}), tag: "nav"},
		{name: "Option", node: Option(Props{}), tag: "option"},
		{name: "P", node: P(Props{}), tag: "p"},
		{name: "Pre", node: Pre(Props{}), tag: "pre"},
		{name: "Section", node: Section(Props{}), tag: "section"},
		{name: "Select", node: Select(Props{}), tag: "select"},
		{name: "Small", node: Small(Props{}), tag: "small"},
		{name: "Span", node: Span(Props{}), tag: "span"},
		{name: "Strong", node: Strong(Props{}), tag: "strong"},
		{name: "Textarea", node: Textarea(Props{}), tag: "textarea"},
		{name: "Time", node: Time(Props{}), tag: "time"},
		{name: "Ul", node: Ul(Props{}), tag: "ul"},
	}

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			if parseTt.node == nil {
				parseT2.Fatal("expected node")
			}
			if parseTt.node.Type != parseTt.tag {
				parseT2.Fatalf("expected tag %q, got %#v", parseTt.tag, parseTt.node.Type)
			}
		})
	}
}

func TestTextAndFragmentHelpers(parseT *testing.T) {
	parseText := Text("hello")
	if parseText == nil || parseText.TextContent != "hello" {
		parseT.Fatalf("expected text helper to preserve content, got %#v", parseText)
	}

	parseFragment := Fragment(parseText)
	if parseFragment == nil || parseFragment.Type != "FRAGMENT" {
		parseT.Fatalf("expected fragment helper to create fragment node, got %#v", parseFragment)
	}
}
