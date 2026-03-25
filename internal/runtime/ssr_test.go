package runtime

import "testing"

func TestRenderToStringHostTree(parseT *testing.T) {
	parseElement := CreateElement("div", map[string]interface{}{
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

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseWant := `<div class="panel" id="root" style="background:black;color:white"><label for="email">Email</label><input disabled id="email"></div>`
	if parseHtml != parseWant {
		parseT.Fatalf("unexpected html\nwant: %s\ngot:  %s", parseWant, parseHtml)
	}
}

func TestRenderToStringEscapesTextAndAttributes(parseT *testing.T) {
	parseElement := CreateElement("div", map[string]interface{}{
		"title": `<unsafe "quote">`,
	}, `hello <world> & friends`)

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseWant := `<div title="&lt;unsafe &#34;quote&#34;&gt;">hello &lt;world&gt; &amp; friends</div>`
	if parseHtml != parseWant {
		parseT.Fatalf("unexpected escaped html\nwant: %s\ngot:  %s", parseWant, parseHtml)
	}
}

func TestRenderToStringFragment(parseT *testing.T) {
	parseElement := CreateElement("FRAGMENT", nil,
		CreateElement("span", nil, "one"),
		CreateElement("span", nil, "two"),
	)

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<span>one</span><span>two</span>` {
		parseT.Fatalf("unexpected fragment html: %s", parseHtml)
	}
}

func TestRenderToStringSkipsChildrenKeyAndHandlers(parseT *testing.T) {
	parseElement := &Element{
		Type: "button",
		Props: map[string]interface{}{
			"id":       "save",
			"key":      "button-1",
			"children": []interface{}{"bad"},
			"onclick":  func() {},
		},
		Children: []interface{}{"Save"},
	}

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<button id="save">Save</button>` {
		parseT.Fatalf("unexpected button html: %s", parseHtml)
	}
}

func TestRenderToStringFunctionComponent(parseT *testing.T) {
	parseComponent := func(parseProps Attrs) *Element {
		return CreateElement("section", map[string]interface{}{"id": parseProps["id"]}, parseProps["label"])
	}
	parseElement := CreateElement(parseComponent, map[string]interface{}{"id": "hero", "label": "Welcome"})

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<section id="hero">Welcome</section>` {
		parseT.Fatalf("unexpected component html: %s", parseHtml)
	}
}
