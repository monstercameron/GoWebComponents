package runtime

import "testing"

func TestRenderToStringHostTree(t *testing.T) {
	element := CreateElement("div", map[string]interface{}{
		"id":        "root",
		"className": "panel",
		"style": map[string]string{
			"background": "black",
			"color":      "white",
		},
	},
		CreateElement("label", map[string]interface{}{"htmlFor": "email"}, "Email"),
		CreateElement("input", map[string]interface{}{"id": "email", "disabled": true, "checked": false}),
	)

	html, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	want := `<div class="panel" id="root" style="background:black;color:white"><label for="email">Email</label><input disabled id="email"></div>`
	if html != want {
		t.Fatalf("unexpected html\nwant: %s\ngot:  %s", want, html)
	}
}

func TestRenderToStringEscapesTextAndAttributes(t *testing.T) {
	element := CreateElement("div", map[string]interface{}{
		"title": `<unsafe "quote">`,
	}, `hello <world> & friends`)

	html, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	want := `<div title="&lt;unsafe &#34;quote&#34;&gt;">hello &lt;world&gt; &amp; friends</div>`
	if html != want {
		t.Fatalf("unexpected escaped html\nwant: %s\ngot:  %s", want, html)
	}
}

func TestRenderToStringFragment(t *testing.T) {
	element := CreateElement("FRAGMENT", nil,
		CreateElement("span", nil, "one"),
		CreateElement("span", nil, "two"),
	)

	html, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if html != `<span>one</span><span>two</span>` {
		t.Fatalf("unexpected fragment html: %s", html)
	}
}

func TestRenderToStringSkipsChildrenKeyAndHandlers(t *testing.T) {
	element := &Element{
		Type: "button",
		Props: map[string]interface{}{
			"id":       "save",
			"key":      "button-1",
			"children": []interface{}{"bad"},
			"onclick":  func() {},
		},
		Children: []interface{}{"Save"},
	}

	html, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if html != `<button id="save">Save</button>` {
		t.Fatalf("unexpected button html: %s", html)
	}
}

func TestRenderToStringFunctionComponent(t *testing.T) {
	component := func(props Attrs) *Element {
		return CreateElement("section", map[string]interface{}{"id": props["id"]}, props["label"])
	}
	element := CreateElement(component, map[string]interface{}{"id": "hero", "label": "Welcome"})

	html, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if html != `<section id="hero">Welcome</section>` {
		t.Fatalf("unexpected component html: %s", html)
	}
}
