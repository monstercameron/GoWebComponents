package runtime

import "testing"

func TestHTMLWrappers_ExhaustiveCoverage(parseT *testing.T) {
	parseProps := map[string]any{"id": "node"}
	parseWithChildren := []struct {
		name string
		tag  string
		fn   func(map[string]any, ...any) *Element
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

	for _, parseTc := range parseWithChildren {
		parseElem := parseTc.fn(parseProps, "child")
		if parseElem.Type != parseTc.tag {
			parseT.Fatalf("%s: expected tag %q, got %#v", parseTc.name, parseTc.tag, parseElem.Type)
		}
		if parseElem.Props["id"] != "node" {
			parseT.Fatalf("%s: expected props to be copied", parseTc.name)
		}
		parseChildren, parseOk := parseElem.Props["children"].([]any)
		if !parseOk || len(parseChildren) != 1 {
			parseT.Fatalf("%s: expected one child in props, got %#v", parseTc.name, parseElem.Props["children"])
		}
	}

	parseWithoutChildren := []struct {
		name string
		tag  string
		fn   func(map[string]any) *Element
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

	for _, parseTc2 := range parseWithoutChildren {
		parseElem2 := parseTc2.fn(parseProps)
		if parseElem2.Type != parseTc2.tag {
			parseT.Fatalf("%s: expected tag %q, got %#v", parseTc2.name, parseTc2.tag, parseElem2.Type)
		}
		parseChildren2, parseOk2 := parseElem2.Props["children"].([]any)
		if !parseOk2 || len(parseChildren2) != 0 {
			parseT.Fatalf("%s: expected empty children, got %#v", parseTc2.name, parseElem2.Props["children"])
		}
	}
}

func TestHTMLHelpers_ExhaustiveCoverage(parseT *testing.T) {
	if parseGot := ClassProps("a")["class"]; parseGot != "a" {
		parseT.Fatalf("ClassProps: got %#v", parseGot)
	}
	if parseGot2 := IdProps("b")["id"]; parseGot2 != "b" {
		parseT.Fatalf("IdProps: got %#v", parseGot2)
	}
	if parseGot3 := HrefProps("/x")["href"]; parseGot3 != "/x" {
		parseT.Fatalf("HrefProps: got %#v", parseGot3)
	}
	if parseGot4 := SrcProps("/y")["src"]; parseGot4 != "/y" {
		parseT.Fatalf("SrcProps: got %#v", parseGot4)
	}
	if parseGot5 := StyleProps("display:block")["style"]; parseGot5 != "display:block" {
		parseT.Fatalf("StyleProps: got %#v", parseGot5)
	}
	if parseGot6 := TypeProps("button")["type"]; parseGot6 != "button" {
		parseT.Fatalf("TypeProps: got %#v", parseGot6)
	}
	if parseGot7 := ValueProps(42)["value"]; parseGot7 != 42 {
		parseT.Fatalf("ValueProps: got %#v", parseGot7)
	}
	if parseGot8 := PlaceholderProps("name")["placeholder"]; parseGot8 != "name" {
		parseT.Fatalf("PlaceholderProps: got %#v", parseGot8)
	}
	if parseGot9 := InputTypeProps("email")["type"]; parseGot9 != "email" {
		parseT.Fatalf("InputTypeProps: got %#v", parseGot9)
	}
	parseClassID := ClassIdProps("c", "d")
	if parseClassID["class"] != "c" || parseClassID["id"] != "d" {
		parseT.Fatalf("ClassIdProps: got %#v", parseClassID)
	}
	if len(EmptyProps()) != 0 {
		parseT.Fatalf("EmptyProps: expected empty map")
	}
}

func TestHTMLComponentHelpers_ExhaustiveCoverage(parseT *testing.T) {
	parseComponentA := func(parseProps map[string]any) *Element {
		return Div(parseProps, "A")
	}
	parseComponentB := func(parseProps2 map[string]any) *Element {
		return Span(parseProps2, "B")
	}

	parseWith := WithComponents("article", map[string]any{"class": "x"}, parseComponentA, parseComponentB)
	if parseWith.Type != "article" || len(parseWith.Children) != 2 {
		parseT.Fatalf("WithComponents: got %#v", parseWith)
	}

	parseDiv := DivWithComponents(nil, parseComponentA, parseComponentB)
	if parseDiv.Type != "div" || len(parseDiv.Children) != 2 {
		parseT.Fatalf("DivWithComponents: got %#v", parseDiv)
	}

	parseSection := SectionWithComponents(nil, parseComponentA, parseComponentB)
	if parseSection.Type != "section" || len(parseSection.Children) != 2 {
		parseT.Fatalf("SectionWithComponents: got %#v", parseSection)
	}

	parseMain := MainWithComponents(nil, parseComponentA, parseComponentB)
	if parseMain.Type != "main" || len(parseMain.Children) != 2 {
		parseT.Fatalf("MainWithComponents: got %#v", parseMain)
	}
}
