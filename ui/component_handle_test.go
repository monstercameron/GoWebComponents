package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func TestCreateElementUsesStableComponentHandle(parseT *testing.T) {
	parseComponent := func() Node { return Text("hello") }
	parseFirst := CreateElement(parseComponent)
	parseSecond := CreateElement(parseComponent)
	if parseFirst == nil || parseSecond == nil {
		parseT.Fatal("expected component elements")
	}
	parseLeft, parseOk := parseFirst.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected runtime component handle, got %#v", parseFirst.Type)
	}
	parseRight, parseOk := parseSecond.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected runtime component handle, got %#v", parseSecond.Type)
	}
	if parseLeft != parseRight {
		parseT.Fatal("expected CreateElement to reuse the same component handle for the same logical component")
	}
	if parseLeft.IdentityKey() == "" {
		parseT.Fatal("expected non-empty component identity")
	}
}

func TestCreateElementDoesNotStoreImplementationInProps(parseT *testing.T) {
	parseComponent := func() Node { return Text("hello") }
	parseNode := CreateElement(parseComponent)
	if parseNode == nil {
		parseT.Fatal("expected component element")
	}
	if _, parseOk := parseNode.Props[propsKey]; parseOk {
		parseT.Fatal("expected no hidden props payload when no explicit props are provided")
	}
	if _, parseOk2 := parseNode.Props["__ui_component"]; parseOk2 {
		parseT.Fatal("expected component implementation binding to live on the handle rather than in props")
	}
}

func TestComponentAliasesCreateElement(parseT *testing.T) {
	parseComponent := func() Node { return Text("hello") }
	parseFirst := CreateElement(parseComponent)
	parseSecond := Component(parseComponent)
	if parseFirst == nil || parseSecond == nil {
		parseT.Fatal("expected component nodes")
	}
	parseLeft, parseOk := parseFirst.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected runtime component handle, got %#v", parseFirst.Type)
	}
	parseRight, parseOk := parseSecond.Type.(*runtime.ComponentType)
	if !parseOk {
		parseT.Fatalf("expected runtime component handle, got %#v", parseSecond.Type)
	}
	if parseLeft != parseRight {
		parseT.Fatal("expected Component to reuse CreateElement component identity")
	}
}

func TestIfEvaluatesLazily(parseT *testing.T) {
	parseTrueCalls := 0
	parseFalseCalls := 0
	parseNode := If(true,
		func() Node {
			parseTrueCalls++
			return Text("yes")
		},
		func() Node {
			parseFalseCalls++
			return Text("no")
		},
	)
	if parseNode == nil || parseNode.TextContent != "yes" {
		parseT.Fatalf("expected true branch node, got %#v", parseNode)
	}
	if parseTrueCalls != 1 || parseFalseCalls != 0 {
		parseT.Fatalf("expected only true branch evaluation, got true=%d false=%d", parseTrueCalls, parseFalseCalls)
	}
}

func TestMatchEvaluatesOnlyFirstMatchingBranch(parseT *testing.T) {
	parseFirstCalls := 0
	parseSecondCalls := 0
	parseDefaultCalls := 0
	parseNode := Match().
		When(false, func() Node {
			parseFirstCalls++
			return Text("first")
		}).
		When(true, func() Node {
			parseSecondCalls++
			return Text("second")
		}).
		Default(func() Node {
			parseDefaultCalls++
			return Text("default")
		})
	if parseNode == nil || parseNode.TextContent != "second" {
		parseT.Fatalf("expected second branch node, got %#v", parseNode)
	}
	if parseFirstCalls != 0 || parseSecondCalls != 1 || parseDefaultCalls != 0 {
		parseT.Fatalf("unexpected branch counts first=%d second=%d default=%d", parseFirstCalls, parseSecondCalls, parseDefaultCalls)
	}
}
