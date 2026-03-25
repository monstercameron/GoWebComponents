package ui

import "github.com/monstercameron/GoWebComponents/internal/runtime"

type HotReloadBoundaryProps struct {
	Child     Node
	Children  []Node
	ResetKeys []interface{}
}

// HotReloadBoundary wraps a subtree in a keyed fragment so changing ResetKeys
// forces that subtree to remount during the next render, including hot reload.
func HotReloadBoundary(parseProps HotReloadBoundaryProps) Node {
	parseChildren := make([]interface{}, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, toInterfaces(parseProps.Children)...)

	return runtime.CreateElement("FRAGMENT", map[string]interface{}{
		"key": hotReloadBoundaryKey(parseProps.ResetKeys),
	}, parseChildren...)
}
