package ui

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

type buildParallelRegionRuntimeSpecProps struct {
	Label string
	Count int
}

type registerParallelRegionProps struct {
	Label string
}

type buildParallelRegionSource struct {
	getSourceIDs []string
}

// ReactiveRegionSourceIDs reports stable source IDs for one test source binding.
func (parseSource buildParallelRegionSource) ReactiveRegionSourceIDs() []string {
	return append([]string(nil), parseSource.getSourceIDs...)
}

// TestBuildParallelRegionRuntimeSpecMapsPublicSpec verifies the public generic spec maps into the runtime2 spec contract.
func TestBuildParallelRegionRuntimeSpecMapsPublicSpec(parseT *testing.T) {
	getRuntimeSpec, parseErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[buildParallelRegionRuntimeSpecProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:summary",
		Props: buildParallelRegionRuntimeSpecProps{
			Label: "Hot",
			Count: 3,
		},
		SourceIDs: []string{"status", "hot", "status"},
	})
	if parseErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseErr)
	}
	if getRuntimeSpec.RendererID != runtime2.RendererID("dashboard.hot-panel") {
		parseT.Fatalf("runtime renderer ID = %q, want %q", getRuntimeSpec.RendererID, "dashboard.hot-panel")
	}
	if getRuntimeSpec.RegionInstanceID != runtime2.RegionInstanceID("dashboard.hot-panel:summary") {
		parseT.Fatalf("runtime region instance ID = %q, want %q", getRuntimeSpec.RegionInstanceID, "dashboard.hot-panel:summary")
	}
	getProps, hasProps := getRuntimeSpec.Props.(buildParallelRegionRuntimeSpecProps)
	if !hasProps {
		parseT.Fatalf("runtime props type = %T, want %T", getRuntimeSpec.Props, buildParallelRegionRuntimeSpecProps{})
	}
	if getProps != (buildParallelRegionRuntimeSpecProps{Label: "Hot", Count: 3}) {
		parseT.Fatalf("runtime props = %+v, want %+v", getProps, buildParallelRegionRuntimeSpecProps{Label: "Hot", Count: 3})
	}
	if !reflect.DeepEqual(getRuntimeSpec.SourceIDs, []string{"hot", "status"}) {
		parseT.Fatalf("runtime source IDs = %+v, want %+v", getRuntimeSpec.SourceIDs, []string{"hot", "status"})
	}
}

// TestBuildParallelRegionRuntimeSpecRejectsInvalidPublicSpec verifies public mapping keeps runtime2 validation guarantees.
func TestBuildParallelRegionRuntimeSpecRejectsInvalidPublicSpec(parseT *testing.T) {
	_, parseErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[map[string]any]{
		RendererID:       "",
		RegionInstanceID: "region-1",
		Props:            map[string]any{"label": "bad"},
	})
	if parseErr == nil {
		parseT.Fatal("expected invalid public parallel-region spec to fail")
	}
}

// TestRegisterParallelRegionBridgesRuntime2Registry verifies public registration stores the renderer and bridges into runtime2.
func TestRegisterParallelRegionBridgesRuntime2Registry(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseRender := func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", parseRender); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	getRender, parseResolveErr := resolveParallelRegionRenderer("dashboard.hot-panel")
	if parseResolveErr != nil {
		parseT.Fatalf("resolveParallelRegionRenderer returned error: %v", parseResolveErr)
	}
	if reflect.ValueOf(getRender).Pointer() != reflect.ValueOf(parseRender).Pointer() {
		parseT.Fatal("expected resolved public renderer to match the registered renderer")
	}
	if _, _, parseResolveRuntimeErr := runtime2.ResolveRenderer(runtime2.RendererID("dashboard.hot-panel")); parseResolveRuntimeErr != nil {
		parseT.Fatalf("runtime2.ResolveRenderer returned error: %v", parseResolveRuntimeErr)
	}
}

// TestBuildParallelRegionSourceIDsPreservesDeclaredOrder verifies public source binding preserves declared order while deduplicating.
func TestBuildParallelRegionSourceIDsPreservesDeclaredOrder(parseT *testing.T) {
	getSourceIDs, parseErr := BuildParallelRegionSourceIDs(
		buildParallelRegionSource{getSourceIDs: []string{"status", "count"}},
		buildParallelRegionSource{getSourceIDs: []string{"count", "user.id"}},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildParallelRegionSourceIDs returned error: %v", parseErr)
	}
	if !reflect.DeepEqual(getSourceIDs, []string{"status", "count", "user.id"}) {
		parseT.Fatalf("public source IDs = %+v, want %+v", getSourceIDs, []string{"status", "count", "user.id"})
	}
}

// TestBuildParallelRegionSourceIDsRejectsInvalidSourceID verifies invalid bound source IDs fail before runtime2 dispatch.
func TestBuildParallelRegionSourceIDsRejectsInvalidSourceID(parseT *testing.T) {
	_, parseErr := BuildParallelRegionSourceIDs(buildParallelRegionSource{getSourceIDs: []string{"bad source"}})
	if parseErr == nil {
		parseT.Fatal("expected invalid bound source ID to fail")
	}
}

// TestParallelRegionBuildsLocalFirstShell verifies public parallel regions render local-first content inside a stable shell marker.
func TestParallelRegionBuildsLocalFirstShell(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	getNode := ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:summary",
		Props: registerParallelRegionProps{
			Label: "Hot",
		},
	})
	if getNode == nil {
		parseT.Fatal("expected ParallelRegion to return a shell node")
	}
	if getNode.Type != "div" {
		parseT.Fatalf("shell node type = %v, want div", getNode.Type)
	}
	getShellMarkerRaw, hasShellMarker := getNode.Props[runtime2.SSRShellMarkerAttribute]
	if !hasShellMarker {
		parseT.Fatalf("expected shell props to include %q", runtime2.SSRShellMarkerAttribute)
	}
	getShellMarker, hasShellMarkerString := getShellMarkerRaw.(string)
	if !hasShellMarkerString {
		parseT.Fatalf("shell marker type = %T, want string", getShellMarkerRaw)
	}
	getMarker, parseMarkerErr := runtime2.ParseSSRShellMarkerAttributeValue(getShellMarker)
	if parseMarkerErr != nil {
		parseT.Fatalf("ParseSSRShellMarkerAttributeValue returned error: %v", parseMarkerErr)
	}
	if getMarker.RegionInstanceID != runtime2.RegionInstanceID("dashboard.hot-panel:summary") {
		parseT.Fatalf("shell marker region instance ID = %q, want %q", getMarker.RegionInstanceID, "dashboard.hot-panel:summary")
	}
	if getMarker.RendererID != runtime2.RendererID("dashboard.hot-panel") {
		parseT.Fatalf("shell marker renderer ID = %q, want %q", getMarker.RendererID, "dashboard.hot-panel")
	}
	if len(getNode.Children) != 1 {
		parseT.Fatalf("shell child count = %d, want 1", len(getNode.Children))
	}
	getChildNode, hasChildNode := getNode.Children[0].(Node)
	if !hasChildNode {
		parseT.Fatalf("shell child type = %T, want ui.Node", getNode.Children[0])
	}
	if getChildNode.TextContent != "Hot" {
		parseT.Fatalf("shell child text = %q, want %q", getChildNode.TextContent, "Hot")
	}
}
