package runtime

import "testing"

func TestComponentTypeIdentityMatchesAcrossDistinctInstances(parseT *testing.T) {
	parseLeft := NewComponentType("component:Example", "Example", "component:Example", nil, nil)
	parseRight := NewComponentType("component:Example", "Example", "component:Example", nil, nil)
	if !isSameType(parseLeft, parseRight) {
		parseT.Fatal("expected component handles with the same identity to compare equal")
	}
}

func TestDescribeCallableIdentityUsesComponentHandleMetadata(parseT *testing.T) {
	parseComponent := NewComponentType("component:Cart", "Cart", "github.com/example/Cart", nil, nil)
	parsePretty, parseQualified := describeCallableIdentity(parseComponent)
	if parsePretty != "Cart" {
		parseT.Fatalf("expected pretty name Cart, got %q", parsePretty)
	}
	if parseQualified != "component:Cart" {
		parseT.Fatalf("expected identity key component:Cart, got %q", parseQualified)
	}
}

func TestComponentTypeRenderUsesUpdatedImplementation(parseT *testing.T) {
	handle := NewComponentType(
		"component:Example",
		"Example",
		"component:Example",
		func() *Element { return &Element{Type: "TEXT_ELEMENT", TextContent: "first"} },
		func(parseImplementation any, parseProps map[string]any) *Element {
			return parseImplementation.(func() *Element)()
		},
	)

	parseFirst := handle.Render(nil)
	if parseFirst == nil || parseFirst.TextContent != "first" {
		parseT.Fatalf("expected first implementation result, got %#v", parseFirst)
	}

	handle.SetImplementation(func() *Element { return &Element{Type: "TEXT_ELEMENT", TextContent: "second"} })
	parseSecond := handle.Render(nil)
	if parseSecond == nil || parseSecond.TextContent != "second" {
		parseT.Fatalf("expected updated implementation result, got %#v", parseSecond)
	}
}

func TestComponentTypeRenderUsesUpdatedRendererWhenSignatureChanges(parseT *testing.T) {
	handle := NewComponentType(
		"component:Example",
		"Example",
		"component:Example",
		func() *Element { return &Element{Type: "TEXT_ELEMENT", TextContent: "first"} },
		func(parseImplementation any, parseProps map[string]any) *Element {
			return parseImplementation.(func() *Element)()
		},
	)

	handle.SetImplementationRenderer(
		func(parseProps map[string]any) *Element {
			parseText, _ := parseProps["label"].(string)
			return &Element{Type: "TEXT_ELEMENT", TextContent: parseText}
		},
		func(parseImplementation any, parseProps map[string]any) *Element {
			return parseImplementation.(func(map[string]any) *Element)(parseProps)
		},
	)

	parseRendered := handle.Render(map[string]any{"label": "second"})
	if parseRendered == nil || parseRendered.TextContent != "second" {
		parseT.Fatalf("expected updated renderer result, got %#v", parseRendered)
	}
}
