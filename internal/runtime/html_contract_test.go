package runtime

import "testing"

func testComponentRef(props map[string]interface{}) *Element {
	return Div(props, "child")
}

func TestWithComponents_WrapsComponentRefsAsElements(t *testing.T) {
	elem := WithComponents("div", map[string]interface{}{"id": "host"}, testComponentRef)

	if elem.Type != "div" {
		t.Fatalf("expected host type div, got %v", elem.Type)
	}
	if elem.Props["id"] != "host" {
		t.Fatalf("expected host props to be preserved")
	}
	if len(elem.Children) != 1 {
		t.Fatalf("expected one child, got %d", len(elem.Children))
	}

	child, ok := elem.Children[0].(*Element)
	if !ok || child == nil {
		t.Fatal("expected component ref child to be wrapped as an element")
	}
	if child.Type == nil {
		t.Fatal("expected wrapped component child to retain its component type")
	}
	if _, ok := child.Type.(func(map[string]interface{}) *Element); !ok {
		t.Fatalf("expected wrapped child type to be component function, got %T", child.Type)
	}
	if childrenProp, ok := elem.Props["children"].([]interface{}); !ok || len(childrenProp) != 1 {
		t.Fatal("expected wrapped children to be reflected in props")
	}
}

func TestDivWithComponents_WrapsAllComponentRefs(t *testing.T) {
	elem := DivWithComponents(nil, testComponentRef, testComponentRef)

	if elem.Type != "div" {
		t.Fatalf("expected div helper to create div host, got %v", elem.Type)
	}
	if len(elem.Children) != 2 {
		t.Fatalf("expected two wrapped component children, got %d", len(elem.Children))
	}

	for i, child := range elem.Children {
		wrapped, ok := child.(*Element)
		if !ok || wrapped == nil {
			t.Fatalf("expected child %d to be wrapped as element, got %T", i, child)
		}
		if _, ok := wrapped.Type.(func(map[string]interface{}) *Element); !ok {
			t.Fatalf("expected child %d type to be a component function, got %T", i, wrapped.Type)
		}
	}
}
