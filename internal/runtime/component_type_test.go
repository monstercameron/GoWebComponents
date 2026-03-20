package runtime

import "testing"

func TestComponentTypeIdentityMatchesAcrossDistinctInstances(t *testing.T) {
	left := NewComponentType("component:Example", "Example", "component:Example", nil, nil)
	right := NewComponentType("component:Example", "Example", "component:Example", nil, nil)
	if !isSameType(left, right) {
		t.Fatal("expected component handles with the same identity to compare equal")
	}
}

func TestDescribeCallableIdentityUsesComponentHandleMetadata(t *testing.T) {
	component := NewComponentType("component:Cart", "Cart", "github.com/example/Cart", nil, nil)
	pretty, qualified := describeCallableIdentity(component)
	if pretty != "Cart" {
		t.Fatalf("expected pretty name Cart, got %q", pretty)
	}
	if qualified != "component:Cart" {
		t.Fatalf("expected identity key component:Cart, got %q", qualified)
	}
}

func TestComponentTypeRenderUsesUpdatedImplementation(t *testing.T) {
	handle := NewComponentType(
		"component:Example",
		"Example",
		"component:Example",
		func() *Element { return &Element{Type: "TEXT_ELEMENT", TextContent: "first"} },
		func(implementation interface{}, props map[string]interface{}) *Element {
			return implementation.(func() *Element)()
		},
	)

	first := handle.Render(nil)
	if first == nil || first.TextContent != "first" {
		t.Fatalf("expected first implementation result, got %#v", first)
	}

	handle.SetImplementation(func() *Element { return &Element{Type: "TEXT_ELEMENT", TextContent: "second"} })
	second := handle.Render(nil)
	if second == nil || second.TextContent != "second" {
		t.Fatalf("expected updated implementation result, got %#v", second)
	}
}
