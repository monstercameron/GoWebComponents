//go:build js && wasm
// +build js,wasm

package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestDiv_OmitsZeroValueProps(t *testing.T) {
	elem := Div(Props{})
	if elem == nil {
		t.Fatal("expected element")
	}
	if len(elem.Props) != 1 {
		t.Fatalf("expected only runtime children prop, got %#v", elem.Props)
	}
	if _, ok := elem.Props["children"]; !ok {
		t.Fatalf("expected runtime children prop, got %#v", elem.Props)
	}
}

func TestInput_KeepsMeaningfulProps(t *testing.T) {
	elem := Input(Props{
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
	if elem == nil {
		t.Fatal("expected element")
	}
	if elem.Props["id"] != "name" {
		t.Fatalf("expected id prop, got %#v", elem.Props["id"])
	}
	if elem.Props["value"] != "alice" {
		t.Fatalf("expected value prop, got %#v", elem.Props["value"])
	}
	if elem.Props["accept"] != "image/*" {
		t.Fatalf("expected accept prop, got %#v", elem.Props["accept"])
	}
	if elem.Props["disabled"] != true {
		t.Fatalf("expected disabled prop, got %#v", elem.Props["disabled"])
	}
	if elem.Props["rows"] != 4 {
		t.Fatalf("expected rows prop, got %#v", elem.Props["rows"])
	}
	if elem.Props["data-mode"] != "demo" {
		t.Fatalf("expected data attribute, got %#v", elem.Props["data-mode"])
	}
	if elem.Props["aria-label"] != "Name" {
		t.Fatalf("expected aria attribute, got %#v", elem.Props["aria-label"])
	}
	if elem.Props["tabIndex"] != 2 {
		t.Fatalf("expected raw prop override, got %#v", elem.Props["tabIndex"])
	}
	if _, ok := elem.Props["required"]; ok {
		t.Fatalf("expected zero-value bool prop to be omitted, got %#v", elem.Props["required"])
	}
}

func TestHiddenInput_UsesHiddenTypeAndProvidedNameValue(t *testing.T) {
	elem := HiddenInput("csrf_token", "token-123")
	if elem == nil {
		t.Fatal("expected element")
	}
	if elem.Props["type"] != "hidden" {
		t.Fatalf("expected hidden type, got %#v", elem.Props["type"])
	}
	if elem.Props["name"] != "csrf_token" {
		t.Fatalf("expected hidden input name, got %#v", elem.Props["name"])
	}
	if elem.Props["value"] != "token-123" {
		t.Fatalf("expected hidden input value, got %#v", elem.Props["value"])
	}
}

func TestFormPropsIncludeEncType(t *testing.T) {
	elem := Form(Props{Action: "/upload", Method: "post", EncType: "multipart/form-data"})
	if elem == nil {
		t.Fatal("expected form element")
	}
	if elem.Props["enctype"] != "multipart/form-data" {
		t.Fatalf("expected enctype prop, got %#v", elem.Props["enctype"])
	}
}

func TestTagBuildersPreservePublicProps(t *testing.T) {
	elem := Tag("input", Props{
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
		OnClick:      ui.RawHandler("click"),
		OnInput:      ui.RawHandler("input"),
		OnChange:     ui.RawHandler("change"),
		OnSubmit:     ui.RawHandler("submit"),
		OnKeyDown:    ui.RawHandler("keydown"),
		OnKeyUp:      ui.RawHandler("keyup"),
		OnFocus:      ui.RawHandler("focus"),
		OnBlur:       ui.RawHandler("blur"),
	})
	if elem == nil {
		t.Fatal("expected element")
	}
	props := elem.Props

	assertions := map[string]interface{}{
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
		"onfocus":      "focus",
		"onblur":       "blur",
	}

	for key, expected := range assertions {
		if props[key] != expected {
			t.Fatalf("expected %s to equal %#v, got %#v", key, expected, props[key])
		}
	}
	if style, ok := props["style"].(map[string]string); !ok || style["color"] != "red" {
		t.Fatalf("expected style map to be preserved, got %#v", props["style"])
	}
}

func TestResourceHintHelpersCreateLinkElements(t *testing.T) {
	preload := Preload("/static/bin/browser-interop.wasm", "fetch")
	if preload == nil {
		t.Fatal("expected preload element")
	}
	if preload.Type != "link" {
		t.Fatalf("expected link tag, got %#v", preload.Type)
	}
	if preload.Props["rel"] != "preload" {
		t.Fatalf("expected preload rel, got %#v", preload.Props["rel"])
	}
	if preload.Props["href"] != "/static/bin/browser-interop.wasm" {
		t.Fatalf("expected preload href, got %#v", preload.Props["href"])
	}
	if preload.Props["as"] != "fetch" {
		t.Fatalf("expected preload as attribute, got %#v", preload.Props["as"])
	}

	modulePreload := ModulePreload("/static/modules/browser-interop-lazy-module.js")
	if modulePreload.Props["rel"] != "modulepreload" {
		t.Fatalf("expected modulepreload rel, got %#v", modulePreload.Props["rel"])
	}
	if modulePreload.Props["as"] != "script" {
		t.Fatalf("expected modulepreload as script, got %#v", modulePreload.Props["as"])
	}

	prefetch := Prefetch("/static/modules/next-route.js")
	if prefetch.Props["rel"] != "prefetch" {
		t.Fatalf("expected prefetch rel, got %#v", prefetch.Props["rel"])
	}

	preconnect := Preconnect("https://cdn.example.test")
	if preconnect.Props["rel"] != "preconnect" {
		t.Fatalf("expected preconnect rel, got %#v", preconnect.Props["rel"])
	}

	dns := DNSPrefetch("https://cdn.example.test")
	if dns.Props["rel"] != "dns-prefetch" {
		t.Fatalf("expected dns-prefetch rel, got %#v", dns.Props["rel"])
	}
}

func TestAccessibilityPropsPreserveSemanticRelationships(t *testing.T) {
	elem := Dialog(Props{
		ID:   "settings-dialog",
		Role: "dialog",
		Aria: map[string]string{
			"labelledby":  "settings-title",
			"describedby": "settings-description",
			"modal":       "true",
		},
		Raw: map[string]interface{}{"tabIndex": -1},
	})
	if elem == nil {
		t.Fatal("expected element")
	}
	if elem.Type != "dialog" {
		t.Fatalf("expected dialog tag, got %#v", elem.Type)
	}
	if elem.Props["id"] != "settings-dialog" {
		t.Fatalf("expected id prop, got %#v", elem.Props["id"])
	}
	if elem.Props["role"] != "dialog" {
		t.Fatalf("expected role prop, got %#v", elem.Props["role"])
	}
	if elem.Props["aria-labelledby"] != "settings-title" {
		t.Fatalf("expected aria-labelledby prop, got %#v", elem.Props["aria-labelledby"])
	}
	if elem.Props["aria-describedby"] != "settings-description" {
		t.Fatalf("expected aria-describedby prop, got %#v", elem.Props["aria-describedby"])
	}
	if elem.Props["aria-modal"] != "true" {
		t.Fatalf("expected aria-modal prop, got %#v", elem.Props["aria-modal"])
	}
	if elem.Props["tabIndex"] != -1 {
		t.Fatalf("expected tabIndex prop, got %#v", elem.Props["tabIndex"])
	}

	label := Label(Props{For: "field-id"}, Text("Field"))
	if label == nil {
		t.Fatal("expected label element")
	}
	if label.Props["htmlFor"] != "field-id" {
		t.Fatalf("expected htmlFor prop, got %#v", label.Props["htmlFor"])
	}
}

func TestTagWrappersExposeExpectedElementTypes(t *testing.T) {
	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node == nil {
				t.Fatal("expected node")
			}
			if tt.node.Type != tt.tag {
				t.Fatalf("expected tag %q, got %#v", tt.tag, tt.node.Type)
			}
		})
	}
}

func TestTextAndFragmentHelpers(t *testing.T) {
	text := Text("hello")
	if text == nil || text.TextContent != "hello" {
		t.Fatalf("expected text helper to preserve content, got %#v", text)
	}

	fragment := Fragment(text)
	if fragment == nil || fragment.Type != "FRAGMENT" {
		t.Fatalf("expected fragment helper to create fragment node, got %#v", fragment)
	}
}
