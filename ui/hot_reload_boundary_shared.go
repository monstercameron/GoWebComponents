package ui

import "github.com/monstercameron/GoWebComponents/internal/runtime"

type HotReloadBoundaryProps struct {
	Child     Node
	Children  []Node
	ResetKeys []interface{}
}

// HotReloadBoundary wraps a subtree in a keyed fragment so changing ResetKeys
// forces that subtree to remount during the next render, including hot reload.
func HotReloadBoundary(parseBoundaryProps HotReloadBoundaryProps) Node {
	parseBoundaryChildren := make([]interface{}, 0, len(parseBoundaryProps.Children)+1)
	if parseBoundaryProps.Child != nil {
		parseBoundaryChildren = append(parseBoundaryChildren, parseBoundaryProps.Child)
	}
	parseBoundaryChildren = append(parseBoundaryChildren, toInterfaces(parseBoundaryProps.Children)...)

	return runtime.CreateElementOwned("FRAGMENT", map[string]interface{}{
		"key": hotReloadBoundaryKey(parseBoundaryProps.ResetKeys),
	}, parseBoundaryChildren...)
}
