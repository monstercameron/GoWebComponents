package html

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestOnClickParallelBuildsMarkerAndHandler verifies the helper sets both the local click handler and the bridge marker.
func TestOnClickParallelBuildsMarkerAndHandler(parseT *testing.T) {
	getProps := PropsOf(OnClickParallel("primary.action", ui.WrapHandler(func() {})))
	getRuntimeProps := toRuntimeProps(getProps)
	if getRuntimeProps["onclick"] == nil {
		parseT.Fatal("expected onclick handler in runtime props")
	}
	if getRuntimeProps["data-"+parallelRegionClickSlotDataKey] != "primary.action" {
		parseT.Fatalf("expected click-slot marker %q, got %#v", "primary.action", getRuntimeProps["data-"+parallelRegionClickSlotDataKey])
	}
}
