package runtime

import "testing"

func TestHTMLWrappers_ExhaustiveCoverage(t *testing.T) {
	props := map[string]interface{}{"id": "node"}
	withChildren := []struct {
		name string
		tag  string
		fn   func(map[string]interface{}, ...interface{}) *Element
	}{
		{"Html", "html", Html},
		{"Head", "head", Head},
		{"Body", "body", Body},
		{"Title", "title", Title},
		{"Style", "style", Style},
		{"Script", "script", Script},
		{"Header", "header", Header},
		{"Nav", "nav", Nav},
		{"Main", "main", Main},
		{"Footer", "footer", Footer},
		{"Section", "section", Section},
		{"Article", "article", Article},
		{"Aside", "aside", Aside},
		{"Div", "div", Div},
		{"P", "p", P},
		{"Span", "span", Span},
		{"Pre", "pre", Pre},
		{"Blockquote", "blockquote", Blockquote},
		{"H1", "h1", H1},
		{"H2", "h2", H2},
		{"H3", "h3", H3},
		{"H4", "h4", H4},
		{"H5", "h5", H5},
		{"H6", "h6", H6},
		{"Ul", "ul", Ul},
		{"Ol", "ol", Ol},
		{"Li", "li", Li},
		{"Dl", "dl", Dl},
		{"Dt", "dt", Dt},
		{"Dd", "dd", Dd},
		{"A", "a", A},
		{"Strong", "strong", Strong},
		{"Em", "em", Em},
		{"Code", "code", Code},
		{"Small", "small", Small},
		{"Mark", "mark", Mark},
		{"Del", "del", Del},
		{"Ins", "ins", Ins},
		{"Sub", "sub", Sub},
		{"Sup", "sup", Sup},
		{"Form", "form", Form},
		{"Label", "label", Label},
		{"Button", "button", Button},
		{"Select", "select", Select},
		{"Option", "option", Option},
		{"Textarea", "textarea", Textarea},
		{"Fieldset", "fieldset", Fieldset},
		{"Legend", "legend", Legend},
		{"Table", "table", Table},
		{"Thead", "thead", Thead},
		{"Tbody", "tbody", Tbody},
		{"Tfoot", "tfoot", Tfoot},
		{"Tr", "tr", Tr},
		{"Th", "th", Th},
		{"Td", "td", Td},
		{"Caption", "caption", Caption},
		{"Colgroup", "colgroup", Colgroup},
		{"Video", "video", Video},
		{"Audio", "audio", Audio},
		{"Picture", "picture", Picture},
		{"Canvas", "canvas", Canvas},
		{"Svg", "svg", Svg},
		{"G", "g", G},
		{"Details", "details", Details},
		{"Summary", "summary", Summary},
		{"Dialog", "dialog", Dialog},
		{"Menu", "menu", Menu},
		{"Iframe", "iframe", Iframe},
		{"Object", "object", Object},
		{"Time", "time", Time},
		{"Progress", "progress", Progress},
		{"Meter", "meter", Meter},
		{"Output", "output", Output},
		{"Data", "data", Data},
		{"Abbr", "abbr", Abbr},
		{"Address", "address", Address},
		{"Cite", "cite", Cite},
		{"Kbd", "kbd", Kbd},
		{"Samp", "samp", Samp},
		{"Var", "var", Var},
		{"Q", "q", Q},
		{"Dfn", "dfn", Dfn},
		{"B", "b", B},
		{"I", "i", I},
		{"U", "u", U},
		{"S", "s", S},
		{"Bdi", "bdi", Bdi},
		{"Bdo", "bdo", Bdo},
		{"Ruby", "ruby", Ruby},
		{"Rt", "rt", Rt},
		{"Rp", "rp", Rp},
		{"Figure", "figure", Figure},
		{"Figcaption", "figcaption", Figcaption},
		{"Optgroup", "optgroup", Optgroup},
		{"Datalist", "datalist", Datalist},
		{"Hgroup", "hgroup", Hgroup},
		{"Portal", "portal", Portal},
		{"Template", "template", Template},
		{"Slot", "slot", Slot},
	}

	for _, tc := range withChildren {
		elem := tc.fn(props, "child")
		if elem.Type != tc.tag {
			t.Fatalf("%s: expected tag %q, got %#v", tc.name, tc.tag, elem.Type)
		}
		if elem.Props["id"] != "node" {
			t.Fatalf("%s: expected props to be copied", tc.name)
		}
		children, ok := elem.Props["children"].([]interface{})
		if !ok || len(children) != 1 {
			t.Fatalf("%s: expected one child in props, got %#v", tc.name, elem.Props["children"])
		}
	}

	withoutChildren := []struct {
		name string
		tag  string
		fn   func(map[string]interface{}) *Element
	}{
		{"Meta", "meta", Meta},
		{"Link", "link", Link},
		{"Hr", "hr", Hr},
		{"Br", "br", Br},
		{"Input", "input", Input},
		{"Col", "col", Col},
		{"Img", "img", Img},
		{"Source", "source", Source},
		{"Path", "path", Path},
		{"Circle", "circle", Circle},
		{"Rect", "rect", Rect},
		{"Line", "line", Line},
		{"Polygon", "polygon", Polygon},
		{"Embed", "embed", Embed},
		{"Param", "param", Param},
		{"Wbr", "wbr", Wbr},
		{"Track", "track", Track},
	}

	for _, tc := range withoutChildren {
		elem := tc.fn(props)
		if elem.Type != tc.tag {
			t.Fatalf("%s: expected tag %q, got %#v", tc.name, tc.tag, elem.Type)
		}
		children, ok := elem.Props["children"].([]interface{})
		if !ok || len(children) != 0 {
			t.Fatalf("%s: expected empty children, got %#v", tc.name, elem.Props["children"])
		}
	}
}

func TestHTMLHelpers_ExhaustiveCoverage(t *testing.T) {
	if got := ClassProps("a")["class"]; got != "a" {
		t.Fatalf("ClassProps: got %#v", got)
	}
	if got := IdProps("b")["id"]; got != "b" {
		t.Fatalf("IdProps: got %#v", got)
	}
	if got := HrefProps("/x")["href"]; got != "/x" {
		t.Fatalf("HrefProps: got %#v", got)
	}
	if got := SrcProps("/y")["src"]; got != "/y" {
		t.Fatalf("SrcProps: got %#v", got)
	}
	if got := StyleProps("display:block")["style"]; got != "display:block" {
		t.Fatalf("StyleProps: got %#v", got)
	}
	if got := TypeProps("button")["type"]; got != "button" {
		t.Fatalf("TypeProps: got %#v", got)
	}
	if got := ValueProps(42)["value"]; got != 42 {
		t.Fatalf("ValueProps: got %#v", got)
	}
	if got := PlaceholderProps("name")["placeholder"]; got != "name" {
		t.Fatalf("PlaceholderProps: got %#v", got)
	}
	if got := InputTypeProps("email")["type"]; got != "email" {
		t.Fatalf("InputTypeProps: got %#v", got)
	}
	classID := ClassIdProps("c", "d")
	if classID["class"] != "c" || classID["id"] != "d" {
		t.Fatalf("ClassIdProps: got %#v", classID)
	}
	if len(EmptyProps()) != 0 {
		t.Fatalf("EmptyProps: expected empty map")
	}
}

func TestHTMLComponentHelpers_ExhaustiveCoverage(t *testing.T) {
	componentA := func(props map[string]interface{}) *Element {
		return Div(props, "A")
	}
	componentB := func(props map[string]interface{}) *Element {
		return Span(props, "B")
	}

	with := WithComponents("article", map[string]interface{}{"class": "x"}, componentA, componentB)
	if with.Type != "article" || len(with.Children) != 2 {
		t.Fatalf("WithComponents: got %#v", with)
	}

	div := DivWithComponents(nil, componentA, componentB)
	if div.Type != "div" || len(div.Children) != 2 {
		t.Fatalf("DivWithComponents: got %#v", div)
	}

	section := SectionWithComponents(nil, componentA, componentB)
	if section.Type != "section" || len(section.Children) != 2 {
		t.Fatalf("SectionWithComponents: got %#v", section)
	}

	main := MainWithComponents(nil, componentA, componentB)
	if main.Type != "main" || len(main.Children) != 2 {
		t.Fatalf("MainWithComponents: got %#v", main)
	}
}
