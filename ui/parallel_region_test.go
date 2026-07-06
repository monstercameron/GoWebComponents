package ui

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/pluginruntime"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
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
	getPreviousPanicOptions := runtime.CurrentUnhandledPanicLoggingOptions()
	runtime.ConfigureUnhandledPanicLogging(runtime.PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport:           getPreviousPanicOptions.OnReport,
	})
	defer runtime.ConfigureUnhandledPanicLogging(getPreviousPanicOptions)
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
	parseRenderOutput, parseRenderErr := buildParallelRegionWorkerRenderOutput(runtime2.WorkerRegionMountSpec{
		RegionID: "dashboard.hot-panel:worker-output",
	})
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
	getPropsOutput, getNodeKey, parsePropsErr := buildParallelRegionWorkerPropsOutput(map[string]any{
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
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: mockdom.NewMockDOMAdapter(),
		Reset:      true,
	})

	getNode := runtime.CreateElement("button", map[string]any{
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
	if canParallelRegionUseRuntime2Lifecycle() {
		if getEventSlotMetadata.Version == "" || len(getEventSlotMetadata.Slots) != 1 {
			parseT.Fatalf("expected wasm bridge to record one click slot, got %+v", getEventSlotMetadata)
		}
		if getEventSlotMetadata.Slots[0].SlotID != "primary.action" || getEventSlotMetadata.Slots[0].EventType != parallelRegionClickEventType {
			parseT.Fatalf("expected click-slot metadata for primary.action, got %+v", getEventSlotMetadata)
		}
	} else if getEventSlotMetadata.Version != "" || len(getEventSlotMetadata.Slots) != 0 {
		parseT.Fatalf("expected native bridge to keep empty event-slot metadata, got %+v", getEventSlotMetadata)
	}
	if _, hasMarker := getBridgedNode.Props[parallelRegionClickSlotProp]; hasMarker {
		parseT.Fatalf("expected native bridge to strip click-slot marker, got %+v", getBridgedNode.Props)
	}
	if canParallelRegionUseRuntime2Lifecycle() {
		parseHandlerType := reflect.TypeOf(getBridgedNode.Props["onclick"])
		if parseHandlerType == nil || parseHandlerType.String() != "func(runtime.GoEvent)" {
			parseT.Fatalf("expected wasm bridge to wrap onclick handler, got %T", getBridgedNode.Props["onclick"])
		}
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

// TestSetParallelRegionWorkerRendererUsesExplicitWorkerRenderOutput verifies configured public regions can provide worker-native render output without cached ui.Node conversion.
func TestSetParallelRegionWorkerRendererUsesExplicitWorkerRenderOutput(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	if parseErr := SetParallelRegionWorkerRendererWithConfig(
		"dashboard.hot-panel",
		func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
			parseProps, hasProps := parseMount.Snapshot.Props.(registerParallelRegionProps)
			if !hasProps {
				return nil, fmt.Errorf("worker props type = %T", parseMount.Snapshot.Props)
			}
			return map[string]any{
				"kind": "text",
				"text": parseProps.Label + " worker",
			}, nil
		},
		ParallelRegionWorkerRendererConfig{
			IsTrusted:                        true,
			HasUpdateValidationOverride:      true,
			ShouldValidateUpdateRenderOutput: false,
		},
	); parseErr != nil {
		parseT.Fatalf("SetParallelRegionWorkerRendererWithConfig returned error: %v", parseErr)
	}
	parseRenderOutput, parseRenderErr := buildParallelRegionWorkerRenderOutput(runtime2.WorkerRegionMountSpec{
		RegionID:   "dashboard.hot-panel:worker-native",
		RendererID: "dashboard.hot-panel",
		Snapshot: runtime2.SnapshotEnvelope{
			Props: registerParallelRegionProps{Label: "Hot"},
		},
	})
	if parseRenderErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerRenderOutput(explicit) returned error: %v", parseRenderErr)
	}
	parseCanonicalIR, parseCanonicalErr := runtime2.BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseRegionDOMIndex := runtime2.BuildRegionDOMIndex()
	if _, parseApplyErr := runtime2.ApplyRegionDOMCanonicalSnapshot(parseRegionDOMIndex, "dashboard.hot-panel:worker-native", parseCanonicalIR); parseApplyErr != nil {
		parseT.Fatalf("ApplyRegionDOMCanonicalSnapshot returned error: %v", parseApplyErr)
	}
	getRootNode, parseRootLookupErr := parseRegionDOMIndex.GetRegionDOMNode("dashboard.hot-panel:worker-native", parseCanonicalIR.GetRootNodeID)
	if parseRootLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseRootLookupErr)
	}
	if len(getRootNode.GetChildNodeIDs) != 1 {
		parseT.Fatalf("root child count = %d, want 1", len(getRootNode.GetChildNodeIDs))
	}
	getChildNode, parseChildLookupErr := parseRegionDOMIndex.GetRegionDOMNode("dashboard.hot-panel:worker-native", getRootNode.GetChildNodeIDs[0])
	if parseChildLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(child) returned error: %v", parseChildLookupErr)
	}
	if getChildNode.GetText != "Hot worker" {
		parseT.Fatalf("worker child text = %q, want %q", getChildNode.GetText, "Hot worker")
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
	// A single plain Text child is stored directly on the shell host element
	// (direct-text fast path) instead of as a separate text child node.
	if len(getNode.Children) != 0 || getNode.TextContent != "Hot" {
		parseT.Fatalf("shell direct text = %q (children %#v), want %q with no child nodes", getNode.TextContent, getNode.Children, "Hot")
	}
}

// TestParallelRegionNativeFallbackKeepsLocalOnlyRendering verifies non-browser builds do not attach runtime2 host lifecycle state.
func TestParallelRegionNativeFallbackKeepsLocalOnlyRendering(parseT *testing.T) {
	skipParallelRegionSSRTestOnWASM(parseT)
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return runtime.CreateElement("button", map[string]any{
			parallelRegionClickSlotProp: "primary.action",
			"onclick":                   func() {},
		}, Text(parseProps.Label))
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

// TestHandleParallelRegionEachNodeVisitsDepthFirstAndStopsOnError verifies subtree traversal order and early exit behavior.
func TestHandleParallelRegionEachNodeVisitsDepthFirstAndStopsOnError(parseT *testing.T) {
	getTree := runtime.CreateElement("section", nil,
		runtime.CreateElement("h1", nil),
		"ignored-text-child",
		runtime.CreateElement("button", nil, runtime.CreateElement("span", nil)),
	)
	var getVisitedTags []string
	parseVisitErr := handleParallelRegionEachNode(Node(getTree), func(parseNode Node) error {
		if parseTag, parseOk := parseNode.Type.(string); parseOk {
			getVisitedTags = append(getVisitedTags, parseTag)
		}
		if parseNode.Type == "button" {
			return fmt.Errorf("stop")
		}
		return nil
	})
	if parseVisitErr == nil || !strings.Contains(parseVisitErr.Error(), "stop") {
		parseT.Fatalf("expected visit error to stop traversal, got %v", parseVisitErr)
	}
	if !reflect.DeepEqual(getVisitedTags, []string{"section", "h1", "TEXT_ELEMENT", "button"}) {
		parseT.Fatalf("visited tags = %+v, want depth-first stop at button", getVisitedTags)
	}
	if parseNilErr := handleParallelRegionEachNode(nil, nil); parseNilErr != nil {
		parseT.Fatalf("expected nil root traversal to no-op, got %v", parseNilErr)
	}
}

// TestBuildParallelRegionEventSlotMetadataAndMergeNormalizeSlots verifies slot metadata validation, empty handling, and deduplicating merges.
func TestBuildParallelRegionEventSlotMetadataAndMergeNormalizeSlots(parseT *testing.T) {
	getMetadata, parseMetadataErr := buildParallelRegionEventSlotMetadata([]runtime2.EventSlotRecord{{
		SlotID:    "primary.action",
		EventType: parallelRegionClickEventType,
	}})
	if parseMetadataErr != nil {
		parseT.Fatalf("buildParallelRegionEventSlotMetadata(valid) returned error: %v", parseMetadataErr)
	}
	if getMetadata.Version != runtime2.EventSlotMetadataVersionV1 || len(getMetadata.Slots) != 1 {
		parseT.Fatalf("event-slot metadata = %+v, want one normalized slot", getMetadata)
	}
	getEmptyMetadata, parseEmptyErr := buildParallelRegionEventSlotMetadata(nil)
	if parseEmptyErr != nil {
		parseT.Fatalf("buildParallelRegionEventSlotMetadata(empty) returned error: %v", parseEmptyErr)
	}
	if getEmptyMetadata.Version != "" || len(getEmptyMetadata.Slots) != 0 {
		parseT.Fatalf("empty event-slot metadata = %+v, want zero value", getEmptyMetadata)
	}
	if _, parseInvalidErr := buildParallelRegionEventSlotMetadata([]runtime2.EventSlotRecord{{
		SlotID:    "",
		EventType: parallelRegionClickEventType,
	}}); parseInvalidErr == nil {
		parseT.Fatal("expected invalid event-slot metadata to fail")
	}
	getMergedMetadata, parseMergeErr := buildParallelRegionMergedEventSlotMetadata(
		runtime2.EventSlotMetadata{
			Version: runtime2.EventSlotMetadataVersionV1,
			Slots: []runtime2.EventSlotRecord{{
				SlotID:    "primary.action",
				EventType: parallelRegionClickEventType,
			}},
		},
		runtime2.EventSlotMetadata{
			Version: runtime2.EventSlotMetadataVersionV1,
			Slots: []runtime2.EventSlotRecord{
				{SlotID: "primary.action", EventType: parallelRegionClickEventType},
				{SlotID: "secondary.action", EventType: "keydown"},
			},
		},
	)
	if parseMergeErr != nil {
		parseT.Fatalf("buildParallelRegionMergedEventSlotMetadata returned error: %v", parseMergeErr)
	}
	if !reflect.DeepEqual(getMergedMetadata.Slots, []runtime2.EventSlotRecord{
		{SlotID: "primary.action", EventType: parallelRegionClickEventType},
		{SlotID: "secondary.action", EventType: "keydown"},
	}) {
		parseT.Fatalf("merged event-slot metadata = %+v", getMergedMetadata)
	}
}

// TestParallelRegionWorkerPropFiltersRecognizeBridgeOnlyProps verifies worker-output prop stripping covers both event props and bridge markers.
func TestParallelRegionWorkerPropFiltersRecognizeBridgeOnlyProps(parseT *testing.T) {
	if !hasParallelRegionWorkerEventProp("onclick") || !hasParallelRegionWorkerEventProp("onscroll") {
		parseT.Fatal("expected interactive worker props to be detected")
	}
	if hasParallelRegionWorkerEventProp("data-testid") {
		parseT.Fatal("did not expect non-event prop to be treated as worker event prop")
	}
	if !shouldParallelRegionStripWorkerProp(parallelRegionClickSlotProp, nil) || !shouldParallelRegionStripWorkerProp("onchange", nil) {
		parseT.Fatal("expected bridge-only props to be stripped")
	}
	if shouldParallelRegionStripWorkerProp("class", "btn") {
		parseT.Fatal("did not expect ordinary props to be stripped")
	}
	// A function-valued prop of ANY name must be stripped (it cannot be JSON-encoded
	// into worker props; leaving it in silently fails the first patch encode).
	if !shouldParallelRegionStripWorkerProp("onCustomHandler", func() {}) {
		parseT.Fatal("expected an arbitrarily-named function-valued prop to be stripped")
	}
	if !shouldParallelRegionStripWorkerProp("data-cb", func(int) string { return "" }) {
		parseT.Fatal("expected any function value to be stripped regardless of name")
	}
}

// TestParallelRegionBridgeFallbackStoreSurfacesAndClears pins the #83 HIGH fix: a
// worker-bridge failure (mount/update/click) is RECORDED per region so it can be
// surfaced on ParallelRegionStatus.GetFallbackReason instead of only logged, and it
// is cleared on recovery and on registry reset so a stale reason never lingers.
func TestParallelRegionBridgeFallbackStoreSurfacesAndClears(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)

	parseID := runtime2.RegionInstanceID("region-fallback-x")
	if parseGot := resolveParallelRegionBridgeFallback(parseID); parseGot != "" {
		parseT.Fatalf("expected no recorded fallback initially, got %q", parseGot)
	}

	recordParallelRegionBridgeFallback(parseID, "worker mount bridge failed: boom")
	if parseGot := resolveParallelRegionBridgeFallback(parseID); parseGot != "worker mount bridge failed: boom" {
		parseT.Fatalf("recorded fallback not surfaced, got %q", parseGot)
	}

	// Empty region or reason must be ignored (never overwrites a recorded reason).
	recordParallelRegionBridgeFallback("", "ignored")
	recordParallelRegionBridgeFallback(parseID, "")
	if parseGot := resolveParallelRegionBridgeFallback(parseID); parseGot != "worker mount bridge failed: boom" {
		parseT.Fatalf("empty record must not clobber the reason, got %q", parseGot)
	}

	// Recovery clears the reason.
	clearParallelRegionBridgeFallback(parseID)
	if parseGot := resolveParallelRegionBridgeFallback(parseID); parseGot != "" {
		parseT.Fatalf("clear must drop the recorded fallback, got %q", parseGot)
	}

	// Registry reset clears any recorded reason.
	recordParallelRegionBridgeFallback(parseID, "again")
	resetParallelRegionRegistry()
	if parseGot := resolveParallelRegionBridgeFallback(parseID); parseGot != "" {
		parseT.Fatalf("reset must clear recorded fallbacks, got %q", parseGot)
	}
}

// TestParallelRegionWorkerPropsOutputStripsFunctionValuesAndStaysJSONEncodable pins the
// #83 func-prop encode fix end to end: an arbitrarily-named function-valued prop is
// removed from the display-only worker props so the result marshals cleanly, while
// ordinary props survive.
func TestParallelRegionWorkerPropsOutputStripsFunctionValuesAndStaysJSONEncodable(parseT *testing.T) {
	parseOut, _, parseErr := buildParallelRegionWorkerPropsOutput(map[string]any{
		"class":           "btn",
		"data-testid":     "widget",
		"onCustomHandler": func() {},
		"reducer":         func(int) int { return 0 },
	})
	if parseErr != nil {
		parseT.Fatalf("building worker props with function values must not error, got %v", parseErr)
	}
	if _, hasFunc := parseOut["onCustomHandler"]; hasFunc {
		parseT.Fatal("function-valued prop must be stripped from worker props output")
	}
	if _, hasReducer := parseOut["reducer"]; hasReducer {
		parseT.Fatal("any function value must be stripped from worker props output")
	}
	if parseOut["class"] != "btn" || parseOut["data-testid"] != "widget" {
		parseT.Fatalf("ordinary props must survive stripping, got %+v", parseOut)
	}
	// The stripped output must be JSON-encodable (the real patch pipeline marshals it).
	if _, parseMarshalErr := json.Marshal(parseOut); parseMarshalErr != nil {
		parseT.Fatalf("worker props output must be JSON-encodable after stripping, got %v", parseMarshalErr)
	}
}

// TestReportParallelRegionDiagnosticErrorAndSharedRuntimeHelpers verifies shared runtime bridge helpers expose diagnostics, shells, transitions, and source atoms.
func TestReportParallelRegionDiagnosticErrorAndSharedRuntimeHelpers(parseT *testing.T) {
	runtime.ClearDiagnostics()
	defer runtime.ClearDiagnostics()
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: mockdom.NewMockDOMAdapter(), Reset: true})
	parseRuntime := runtime.GetGlobalRuntime()
	if parseSetErr := parseRuntime.SetAtomValue("dashboard.count", 7); parseSetErr != nil {
		parseT.Fatalf("SetAtomValue returned error: %v", parseSetErr)
	}
	reportParallelRegionDiagnosticError("parallel-region shared helper failure")
	getDiagnostics := runtime.GetDiagnostics()
	if len(getDiagnostics) != 1 || getDiagnostics[0].Source != "ui" || getDiagnostics[0].Severity != runtime.DiagnosticError {
		parseT.Fatalf("diagnostics = %+v, want one ui error diagnostic", getDiagnostics)
	}
	getEmptyShell := renderParallelRegionShellNode(map[string]any{"id": "shell"}, nil)
	if getEmptyShell.Type != "div" || len(getEmptyShell.Children) != 0 {
		parseT.Fatalf("empty shell = %+v, want div without children", getEmptyShell)
	}
	getChildShell := renderParallelRegionShellNode(map[string]any{"id": "shell"}, Text("hot"))
	// A single plain Text child is stored directly on the shell host element
	// (direct-text fast path) instead of as a separate text child node.
	if len(getChildShell.Children) != 0 || getChildShell.TextContent != "hot" {
		parseT.Fatalf("shell direct text = %q (children %#v), want %q with no child nodes", getChildShell.TextContent, getChildShell.Children, "hot")
	}
	if isParallelRegionTransitionUpdate() {
		parseT.Fatal("did not expect transition update without current fiber")
	}
	getSourceValue, hasSourceValue := getParallelRegionSourceAtomValue("dashboard.count")
	if !hasSourceValue || getSourceValue != 7 {
		parseT.Fatalf("source atom value = %#v, ok=%t, want 7,true", getSourceValue, hasSourceValue)
	}
}

// TestHandleParallelRegionRendererMetadataMergesSlotsIntoRegistry verifies slot declarations update the shared renderer registry without duplicating entries.
func TestHandleParallelRegionRendererMetadataMergesSlotsIntoRegistry(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	if parseErr := handleParallelRegionRendererMetadata("dashboard.hot-panel", runtime2.EventSlotMetadata{}); parseErr != nil {
		parseT.Fatalf("handleParallelRegionRendererMetadata(empty) returned error: %v", parseErr)
	}
	getMetadata := runtime2.EventSlotMetadata{
		Version: runtime2.EventSlotMetadataVersionV1,
		Slots: []runtime2.EventSlotRecord{{
			SlotID:    "primary.action",
			EventType: parallelRegionClickEventType,
		}},
	}
	if parseErr := handleParallelRegionRendererMetadata("dashboard.hot-panel", getMetadata); parseErr != nil {
		parseT.Fatalf("handleParallelRegionRendererMetadata(first) returned error: %v", parseErr)
	}
	if parseErr := handleParallelRegionRendererMetadata("dashboard.hot-panel", getMetadata); parseErr != nil {
		parseT.Fatalf("handleParallelRegionRendererMetadata(second) returned error: %v", parseErr)
	}
	_, getResolvedMetadata, parseResolveErr := runtime2.ResolveRenderer("dashboard.hot-panel")
	if parseResolveErr != nil {
		parseT.Fatalf("ResolveRenderer returned error: %v", parseResolveErr)
	}
	if len(getResolvedMetadata.EventSlotMetadata.Slots) != 1 || getResolvedMetadata.EventSlotMetadata.Slots[0].SlotID != "primary.action" {
		parseT.Fatalf("resolved renderer metadata = %+v, want one merged slot", getResolvedMetadata.EventSlotMetadata)
	}
}

// TestHandleParallelRegionPostRenderAttachHandlesMissingAndInvalidAnchors verifies post-render attach tolerates missing anchors and rejects mismatched shell tags.
func TestHandleParallelRegionPostRenderAttachHandlesMissingAndInvalidAnchors(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	if parseErr := handleParallelRegionPostRenderAttachByID(""); parseErr != nil {
		parseT.Fatalf("handleParallelRegionPostRenderAttachByID(empty) returned error: %v", parseErr)
	}
	if parseErr := handleParallelRegionPostRenderAttachByID("dashboard.hot-panel:missing"); parseErr != nil {
		parseT.Fatalf("handleParallelRegionPostRenderAttachByID(missing) returned error: %v", parseErr)
	}
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:attach",
		Props:            registerParallelRegionProps{Label: "Attach"},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}
	getHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(getRuntimeSpec.RegionInstanceID, []runtime2.SchedulerShardID{"ui-parallel-region"})
	if parseHostAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseHostAdapterErr)
	}
	if _, parseMountErr := getHostAdapter.HandleHostRegionMount(getRuntimeSpec, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	storeParallelRegionAdapterMu.Lock()
	cacheParallelRegionAdapterByID[getRuntimeSpec.RegionInstanceID] = getHostAdapter
	storeParallelRegionAdapterMu.Unlock()
	if parseAttachErr := handleParallelRegionPostRenderAttachByID(getRuntimeSpec.RegionInstanceID); parseAttachErr != nil {
		parseT.Fatalf("handleParallelRegionPostRenderAttachByID(missing-anchor) returned error: %v", parseAttachErr)
	}
	getAnchorNode, parseAnchorLookupErr := getHostAdapter.GetHostRegionDOMIndex().GetRegionDOMNode(string(getRuntimeSpec.RegionInstanceID), 1)
	if parseAnchorLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(anchor) returned error: %v", parseAnchorLookupErr)
	}
	if getAnchorNode.GetTag != "div" {
		parseT.Fatalf("anchor tag = %q, want div", getAnchorNode.GetTag)
	}
	if parseSetErr := getHostAdapter.GetHostRegionDOMIndex().SetRegionDOMNode(string(getRuntimeSpec.RegionInstanceID), 1, &runtime2.RegionDOMNode{
		GetNodeID: 1,
		GetTag:    "span",
	}); parseSetErr != nil {
		parseT.Fatalf("SetRegionDOMNode returned error: %v", parseSetErr)
	}
	if parseAttachErr := handleParallelRegionPostRenderAttachByID(getRuntimeSpec.RegionInstanceID); parseAttachErr == nil || !strings.Contains(parseAttachErr.Error(), "shell anchor tag") {
		parseT.Fatalf("expected invalid anchor tag error, got %v", parseAttachErr)
	}
}

// TestBuildParallelRegionSchedulerShardIDsValidateInputs verifies default shards, trimming, and duplicate or blank validation.
func TestBuildParallelRegionSchedulerShardIDsValidateInputs(parseT *testing.T) {
	getDefaultShards, parseDefaultErr := buildParallelRegionSchedulerShardIDs(nil)
	if parseDefaultErr != nil {
		parseT.Fatalf("buildParallelRegionSchedulerShardIDs(default) returned error: %v", parseDefaultErr)
	}
	if !reflect.DeepEqual(getDefaultShards, []runtime2.SchedulerShardID{"ui-parallel-region"}) {
		parseT.Fatalf("default scheduler shards = %+v", getDefaultShards)
	}
	getCustomShards, parseCustomErr := buildParallelRegionSchedulerShardIDs([]string{" primary ", "secondary"})
	if parseCustomErr != nil {
		parseT.Fatalf("buildParallelRegionSchedulerShardIDs(custom) returned error: %v", parseCustomErr)
	}
	if !reflect.DeepEqual(getCustomShards, []runtime2.SchedulerShardID{"primary", "secondary"}) {
		parseT.Fatalf("custom scheduler shards = %+v", getCustomShards)
	}
	if _, parseBlankErr := buildParallelRegionSchedulerShardIDs([]string{" "}); parseBlankErr == nil {
		parseT.Fatal("expected blank scheduler shard ID to fail")
	}
	if _, parseDuplicateErr := buildParallelRegionSchedulerShardIDs([]string{"primary", "primary"}); parseDuplicateErr == nil {
		parseT.Fatal("expected duplicate scheduler shard IDs to fail")
	}
	getReactiveSources := buildParallelRegionReactiveSources([]string{"status", "user.id"})
	if len(getReactiveSources) != 2 {
		parseT.Fatalf("reactive source count = %d, want 2", len(getReactiveSources))
	}
}

// TestBuildParallelRegionHostAdapterHandlesReuseRemountAndShardChanges verifies cached adapters reuse on identical specs, remount on renderer changes, and recreate on shard changes.
func TestBuildParallelRegionHostAdapterHandlesReuseRemountAndShardChanges(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:host-adapter",
		Props:            registerParallelRegionProps{Label: "One"},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}
	getAdapter, getMounted, parseBuildErr := buildParallelRegionHostAdapter(getRuntimeSpec, []runtime2.SchedulerShardID{"primary"})
	if parseBuildErr != nil {
		parseT.Fatalf("buildParallelRegionHostAdapter(first) returned error: %v", parseBuildErr)
	}
	if !getMounted {
		parseT.Fatal("expected first host adapter build to mount")
	}
	getSameAdapter, getRemounted, parseSameErr := buildParallelRegionHostAdapter(getRuntimeSpec, []runtime2.SchedulerShardID{"primary"})
	if parseSameErr != nil {
		parseT.Fatalf("buildParallelRegionHostAdapter(second) returned error: %v", parseSameErr)
	}
	if getSameAdapter != getAdapter || getRemounted {
		parseT.Fatalf("expected same adapter without remount, got same=%t remounted=%t", getSameAdapter == getAdapter, getRemounted)
	}
	getRuntimeSpec.RendererID = "dashboard.hot-panel-alt"
	getRemountAdapter, getDidRemount, parseRemountErr := buildParallelRegionHostAdapter(getRuntimeSpec, []runtime2.SchedulerShardID{"primary"})
	if parseRemountErr != nil {
		parseT.Fatalf("buildParallelRegionHostAdapter(remount) returned error: %v", parseRemountErr)
	}
	if getRemountAdapter != getAdapter || !getDidRemount {
		parseT.Fatalf("expected structural remount on renderer change, got same=%t remounted=%t", getRemountAdapter == getAdapter, getDidRemount)
	}
	getRecreatedAdapter, getDidRecreate, parseShardErr := buildParallelRegionHostAdapter(getRuntimeSpec, []runtime2.SchedulerShardID{"secondary"})
	if parseShardErr != nil {
		parseT.Fatalf("buildParallelRegionHostAdapter(shard-change) returned error: %v", parseShardErr)
	}
	if getRecreatedAdapter == getAdapter || !getDidRecreate {
		parseT.Fatalf("expected shard change to recreate adapter, got same=%t recreated=%t", getRecreatedAdapter == getAdapter, getDidRecreate)
	}
	if _, parseNilRemountErr := handleParallelRegionStructuralRemount(nil, getRuntimeSpec); parseNilRemountErr == nil {
		parseT.Fatal("expected nil host adapter structural remount to fail")
	}
	if hasParallelRegionSchedulerShardChange([]runtime2.SchedulerShardID{"a"}, []runtime2.SchedulerShardID{"a"}) {
		parseT.Fatal("did not expect identical scheduler shard lists to report changes")
	}
	if !hasParallelRegionSchedulerShardChange([]runtime2.SchedulerShardID{"a"}, []runtime2.SchedulerShardID{"b"}) {
		parseT.Fatal("expected different scheduler shard lists to report changes")
	}
}

// TestBuildParallelRegionSourceSnapshotValidatesTrackingAndAvailability verifies declared source snapshots require tracked versions and available atoms.
func TestBuildParallelRegionSourceSnapshotValidatesTrackingAndAvailability(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: mockdom.NewMockDOMAdapter(), Reset: true})
	getEmptyValues, getEmptyVersions, parseEmptyErr := buildParallelRegionSourceSnapshot("dashboard.hot-panel:sources", nil)
	if parseEmptyErr != nil {
		parseT.Fatalf("buildParallelRegionSourceSnapshot(empty) returned error: %v", parseEmptyErr)
	}
	if len(getEmptyValues) != 0 || len(getEmptyVersions) != 0 {
		parseT.Fatalf("empty source snapshot = values:%+v versions:%+v", getEmptyValues, getEmptyVersions)
	}
	if _, _, parseVersionErr := buildParallelRegionSourceSnapshot("dashboard.hot-panel:sources", []string{"count"}); parseVersionErr == nil {
		parseT.Fatal("expected untracked input version to fail")
	}
	buildParallelRegionNextInputVersion("dashboard.hot-panel:sources")
	if _, _, parseMissingErr := buildParallelRegionSourceSnapshot("dashboard.hot-panel:sources", []string{"count"}); parseMissingErr == nil {
		parseT.Fatal("expected missing source atom to fail")
	}
	if parseSetErr := runtime.GetGlobalRuntime().SetAtomValue("count", 3); parseSetErr != nil {
		parseT.Fatalf("SetAtomValue returned error: %v", parseSetErr)
	}
	if _, _, parseWhitespaceErr := buildParallelRegionSourceSnapshot("dashboard.hot-panel:sources", []string{" count "}); parseWhitespaceErr == nil {
		parseT.Fatal("expected source IDs with surrounding whitespace to fail")
	}
	getSourceValues, getSourceVersions, parseSourceErr := buildParallelRegionSourceSnapshot("dashboard.hot-panel:sources", []string{"count"})
	if parseSourceErr != nil {
		parseT.Fatalf("buildParallelRegionSourceSnapshot(valid) returned error: %v", parseSourceErr)
	}
	if getSourceValues["count"] != 3 || getSourceVersions["count"] != 1 {
		parseT.Fatalf("source snapshot = values:%+v versions:%+v", getSourceValues, getSourceVersions)
	}
}

// TestBuildParallelRegionLocalNodeRejectsInvalidRendererShapes verifies renderer shape, props conversion, and non-node return handling.
func TestBuildParallelRegionLocalNodeRejectsInvalidRendererShapes(parseT *testing.T) {
	if _, parseCallErr := buildParallelRegionLocalNode(nil, nil); parseCallErr == nil {
		parseT.Fatal("expected non-callable renderer to fail")
	}
	if _, parseArityErr := buildParallelRegionLocalNode(func() Node { return Text("bad") }, nil); parseArityErr == nil {
		parseT.Fatal("expected zero-arg renderer to fail")
	}
	if _, parseReturnErr := buildParallelRegionLocalNode(func(parseProps registerParallelRegionProps) string { return parseProps.Label }, registerParallelRegionProps{Label: "bad"}); parseReturnErr == nil {
		parseT.Fatal("expected non-node renderer return type to fail without panicking")
	}
	type renderParallelRegionAliasProps registerParallelRegionProps
	getNode, parseNodeErr := buildParallelRegionLocalNode(func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}, renderParallelRegionAliasProps{Label: "converted"})
	if parseNodeErr != nil {
		parseT.Fatalf("buildParallelRegionLocalNode(convertible) returned error: %v", parseNodeErr)
	}
	if getNode == nil || getNode.TextContent != "converted" {
		parseT.Fatalf("converted node = %+v, want text node", getNode)
	}
	getNilNode, parseNilErr := buildParallelRegionLocalNode(func(parseProps registerParallelRegionProps) Node {
		return nil
	}, registerParallelRegionProps{})
	if parseNilErr != nil || getNilNode != nil {
		parseT.Fatalf("nil renderer node = %+v err=%v, want nil,nil", getNilNode, parseNilErr)
	}
	if _, parseMismatchErr := buildParallelRegionLocalNode(func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}, "bad-props"); parseMismatchErr == nil {
		parseT.Fatal("expected mismatched props type to fail")
	}
}

// TestBuildParallelRegionWorkerBridgeCoversConversionBranches verifies worker render conversion handles shell, props, fragments, and unsupported nodes clearly.
func TestBuildParallelRegionWorkerBridgeCoversConversionBranches(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	runtime2.ResetCapabilityReport()
	parseT.Cleanup(runtime2.ResetCapabilityReport)
	getFallbackReport := buildParallelRegionWorkerCapabilityReport()
	if !getFallbackReport.HasStructuredCloneSupport || !getFallbackReport.HasWorkerSupport {
		parseT.Fatalf("fallback capability report = %+v, want structured-clone fallback", getFallbackReport)
	}
	if parseOverrideErr := runtime2.SetCapabilityReportOverride(runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasBinaryTransportSupport: true,
	}); parseOverrideErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride returned error: %v", parseOverrideErr)
	}
	getOverrideReport := buildParallelRegionWorkerCapabilityReport()
	if !getOverrideReport.HasBinaryTransportSupport {
		parseT.Fatalf("override capability report = %+v, want binary transport support", getOverrideReport)
	}
	getShellOutput, parseShellErr := buildParallelRegionWorkerShellOutput(nil)
	if parseShellErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerShellOutput(nil) returned error: %v", parseShellErr)
	}
	getShellMap, hasShellMap := getShellOutput.(map[string]any)
	if !hasShellMap || getShellMap["tag"] != "div" {
		parseT.Fatalf("shell output = %#v, want div shell map", getShellOutput)
	}
	getButtonNode := runtime.CreateElement("button", map[string]any{
		"key":         "action-1",
		"class":       "primary",
		"children":    "ignored",
		"data-testid": "cta",
	}, Text("Click"))
	getNodeOutput, parseNodeErr := buildParallelRegionWorkerNodeOutput(Node(getButtonNode))
	if parseNodeErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerNodeOutput(host) returned error: %v", parseNodeErr)
	}
	getElementOutput, hasElementOutput := getNodeOutput.(map[string]any)
	if !hasElementOutput || getElementOutput["tag"] != "button" || getElementOutput["key"] != "action-1" {
		parseT.Fatalf("worker host output = %#v", getNodeOutput)
	}
	getFragmentOutput, parseFragmentErr := buildParallelRegionWorkerNodeOutput(Fragment(Text("One"), Text("Two")))
	if parseFragmentErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerNodeOutput(fragment) returned error: %v", parseFragmentErr)
	}
	getFragmentMap, hasFragmentMap := getFragmentOutput.(map[string]any)
	if !hasFragmentMap || getFragmentMap["kind"] != "fragment" {
		parseT.Fatalf("worker fragment output = %#v", getFragmentOutput)
	}
	if _, parsePortalErr := buildParallelRegionWorkerNodeOutput(&runtime.Element{Type: runtime.PortalNodeType}); parsePortalErr == nil {
		parseT.Fatal("expected portal node conversion to fail")
	}
	if _, parseReactiveRegionErr := buildParallelRegionWorkerNodeOutput(&runtime.Element{Type: runtime.ReactiveRegionNodeType}); parseReactiveRegionErr == nil {
		parseT.Fatal("expected reactive-region node conversion to fail")
	}
	if _, parseReactiveTextErr := buildParallelRegionWorkerNodeOutput(&runtime.Element{Type: runtime.ReactiveTextNodeType}); parseReactiveTextErr == nil {
		parseT.Fatal("expected reactive-text node conversion to fail")
	}
	if _, parseUnknownErr := buildParallelRegionWorkerNodeOutput(&runtime.Element{Type: 123}); parseUnknownErr == nil {
		parseT.Fatal("expected unknown node conversion to fail")
	}
	getChildrenOutput, parseChildrenErr := buildParallelRegionWorkerChildrenOutput([]any{nil, Text("One"), "Two"})
	if parseChildrenErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerChildrenOutput(valid) returned error: %v", parseChildrenErr)
	}
	if len(getChildrenOutput) != 2 {
		parseT.Fatalf("worker children output count = %d, want 2", len(getChildrenOutput))
	}
	if _, parseChildTypeErr := buildParallelRegionWorkerChildrenOutput([]any{1}); parseChildTypeErr == nil {
		parseT.Fatal("expected unsupported child type to fail")
	}
	if _, _, parseKeyErr := buildParallelRegionWorkerPropsOutput(map[string]any{"key": 7}); parseKeyErr == nil {
		parseT.Fatal("expected non-string worker key prop to fail")
	}
}

// TestBuildParallelRegionWorkerRenderOutputAndUpdateEdgeBranches verifies worker render recache, no-schedule updates, nil adapters, and missing scheduled snapshots.
func TestBuildParallelRegionWorkerRenderOutputAndUpdateEdgeBranches(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter: mockdom.NewMockDOMAdapter(),
		Reset:      true,
	})
	if parseErr := RegisterParallelRegion("dashboard.hot-panel", func(parseProps registerParallelRegionProps) Node {
		return runtime.CreateElement("button", map[string]any{
			parallelRegionClickSlotProp: "primary.action",
			"onclick":                   func() {},
		}, Text(parseProps.Label))
	}); parseErr != nil {
		parseT.Fatalf("RegisterParallelRegion returned error: %v", parseErr)
	}
	getRenderOutput, parseRenderErr := buildParallelRegionWorkerRenderOutput(runtime2.WorkerRegionMountSpec{
		RegionID:   "dashboard.hot-panel:worker-recache",
		RendererID: "dashboard.hot-panel",
		Snapshot: runtime2.SnapshotEnvelope{
			Props: registerParallelRegionProps{Label: "Worker"},
		},
		RenderInput: runtime2.WorkerRenderInput{
			GetEventSlot: &runtime2.EventSlotDispatch{SlotID: "primary.action", EventType: parallelRegionClickEventType},
		},
	})
	if parseRenderErr != nil {
		parseT.Fatalf("buildParallelRegionWorkerRenderOutput(recache) returned error: %v", parseRenderErr)
	}
	if getRenderOutput == nil {
		parseT.Fatal("expected worker render output after event-slot recache")
	}
	getRecachedNode, hasRecachedNode := resolveParallelRegionRenderedNode("dashboard.hot-panel:worker-recache")
	if !hasRecachedNode {
		parseT.Fatal("expected worker event-slot recache to store one rendered node")
	}
	if getRecachedNode == nil {
		parseT.Fatal("expected worker event-slot recache to keep one non-nil rendered node")
	}
	if canParallelRegionUseRuntime2Lifecycle() {
		parseHandlerType := reflect.TypeOf(getRecachedNode.Props["onclick"])
		if parseHandlerType == nil || parseHandlerType.String() != "func(runtime.GoEvent)" {
			parseT.Fatalf("expected worker event-slot recache to preserve bridged onclick handler, got %T", getRecachedNode.Props["onclick"])
		}
	}
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:update-branches",
		Props:            registerParallelRegionProps{Label: "One"},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}
	getHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(getRuntimeSpec.RegionInstanceID, []runtime2.SchedulerShardID{"ui-parallel-region"})
	if parseHostAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseHostAdapterErr)
	}
	if _, parseMountErr := getHostAdapter.HandleHostRegionMount(getRuntimeSpec, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getRenderedNode, parseNodeErr := buildParallelRegionLocalNode(func(parseProps registerParallelRegionProps) Node {
		return Text(parseProps.Label)
	}, getRuntimeSpec.Props)
	if parseNodeErr != nil {
		parseT.Fatalf("buildParallelRegionLocalNode returned error: %v", parseNodeErr)
	}
	storeParallelRegionRenderedNode(getRuntimeSpec.RegionInstanceID, getRenderedNode)
	getInputVersion := buildParallelRegionNextInputVersion(getRuntimeSpec.RegionInstanceID)
	if parseMountErr := handleParallelRegionWorkerMount(getHostAdapter, getRuntimeSpec, getInputVersion); parseMountErr != nil {
		parseT.Fatalf("handleParallelRegionWorkerMount returned error: %v", parseMountErr)
	}
	if parseUpdateErr := handleParallelRegionWorkerUpdate(getHostAdapter, getRuntimeSpec, getInputVersion, runtime2.HostRegionUpdateDispatchTransportResult{}); parseUpdateErr != nil {
		parseT.Fatalf("handleParallelRegionWorkerUpdate(no-schedule) returned error: %v", parseUpdateErr)
	}
	if parseUpdateErr := handleParallelRegionWorkerUpdate(nil, getRuntimeSpec, getInputVersion, runtime2.HostRegionUpdateDispatchTransportResult{}); parseUpdateErr == nil {
		parseT.Fatal("expected nil host adapter update to fail")
	}
	if parseUpdateErr := handleParallelRegionWorkerUpdate(getHostAdapter, getRuntimeSpec, getInputVersion, runtime2.HostRegionUpdateDispatchTransportResult{
		GetDispatchResult: runtime2.HostRegionUpdateDispatchResult{HasScheduled: true},
	}); parseUpdateErr == nil {
		parseT.Fatal("expected missing scheduled snapshot to fail")
	}
	if _, parseDispatchErr := handleParallelRegionUpdateDispatch(nil, getRuntimeSpec, getInputVersion); parseDispatchErr == nil {
		parseT.Fatal("expected nil host adapter dispatch to fail")
	}
}

// TestParallelRegionStatusAndSourceLookupHelpers verifies the remaining public helper adapters preserve cloned source IDs, source lookups, and status projections.
func TestParallelRegionStatusAndSourceLookupHelpers(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: mockdom.NewMockDOMAdapter(), Reset: true})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("status", "ready"); parseErr != nil {
		parseT.Fatalf("SetAtomValue returned error: %v", parseErr)
	}
	getReactiveSource := parallelRegionReactiveSource{getSourceIDs: []string{"status"}}
	getSourceIDs := getReactiveSource.ReactiveRegionSourceIDs()
	if !reflect.DeepEqual(getSourceIDs, []string{"status"}) {
		parseT.Fatalf("ReactiveRegionSourceIDs = %+v, want [status]", getSourceIDs)
	}
	getSourceIDs[0] = "mutated"
	if !reflect.DeepEqual(getReactiveSource.ReactiveRegionSourceIDs(), []string{"status"}) {
		parseT.Fatalf("ReactiveRegionSourceIDs should return a clone, got %+v", getReactiveSource.ReactiveRegionSourceIDs())
	}
	buildParallelRegionNextInputVersion("dashboard.hot-panel:lookup")
	getLookup := buildParallelRegionSourceLookup("dashboard.hot-panel:lookup")
	getSourceValues, getSourceVersions, parseLookupErr := getLookup([]string{"status"})
	if parseLookupErr != nil {
		parseT.Fatalf("source lookup returned error: %v", parseLookupErr)
	}
	if getSourceValues["status"] != "ready" || getSourceVersions["status"] != 1 {
		parseT.Fatalf("source lookup = values:%+v versions:%+v", getSourceValues, getSourceVersions)
	}
	getProjectedStatus := buildParallelRegionStatus(runtime2.HostRegionRuntimeStatus{
		GetRegionInstanceID:            "dashboard.hot-panel:lookup",
		GetRegionMode:                  runtime2.HostRegionRuntimeModeWorkerAttached,
		GetAssignedWorkerShard:         "worker-1",
		GetRendererID:                  "dashboard.hot-panel",
		GetEpoch:                       4,
		GetIsHydrationComplete:         true,
		HasHydratedShellAnchor:         true,
		HasPostHydrationAttached:       true,
		GetLastSnapshotVersion:         5,
		GetLastDispatchedVersion:       5,
		GetLastCommittedVersion:        5,
		GetTransportTier:               runtime2.TransportTierBinary,
		GetDroppedStalePatchCount:      2,
		GetIgnoredStaleDiagnosticCount: 1,
		GetFallbackReason:              "none",
	})
	if getProjectedStatus.GetRegionInstanceID != "dashboard.hot-panel:lookup" || getProjectedStatus.GetTransportTier != string(runtime2.TransportTierBinary) {
		parseT.Fatalf("projected status = %+v", getProjectedStatus)
	}
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(ParallelRegionSpec[registerParallelRegionProps]{
		RendererID:       "dashboard.hot-panel",
		RegionInstanceID: "dashboard.hot-panel:status-helper",
		Props:            registerParallelRegionProps{Label: "Status"},
	})
	if parseRuntimeSpecErr != nil {
		parseT.Fatalf("buildParallelRegionRuntimeSpec returned error: %v", parseRuntimeSpecErr)
	}
	getHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(getRuntimeSpec.RegionInstanceID, []runtime2.SchedulerShardID{"ui-parallel-region"})
	if parseHostAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseHostAdapterErr)
	}
	if _, parseMountErr := getHostAdapter.HandleHostRegionMount(getRuntimeSpec, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	storeParallelRegionAdapterMu.Lock()
	cacheParallelRegionAdapterByID[getRuntimeSpec.RegionInstanceID] = getHostAdapter
	storeParallelRegionAdapterMu.Unlock()
	getStatus, hasStatus, parseStatusErr := GetParallelRegionRuntimeStatus("dashboard.hot-panel:status-helper")
	if parseStatusErr != nil {
		parseT.Fatalf("GetParallelRegionRuntimeStatus returned error: %v", parseStatusErr)
	}
	if !hasStatus || getStatus.GetRegionInstanceID != "dashboard.hot-panel:status-helper" {
		parseT.Fatalf("runtime status = %+v hasStatus=%t", getStatus, hasStatus)
	}
}

// TestBuildUIRuntime2MetaSnapshotNormalizesTrackedRegions verifies tracked parallel-region adapters flow into the runtime2 plugin service.
func TestBuildUIRuntime2MetaSnapshotNormalizesTrackedRegions(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	parseT.Cleanup(runtime2.ResetCapabilityReport)

	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if parseErr := runtime2.SetCapabilityReportOverride(parseCapabilityReport); parseErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride returned error: %v", parseErr)
	}
	getHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("dashboard.hot-panel:meta"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseHostAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseHostAdapterErr)
	}
	getHostAdapter.SetHostRegionRoundTripTimingEnabled(true)
	if _, parseMountErr := getHostAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("dashboard.hot-panel:meta"),
		Props:            map[string]any{"title": "Orders"},
	}, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getSharedSnapshotPage, parseSharedPageErr := runtime2.BuildSharedSnapshotPage(4096)
	if parseSharedPageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseSharedPageErr)
	}
	if _, parseDispatchErr := getHostAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("dashboard.hot-panel:meta"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		parseCapabilityReport,
		getSharedSnapshotPage,
	); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	if parseHydrationErr := getHostAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := getHostAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	if _, parseAttachErr := getHostAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	parseDiagnosticEnvelope, parseDiagnosticErr := runtime2.BuildControlDiagnosticEnvelope("dashboard.hot-panel:meta", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindPatchReady,
		DiagnosticText: "patch ready",
		TransportTier:  runtime2.TransportTierSharedBuffer,
		DiagnosticTiming: &runtime2.DiagnosticTimingMetrics{
			QueueNanos:  7,
			RenderNanos: 9,
		},
		DiagnosticSize: &runtime2.DiagnosticSizeMetrics{
			SnapshotBytes: 256,
			PatchBytes:    64,
		},
		DiagnosticDowngrade: &runtime2.DiagnosticDowngradeReason{
			Path:   runtime2.DiagnosticDowngradePathSharedMemory,
			Reason: string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage),
		},
	})
	if parseDiagnosticErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseDiagnosticErr)
	}
	if _, parseControlErr := runtime2.HandleHostControlEnvelope(getHostAdapter, parseDiagnosticEnvelope); parseControlErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope returned error: %v", parseControlErr)
	}
	storeParallelRegionAdapterMu.Lock()
	cacheParallelRegionAdapterByID[runtime2.RegionInstanceID("dashboard.hot-panel:meta")] = getHostAdapter
	storeParallelRegionAdapterMu.Unlock()

	getSnapshot, parseSnapshotErr := runtime2.BuildRuntime2MetaService().GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{})
	if parseSnapshotErr != nil {
		parseT.Fatalf("GetRuntime2MetaSnapshot returned error: %v", parseSnapshotErr)
	}
	if getSnapshot.Meta.BackendID != string(pluginruntime.BackendIDRuntime2) || getSnapshot.Meta.Truncated {
		parseT.Fatalf("unexpected runtime2 meta snapshot metadata: %+v", getSnapshot.Meta)
	}
	if len(getSnapshot.Regions) != 1 {
		parseT.Fatalf("expected one runtime2 region snapshot, got %+v", getSnapshot.Regions)
	}
	getRegion := getSnapshot.Regions[0]
	if getRegion.RegionInstanceID != "dashboard.hot-panel:meta" || getRegion.RegionMode != string(runtime2.HostRegionRuntimeModeWorkerAttached) {
		parseT.Fatalf("unexpected runtime2 region snapshot: %+v", getRegion)
	}
	if !getRegion.IsHydrationComplete || !getRegion.HasHydratedShellAnchor || !getRegion.HasPostHydrationAttached {
		parseT.Fatalf("expected hydrated worker-attached runtime2 region snapshot, got %+v", getRegion)
	}
	if getRegion.TransportTier != string(runtime2.TransportTierSharedBuffer) || getRegion.DiagnosticCount != 1 {
		parseT.Fatalf("expected shared transport tier and one diagnostic, got %+v", getRegion)
	}
	if len(getSnapshot.Diagnostics) != 1 {
		parseT.Fatalf("expected one runtime2 diagnostic snapshot, got %+v", getSnapshot.Diagnostics)
	}
	getDiagnostic := getSnapshot.Diagnostics[0]
	if getDiagnostic.RegionInstanceID != "dashboard.hot-panel:meta" ||
		getDiagnostic.Type != string(runtime2.DiagnosticEventKindPatchReady) ||
		getDiagnostic.TransportTier != string(runtime2.TransportTierSharedBuffer) ||
		getDiagnostic.DowngradeReason != string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage) {
		parseT.Fatalf("unexpected runtime2 diagnostic snapshot: %+v", getDiagnostic)
	}
	if !getSnapshot.Capabilities["hasWorkerSupport"] || !getSnapshot.Capabilities["hasSharedMemoryTransportSupport"] {
		parseT.Fatalf("expected capability metadata to be preserved, got %+v", getSnapshot.Capabilities)
	}
}

// TestBuildUIRuntime2MetaSnapshotAppliesBudgetAndStableOrdering verifies runtime2 plugin snapshots sort and truncate deterministically.
func TestBuildUIRuntime2MetaSnapshotAppliesBudgetAndStableOrdering(parseT *testing.T) {
	resetParallelRegionRegistry()
	parseT.Cleanup(resetParallelRegionRegistry)
	parseT.Cleanup(runtime2.ResetCapabilityReport)

	if parseErr := runtime2.SetCapabilityReportOverride(runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	}); parseErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride returned error: %v", parseErr)
	}
	for _, parseRegionID := range []string{"dashboard.hot-panel:zeta", "dashboard.hot-panel:alpha"} {
		getHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID(parseRegionID), []runtime2.SchedulerShardID{"shard-a"})
		if parseHostAdapterErr != nil {
			parseT.Fatalf("BuildHostRegionAdapter(%q) returned error: %v", parseRegionID, parseHostAdapterErr)
		}
		if _, parseMountErr := getHostAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID(parseRegionID),
		}, 1); parseMountErr != nil {
			parseT.Fatalf("HandleHostRegionMount(%q) returned error: %v", parseRegionID, parseMountErr)
		}
		parseDiagnosticEnvelope, parseDiagnosticErr := runtime2.BuildControlDiagnosticEnvelope(runtime2.RegionInstanceID(parseRegionID), runtime2.ControlDiagnosticEnvelopeSpec{
			DiagnosticType: runtime2.DiagnosticEventKindUpdate,
			DiagnosticText: "update " + parseRegionID,
			TransportTier:  runtime2.TransportTierStructuredClone,
		})
		if parseDiagnosticErr != nil {
			parseT.Fatalf("BuildControlDiagnosticEnvelope(%q) returned error: %v", parseRegionID, parseDiagnosticErr)
		}
		if _, parseControlErr := runtime2.HandleHostControlEnvelope(getHostAdapter, parseDiagnosticEnvelope); parseControlErr != nil {
			parseT.Fatalf("HandleHostControlEnvelope(%q) returned error: %v", parseRegionID, parseControlErr)
		}
		storeParallelRegionAdapterMu.Lock()
		cacheParallelRegionAdapterByID[runtime2.RegionInstanceID(parseRegionID)] = getHostAdapter
		storeParallelRegionAdapterMu.Unlock()
	}

	getSnapshot, parseSnapshotErr := runtime2.BuildRuntime2MetaService().GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseSnapshotErr != nil {
		parseT.Fatalf("GetRuntime2MetaSnapshot returned error: %v", parseSnapshotErr)
	}
	if !getSnapshot.Meta.Truncated {
		parseT.Fatalf("expected budgeted runtime2 snapshot to be truncated, got %+v", getSnapshot.Meta)
	}
	if len(getSnapshot.Regions) != 1 || getSnapshot.Regions[0].RegionInstanceID != "dashboard.hot-panel:alpha" {
		parseT.Fatalf("expected sorted runtime2 region truncation, got %+v", getSnapshot.Regions)
	}
	if len(getSnapshot.Diagnostics) != 1 || getSnapshot.Diagnostics[0].RegionInstanceID != "dashboard.hot-panel:alpha" {
		parseT.Fatalf("expected sorted runtime2 diagnostic truncation, got %+v", getSnapshot.Diagnostics)
	}
}
