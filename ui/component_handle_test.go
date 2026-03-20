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
