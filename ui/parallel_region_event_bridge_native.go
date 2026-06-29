//go:build !js || !wasm

package ui

import (
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// buildParallelRegionBridgedNode strips bridge-only event slot markers on non-browser targets without changing local handler behavior.
func buildParallelRegionBridgedNode(
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseNode Node,
) (Node, runtime2.EventSlotMetadata, error) {
	_ = parseRuntimeSpec
	parseBridgeErr := handleParallelRegionEachNode(parseNode, func(parseCurrentNode Node) error {
		if parseCurrentNode == nil || parseCurrentNode.Props == nil {
			return nil
		}
		delete(parseCurrentNode.Props, parallelRegionClickSlotProp)
		runtime.RefreshElementHostProps(parseCurrentNode)
		return nil
	})
	if parseBridgeErr != nil {
		return nil, runtime2.EventSlotMetadata{}, parseBridgeErr
	}
	return parseNode, runtime2.EventSlotMetadata{}, nil
}
