package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestDataAttrSingleAttribute pins the zero-alloc single data-* field on both
// construction lanes and its coexistence with the Data map.
func TestDataAttrSingleAttribute(parseT *testing.T) {
	getMarkup, parseErr := ui.RenderToString(Div(Props{
		Class:    "row",
		DataAttr: DataAttribute{Name: "row-id", Value: "7"},
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if !strings.Contains(getMarkup, `data-row-id="7"`) {
		parseT.Fatalf("compact lane missing data attr: %s", getMarkup)
	}

	getCombined, parseErr2 := ui.RenderToString(Div(Props{
		DataAttr: DataAttribute{Name: "row-id", Value: "7"},
		Data:     map[string]string{"zone": "a"},
	}))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString combined: %v", parseErr2)
	}
	if !strings.Contains(getCombined, `data-row-id="7"`) || !strings.Contains(getCombined, `data-zone="a"`) {
		parseT.Fatalf("combined attrs missing: %s", getCombined)
	}
	if strings.Index(getCombined, "data-row-id") > strings.Index(getCombined, "data-zone") {
		parseT.Fatalf("data segment not sorted: %s", getCombined)
	}

	// Map lane (event handler forces it): the attr must still land.
	getMapLane, parseErr3 := ui.RenderToString(Button(Props{
		DataAttr: DataAttribute{Name: "action", Value: "go"},
		OnClick:  ui.WrapHandler(func() {}),
	}, Text("x")))
	if parseErr3 != nil {
		parseT.Fatalf("RenderToString map lane: %v", parseErr3)
	}
	if !strings.Contains(getMapLane, `data-action="go"`) {
		parseT.Fatalf("map lane missing data attr: %s", getMapLane)
	}
}

// TestPropsTextDirectContent pins the zero-child-element text field on both
// lanes and its ignored-with-children contract.
func TestPropsTextDirectContent(parseT *testing.T) {
	getMarkup, parseErr := ui.RenderToString(Div(Props{Class: "row", Text: "hello direct"}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if !strings.Contains(getMarkup, ">hello direct<") || !strings.Contains(getMarkup, `class="row"`) {
		parseT.Fatalf("compact direct text wrong: %s", getMarkup)
	}

	getMapLane, parseErr2 := ui.RenderToString(Button(Props{Text: "click", OnClick: ui.WrapHandler(func() {})}))
	if parseErr2 != nil {
		parseT.Fatalf("map lane: %v", parseErr2)
	}
	if !strings.Contains(getMapLane, ">click<") {
		parseT.Fatalf("map lane direct text wrong: %s", getMapLane)
	}

	getWithChildren, parseErr3 := ui.RenderToString(Div(Props{Text: "ignored"}, Span(Props{}, Text("child"))))
	if parseErr3 != nil {
		parseT.Fatalf("with children: %v", parseErr3)
	}
	if strings.Contains(getWithChildren, "ignored") {
		parseT.Fatalf("Text must be ignored when children are passed: %s", getWithChildren)
	}
}
