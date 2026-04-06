package html

import "testing"

// TestOnClickParallelBuildsMarkerAndHandler verifies the helper sets both the local click handler and the bridge marker.
func TestOnClickParallelBuildsMarkerAndHandler(parseT *testing.T) {
	getProps := PropsOf(OnClickParallel("primary.action", func() {}))
	getRuntimeProps := toRuntimeProps(getProps)
	if getRuntimeProps["onclick"] == nil {
		parseT.Fatal("expected onclick handler in runtime props")
	}
	if getRuntimeProps["data-"+parallelRegionClickSlotDataKey] != "primary.action" {
		parseT.Fatalf("expected click-slot marker %q, got %#v", "primary.action", getRuntimeProps["data-"+parallelRegionClickSlotDataKey])
	}
}
