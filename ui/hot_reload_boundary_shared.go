package ui

import "github.com/monstercameron/GoWebComponents/internal/runtime"

type HotReloadBoundaryProps struct {
	Child     Node
	Children  []Node
	ResetKeys []interface{}
}

// HotReloadBoundary wraps a subtree in a keyed fragment so changing ResetKeys
// forces that subtree to remount during the next render, including hot reload.
func HotReloadBoundary(props HotReloadBoundaryProps) Node {
	children := make([]interface{}, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, toInterfaces(props.Children)...)

	return runtime.CreateElement("FRAGMENT", map[string]interface{}{
		"key": hotReloadBoundaryKey(props.ResetKeys),
	}, children...)
}
