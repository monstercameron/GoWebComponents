package ui

import "testing"

func TestHotReloadBoundaryUsesStableFragmentKey(t *testing.T) {
	child := Fragment()
	node := HotReloadBoundary(HotReloadBoundaryProps{
		Child:     child,
		ResetKeys: []interface{}{"cart-v2", 3},
	})
	if node == nil {
		t.Fatal("expected hot reload boundary node")
	}
	if got, ok := node.Type.(string); !ok || got != "FRAGMENT" {
		t.Fatalf("expected fragment boundary, got %#v", node.Type)
	}
	if got := node.Props["key"]; got != `__gwc_hotreload_boundary__:["cart-v2",3]` {
		t.Fatalf("expected serialized reset key, got %#v", got)
	}
	children, ok := node.Props["children"].([]interface{})
	if !ok || len(children) != 1 || children[0] != child {
		t.Fatalf("expected child to be preserved, got %#v", node.Props["children"])
	}
}

func TestHotReloadBoundaryDefaultKeyIsStable(t *testing.T) {
	first := HotReloadBoundary(HotReloadBoundaryProps{})
	second := HotReloadBoundary(HotReloadBoundaryProps{})
	if first == nil || second == nil {
		t.Fatal("expected hot reload boundary nodes")
	}
	if first.Props["key"] != second.Props["key"] {
		t.Fatalf("expected stable default key, got %#v and %#v", first.Props["key"], second.Props["key"])
	}
}
