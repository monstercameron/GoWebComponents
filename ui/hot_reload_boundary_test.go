package ui

import "testing"

func TestHotReloadBoundaryUsesStableFragmentKey(parseT *testing.T) {
	parseChild := Fragment()
	parseNode := HotReloadBoundary(HotReloadBoundaryProps{
		Child:     parseChild,
		ResetKeys: []interface{}{"cart-v2", 3},
	})
	if parseNode == nil {
		parseT.Fatal("expected hot reload boundary node")
	}
	if parseGot, parseOk := parseNode.Type.(string); !parseOk || parseGot != "FRAGMENT" {
		parseT.Fatalf("expected fragment boundary, got %#v", parseNode.Type)
	}
	if parseGot2 := parseNode.Props["key"]; parseGot2 != `__gwc_hotreload_boundary__:["cart-v2",3]` {
		parseT.Fatalf("expected serialized reset key, got %#v", parseGot2)
	}
	parseChildren, parseOk2 := parseNode.Props["children"].([]interface{})
	if !parseOk2 || len(parseChildren) != 1 || parseChildren[0] != parseChild {
		parseT.Fatalf("expected child to be preserved, got %#v", parseNode.Props["children"])
	}
}

func TestHotReloadBoundaryDefaultKeyIsStable(parseT *testing.T) {
	parseFirst := HotReloadBoundary(HotReloadBoundaryProps{})
	parseSecond := HotReloadBoundary(HotReloadBoundaryProps{})
	if parseFirst == nil || parseSecond == nil {
		parseT.Fatal("expected hot reload boundary nodes")
	}
	if parseFirst.Props["key"] != parseSecond.Props["key"] {
		parseT.Fatalf("expected stable default key, got %#v and %#v", parseFirst.Props["key"], parseSecond.Props["key"])
	}
}
