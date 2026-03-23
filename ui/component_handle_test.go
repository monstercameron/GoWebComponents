package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func TestCreateElementUsesStableComponentHandle(t *testing.T) {
	component := func() Node { return Text("hello") }
	first := CreateElement(component)
	second := CreateElement(component)
	if first == nil || second == nil {
		t.Fatal("expected component elements")
	}
	left, ok := first.Type.(*runtime.ComponentType)
	if !ok {
		t.Fatalf("expected runtime component handle, got %#v", first.Type)
	}
	right, ok := second.Type.(*runtime.ComponentType)
	if !ok {
		t.Fatalf("expected runtime component handle, got %#v", second.Type)
	}
	if left != right {
		t.Fatal("expected CreateElement to reuse the same component handle for the same logical component")
	}
	if left.IdentityKey() == "" {
		t.Fatal("expected non-empty component identity")
	}
}

func TestCreateElementDoesNotStoreImplementationInProps(t *testing.T) {
	component := func() Node { return Text("hello") }
	node := CreateElement(component)
	if node == nil {
		t.Fatal("expected component element")
	}
	if _, ok := node.Props[propsKey]; ok {
		t.Fatal("expected no hidden props payload when no explicit props are provided")
	}
	if _, ok := node.Props["__ui_component"]; ok {
		t.Fatal("expected component implementation binding to live on the handle rather than in props")
	}
}

func TestComponentAliasesCreateElement(t *testing.T) {
	component := func() Node { return Text("hello") }
	first := CreateElement(component)
	second := Component(component)
	if first == nil || second == nil {
		t.Fatal("expected component nodes")
	}
	left, ok := first.Type.(*runtime.ComponentType)
	if !ok {
		t.Fatalf("expected runtime component handle, got %#v", first.Type)
	}
	right, ok := second.Type.(*runtime.ComponentType)
	if !ok {
		t.Fatalf("expected runtime component handle, got %#v", second.Type)
	}
	if left != right {
		t.Fatal("expected Component to reuse CreateElement component identity")
	}
}

func TestIfEvaluatesLazily(t *testing.T) {
	trueCalls := 0
	falseCalls := 0
	node := If(true,
		func() Node {
			trueCalls++
			return Text("yes")
		},
		func() Node {
			falseCalls++
			return Text("no")
		},
	)
	if node == nil || node.TextContent != "yes" {
		t.Fatalf("expected true branch node, got %#v", node)
	}
	if trueCalls != 1 || falseCalls != 0 {
		t.Fatalf("expected only true branch evaluation, got true=%d false=%d", trueCalls, falseCalls)
	}
}

func TestMatchEvaluatesOnlyFirstMatchingBranch(t *testing.T) {
	firstCalls := 0
	secondCalls := 0
	defaultCalls := 0
	node := Match().
		When(false, func() Node {
			firstCalls++
			return Text("first")
		}).
		When(true, func() Node {
			secondCalls++
			return Text("second")
		}).
		Default(func() Node {
			defaultCalls++
			return Text("default")
		})
	if node == nil || node.TextContent != "second" {
		t.Fatalf("expected second branch node, got %#v", node)
	}
	if firstCalls != 0 || secondCalls != 1 || defaultCalls != 0 {
		t.Fatalf("unexpected branch counts first=%d second=%d default=%d", firstCalls, secondCalls, defaultCalls)
	}
}
