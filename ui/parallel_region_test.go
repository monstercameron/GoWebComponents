package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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

func assertParallelRegionRenderError(parseT *testing.T, parseName string, parseFn func() error, parseContains string) {
	parseT.Helper()
	parseErr := parseFn()
	if parseErr == nil {
		parseT.Fatalf("%s: expected render error", parseName)
	}
	getMessage := fmt.Sprint(parseErr)
	if parseContains != "" && !strings.Contains(getMessage, parseContains) {
		parseT.Fatalf("%s: error = %q, want substring %q", parseName, getMessage, parseContains)
	}
}

// skipParallelRegionSSRTestOnWASM skips RenderToString-based parallel-region tests when wasm browser lifecycle support is active.
func skipParallelRegionSSRTestOnWASM(parseT *testing.T) {
	parseT.Helper()
	if canParallelRegionUseRuntime2Lifecycle() {
		parseT.Skip("RenderToString-based ParallelRegion assertions are covered by native tests; wasm coverage lives in ui_wasm_test.go")
	}
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
	_, getRuntimeMetadata, parseResolveRuntimeErr := runtime2.ResolveRenderer(runtime2.RendererID("dashboard.hot-panel"))
	if parseResolveRuntimeErr != nil {
		parseT.Fatalf("runtime2.ResolveRenderer returned error: %v", parseResolveRuntimeErr)
	}
	if !runtime2.HasRendererFeatureFlag(getRuntimeMetadata, "display-only") {
		parseT.Fatalf("expected runtime2 renderer metadata to include display-only feature flag, got %+v", getRuntimeMetadata)
	}
	if _, hasWorkerRenderer := cacheParallelRegionWorkerRuntime.GetWorkerRegionState("dashboard.hot-panel"); hasWorkerRenderer {
		parseT.Fatal("did not expect worker runtime state before mount")
	}
}

// TestBuildParallelRegionWorkerRenderOutputWrapsShell verifies the worker bridge converts cached public nodes into shell-wrapped runtime2 render output.
func TestBuildParallelRegionWorkerRenderOutputWrapsShell(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	storeParallelRegionRenderedNode(
		runtime2.RegionInstanceID("dashboard.hot-panel:worker-output"),
		Fragment(Text("hello")),
	)
	parseRenderOutput, parseRenderErr := buildParallelRegionWorkerRenderOutput(
		"dashboard.hot-panel:worker-output",
		"",
		nil,
		nil,
	)
	if parseRenderErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerRenderOutput returned error: %v", parseRenderErr)
	}
	parseCanonicalIR, parseCanonicalErr := runtime2.BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseRegionDOMIndex := runtime2.BuildRegionDOMIndex()
	if _, parseApplyErr := runtime2.ApplyRegionDOMCanonicalSnapshot(parseRegionDOMIndex, "dashboard.hot-panel:worker-output", parseCanonicalIR); parseApplyErr != nil {
		parseT.Fatalf("ApplyRegionDOMCanonicalSnapshot returned error: %v", parseApplyErr)
	}
	parseRootNode, parseRootLookupErr := parseRegionDOMIndex.GetRegionDOMNode("dashboard.hot-panel:worker-output", parseCanonicalIR.GetRootNodeID)
	if parseRootLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseRootLookupErr)
	}
	if parseRootNode.GetTag != "div" {
		parseT.Fatalf("worker bridge root tag = %q, want %q", parseRootNode.GetTag, "div")
	}
}

// TestBuildParallelRegionWorkerPropsOutputStripsInteractiveProps verifies display-only worker output drops local event handlers and bridge-only markers.
func TestBuildParallelRegionWorkerPropsOutputStripsInteractiveProps(parseT *testing.T) {
	getPropsOutput, getNodeKey, parsePropsErr := buildParallelRegionWorkerPropsOutput(map[string]interface{}{
		"key":                       "slot-1",
		"onclick":                   func() {},
		parallelRegionClickSlotProp: "primary.action",
		"data-testid":               "action",
	})
	if parsePropsErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerPropsOutput returned error: %v", parsePropsErr)
	}
	if getNodeKey != "slot-1" {
		parseT.Fatalf("worker props key = %q, want %q", getNodeKey, "slot-1")
	}
	if _, hasClickHandler := getPropsOutput["onclick"]; hasClickHandler {
		parseT.Fatalf("expected onclick to be stripped from worker props, got %+v", getPropsOutput)
	}
	if _, hasClickSlot := getPropsOutput[parallelRegionClickSlotProp]; hasClickSlot {
		parseT.Fatalf("expected click-slot marker to be stripped from worker props, got %+v", getPropsOutput)
	}
	if getPropsOutput["data-testid"] != "action" {
		parseT.Fatalf("expected unrelated props to remain, got %+v", getPropsOutput)
	}
}

// TestBuildParallelRegionBridgedNodeStripsInternalClickSlotMarker verifies native builds strip bridge-only click-slot markers before local render output escapes.
func TestBuildParallelRegionBridgedNodeStripsInternalClickSlotMarker(parseT *testing.T) {
	getNode := runtime.CreateElement("button", map[string]interface{}{
		parallelRegionClickSlotProp: "primary.action",
		"onclick":                   func() {},
	})
	getBridgedNode, getEventSlotMetadata, parseBridgeErr := buildParallelRegionBridgedNode(runtime2.ParallelRegionSpec{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:strip-marker",
	}, Node(getNode))
	if parseBridgeErr != nil {
		parseT.Fatalf("buildParallelRegionBridgedNode returned error: %v", parseBridgeErr)
	}
	if getEventSlotMetadata.Version != "" || len(getEventSlotMetadata.Slots) != 0 {
		parseT.Fatalf("expected native bridge to keep empty event-slot metadata, got %+v", getEventSlotMetadata)
	}
	if _, hasMarker := getBridgedNode.Props[parallelRegionClickSlotProp]; hasMarker {
		parseT.Fatalf("expected native bridge to strip click-slot marker, got %+v", getBridgedNode.Props)
	}
}

// TestHandleParallelRegionWorkerUpdateCommitsPatch verifies the public bridge can mount worker state and commit one follow-up patch into runtime2 host status.
func TestHandleParallelRegionWorkerUpdateCommitsPatch(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	parseRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:bridge-commit",
		Props: registerParallelRegionProps{
			Label: "One",
		},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}
	parseHostRegionAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(
		parseRuntimeSpec.RegionInstanceID,
		[]runtime2.SchedulerShardID{"ui-parallel-region"},
	)
	if parseHostAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseHostAdapterErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(parseRuntimeSpec, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseFirstNode, parseFirstNodeErr := buildParallelRegionLocalNode(
		func(parseProps registerParallelRegionProps) Node {
			return Text(parseProps.Label)
		},
		parseRuntimeSpec.Props,
	)
	if parseFirstNodeErr != nil {
		parseT.Fatalf("buildParallelRegionLocalNode(first) returned error: %v", parseFirstNodeErr)
	}
	storeParallelRegionRenderedNode(parseRuntimeSpec.RegionInstanceID, parseFirstNode)
	buildParallelRegionNextInputVersion(parseRuntimeSpec.RegionInstanceID)
	if parseWorkerMountErr := handleParallelRegionWorkerMount(parseHostRegionAdapter, parseRuntimeSpec, 1); parseWorkerMountErr != nil {
		parseT.Fatalf("handleParallelRegionWorkerMount returned error: %v", parseWorkerMountErr)
	}
	parseRuntimeSpec.Props = registerParallelRegionProps{
		Label: "Two",
	}
	parseSecondNode, parseSecondNodeErr := buildParallelRegionLocalNode(
		func(parseProps registerParallelRegionProps) Node {
			return Text(parseProps.Label)
		},
		parseRuntimeSpec.Props,
	)
	if parseSecondNodeErr != nil {
		parseT.Fatalf("buildParallelRegionLocalNode(second) returned error: %v", parseSecondNodeErr)
	}
	storeParallelRegionRenderedNode(parseRuntimeSpec.RegionInstanceID, parseSecondNode)
	getInputVersion := buildParallelRegionNextInputVersion(parseRuntimeSpec.RegionInstanceID)
	parseDispatchResult, parseDispatchErr := handleParallelRegionUpdateDispatch(parseHostRegionAdapter, parseRuntimeSpec, getInputVersion)
	if parseDispatchErr != nil {
		parseT.Fatalf("handleParallelRegionUpdateDispatch returned error: %v", parseDispatchErr)
	}
	if parseWorkerUpdateErr := handleParallelRegionWorkerUpdate(parseHostRegionAdapter, parseRuntimeSpec, getInputVersion, parseDispatchResult); parseWorkerUpdateErr != nil {
		parseT.Fatalf("handleParallelRegionWorkerUpdate returned error: %v", parseWorkerUpdateErr)
	}
	parseRuntimeStatus, hasRuntimeStatus := parseHostRegionAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		parseT.Fatal("expected host runtime status after worker patch commit")
	}
	if parseRuntimeStatus.GetLastCommittedVersion != 2 {
		parseT.Fatalf("committed version = %d, want 2", parseRuntimeStatus.GetLastCommittedVersion)
	}
}

// TestRegisterParallelRegionRejectsDuplicatePublicRegistration verifies duplicate public registration fails clearly.
func TestRegisterParallelRegionRejectsDuplicatePublicRegistration(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseRender := func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", parseRender); parseErr != nil {
		parseT.Fatalf("first RegisterParallelRegion returned error: %v", parseErr)
	}
	parseErr := RegisterParallelRegion("dashboard.hot-panel", parseRender)
	if parseErr == nil {
		parseT.Fatal("expected duplicate RegisterParallelRegion to fail")
	}
	if !strings.Contains(parseErr.Error(), "already registered") {
		parseT.Fatalf("duplicate RegisterParallelRegion error = %q, want already-registered guidance", parseErr.Error())
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

// TestBuildParallelRegionNextInputVersionIncrementsAndResets verifies browser-side public region input versions advance monotonically and clear on owner removal.
func TestBuildParallelRegionNextInputVersionIncrementsAndResets(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:summary"); getInputVersion != 0 {
		parseT.Fatalf("expected zero input version before tracking begins, got %d", getInputVersion)
	}
	if getInputVersion := buildParallelRegionNextInputVersion(runtime2.RegionInstanceID("dashboard.hot-panel:summary")); getInputVersion != 1 {
		parseT.Fatalf("expected first input version 1, got %d", getInputVersion)
	}
	if getInputVersion := buildParallelRegionNextInputVersion(runtime2.RegionInstanceID("dashboard.hot-panel:summary")); getInputVersion != 2 {
		parseT.Fatalf("expected second input version 2, got %d", getInputVersion)
	}
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:summary"); getInputVersion != 2 {
		parseT.Fatalf("expected resolved input version 2, got %d", getInputVersion)
	}
	if parseErr := handleParallelRegionOwnerRemove("dashboard.hot-panel:summary"); parseErr != nil {
		parseT.Fatalf("handleParallelRegionOwnerRemove returned error: %v", parseErr)
	}
	if getInputVersion := resolveParallelRegionInputVersion("dashboard.hot-panel:summary"); getInputVersion != 0 {
		parseT.Fatalf("expected input version to clear after owner removal, got %d", getInputVersion)
	}
}

// TestGetParallelRegionRuntimeStatusRejectsInvalidRegionInstanceID verifies the public status helper keeps region-instance validation.
func TestGetParallelRegionRuntimeStatusRejectsInvalidRegionInstanceID(parseT *testing.T) {
	_, _, parseErr := GetParallelRegionRuntimeStatus("")
	if parseErr == nil {
		parseT.Fatal("expected invalid public region-instance ID to fail")
	}
}

// TestGetParallelRegionRuntimeStatusReportsMissingForUntrackedRegion verifies the public status helper stays read-only when no runtime2 adapter exists.
func TestGetParallelRegionRuntimeStatusReportsMissingForUntrackedRegion(parseT *testing.T) {
	getStatus, hasStatus, parseErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:missing")
	if parseErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseErr)
	}
	if hasStatus {
		parseT.Fatalf("expected missing public region status, got %+v", getStatus)
	}
}

// TestParallelRegionBuildsLocalFirstShell verifies public parallel regions render local-first content inside a stable shell marker.
func TestParallelRegionBuildsLocalFirstShell(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	getMarkup, parseRenderErr := RenderToString(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:summary",
		Props: registerParallelRegionProps{
			Label: "Hot",
		},
	}))
	if parseRenderErr != nil {
		parseT.Fatalf("RenderToString(ParallelRegion) returned error: %v", parseRenderErr)
	}
	if !strings.Contains(getMarkup, runtime2.SSRShellMarkerAttribute) {
		parseT.Fatalf("expected shell markup to include %q, got %q", runtime2.SSRShellMarkerAttribute, getMarkup)
	}
	getNode := renderParallelRegionComponent(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:summary",
		Props: registerParallelRegionProps{
			Label: "Hot",
		},
	})
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
	if getNode.Type != "div" {
		parseT.Fatalf("shell node type = %v, want div", getNode.Type)
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

// TestParallelRegionNativeFallbackKeepsLocalOnlyRendering verifies non-browser builds do not attach runtime2 host lifecycle state.
func TestParallelRegionNativeFallbackKeepsLocalOnlyRendering(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}

	getMarkup, parseRenderErr := RenderToString(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:native",
		Props: registerParallelRegionProps{
			Label: "Native",
		},
	}))
	if parseRenderErr != nil {
		parseT.Fatalf("RenderToString(ParallelRegion) returned error: %v", parseRenderErr)
	}
	if !strings.Contains(getMarkup, "Native") {
		parseT.Fatalf("expected native ParallelRegion markup to include rendered content, got %q", getMarkup)
	}
	if _, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter("dashboard.hot-panel:native"); hasParallelRegionHostAdapter {
		parseT.Fatal("did not expect native ParallelRegion to mount a runtime2 host adapter")
	}
}

// TestParallelRegionRejectsMissingRendererAtPublicUILayer verifies missing public registrations fail at render time with actionable guidance.
func TestParallelRegionRejectsMissingRendererAtPublicUILayer(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	assertParallelRegionRenderError(parseT, "missing renderer", func() error {
		_, parseErr := RenderToString(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
			RendererID:       "dashboard.hot-panel",
			RegionInstanceID: "dashboard.hot-panel:missing",
			Props: registerParallelRegionProps{
				Label: "Missing",
			},
		}))
		return parseErr
	}, "renderer resolution failed")
}

// TestParallelRegionRejectsInvalidPropsAtPublicUILayer verifies invalid public props fail before runtime2 dispatch.
func TestParallelRegionRejectsInvalidPropsAtPublicUILayer(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps map[string]any) Node {
		return Text("never")
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	assertParallelRegionRenderError(parseT, "invalid props", func() error {
		_, parseErr := RenderToString(ParallelRegion(ParallelRegionSpec[map[string]any]{
			RendererID:       "dashboard.hot-panel",
			RegionInstanceID: "dashboard.hot-panel:invalid-props",
			Props: map[string]any{
				"onClick": func() {},
			},
		}))
		return parseErr
	}, "unsupported event-closure prop")
}

// TestParallelRegionRejectsInvalidSourceIDAtPublicUILayer verifies invalid source IDs fail through the public ui surface.
func TestParallelRegionRejectsInvalidSourceIDAtPublicUILayer(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	assertParallelRegionRenderError(parseT, "invalid source id", func() error {
		_, parseErr := RenderToString(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
			RendererID:       "dashboard.hot-panel",
			RegionInstanceID: "dashboard.hot-panel:invalid-source",
			Props: registerParallelRegionProps{
				Label: "Invalid",
			},
			SourceIDs: []string{"bad source"},
		}))
		return parseErr
	}, "source ID")
}

// TestParallelRegionRejectsInvalidRegionInstanceIDAtPublicUILayer verifies invalid region IDs fail through the public ui surface.
func TestParallelRegionRejectsInvalidRegionInstanceIDAtPublicUILayer(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	assertParallelRegionRenderError(parseT, "invalid region instance id", func() error {
		_, parseErr := RenderToString(ParallelRegion(ParallelRegionSpec[registerParallelRegionProps]{
			RendererID:       "dashboard.hot-panel",
			RegionInstanceID: "",
			Props: registerParallelRegionProps{
				Label: "Invalid",
			},
		}))
		return parseErr
	}, "region instance ID is required")
}
