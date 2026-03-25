package runtime

import "testing"

func testComponentRef(parseProps map[string]interface{}) *Element {
	return Div(parseProps, "child")
}

func TestWithComponents_WrapsComponentRefsAsElements(parseT *testing.T) {
	parseElem := WithComponents("div", map[string]interface{}{"id": "host"}, testComponentRef)

	if parseElem.Type != "div" {
		parseT.Fatalf("expected host type div, got %v", parseElem.Type)
	}
	if parseElem.Props["id"] != "host" {
		parseT.Fatalf("expected host props to be preserved")
	}
	if len(parseElem.Children) != 1 {
		parseT.Fatalf("expected one child, got %d", len(parseElem.Children))
	}

	parseChild, parseOk := parseElem.Children[0].(*Element)
	if !parseOk || parseChild == nil {
		parseT.Fatal("expected component ref child to be wrapped as an element")
	}
	if parseChild.Type == nil {
		parseT.Fatal("expected wrapped component child to retain its component type")
	}
	if _, parseOk2 := parseChild.Type.(func(map[string]interface{}) *Element); !parseOk2 {
		parseT.Fatalf("expected wrapped child type to be component function, got %T", parseChild.Type)
	}
	if parseChildrenProp, parseOk3 := parseElem.Props["children"].([]interface{}); !parseOk3 || len(parseChildrenProp) != 1 {
		parseT.Fatal("expected wrapped children to be reflected in props")
	}
}

func TestDivWithComponents_WrapsAllComponentRefs(parseT *testing.T) {
	parseElem := DivWithComponents(nil, testComponentRef, testComponentRef)

	if parseElem.Type != "div" {
		parseT.Fatalf("expected div helper to create div host, got %v", parseElem.Type)
	}
	if len(parseElem.Children) != 2 {
		parseT.Fatalf("expected two wrapped component children, got %d", len(parseElem.Children))
	}

	for parseI, parseChild := range parseElem.Children {
		parseWrapped, parseOk := parseChild.(*Element)
		if !parseOk || parseWrapped == nil {
			parseT.Fatalf("expected child %d to be wrapped as element, got %T", parseI, parseChild)
		}
		if _, parseOk2 := parseWrapped.Type.(func(map[string]interface{}) *Element); !parseOk2 {
			parseT.Fatalf("expected child %d type to be a component function, got %T", parseI, parseWrapped.Type)
		}
	}
}
