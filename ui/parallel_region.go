package ui

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// ParallelRegionSpec stores the public serializable input contract for one parallel region instance.
type ParallelRegionSpec[Props any] struct {
	RendererID       string
	RegionInstanceID string
	Props            Props
	SourceIDs        []string
}

type parallelRegionRendererEntry struct {
	getRender any
}

type parallelRegionReactiveSource struct {
	getSourceIDs []string
}

var (
	storeParallelRegionRendererMu          sync.RWMutex
	cacheParallelRegionRendererByID        = map[runtime2.RendererID]parallelRegionRendererEntry{}
	storeParallelRegionAdapterMu           sync.RWMutex
	cacheParallelRegionAdapterByID         = map[runtime2.RegionInstanceID]*runtime2.HostRegionAdapter{}
	cacheParallelRegionInputVersionByID    = map[runtime2.RegionInstanceID]uint64{}
	cacheParallelRegionHydrationMarkerByID = map[runtime2.RegionInstanceID]runtime2.SSRShellMarker{}
)

// buildParallelRegionRuntimeSpec converts one public parallel-region spec into the runtime2 contract.
func buildParallelRegionRuntimeSpec[Props any](parseSpec ParallelRegionSpec[Props]) (runtime2.ParallelRegionSpec, error) {
	getRuntimeSpec, parseErr := runtime2.NormalizeParallelRegionSpec(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID(parseSpec.RendererID),
		RegionInstanceID: runtime2.RegionInstanceID(parseSpec.RegionInstanceID),
		Props:            parseSpec.Props,
		SourceIDs:        append([]string(nil), parseSpec.SourceIDs...),
	})
	if parseErr != nil {
		return runtime2.ParallelRegionSpec{}, parseErr
	}
	return getRuntimeSpec, nil
}

// RegisterParallelRegion registers one public parallel-region renderer and bridges it into the runtime2 registry.
func RegisterParallelRegion[Props any](parseRendererID string, parseRender func(Props) Node) error {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return parseRendererIDErr
	}
	if parseRender == nil {
		return fmt.Errorf("ui: parallel-region renderer is required")
	}
	storeParallelRegionRendererMu.Lock()
	defer storeParallelRegionRendererMu.Unlock()
	if _, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]; hasParallelRegionRenderer {
		return fmt.Errorf("ui: parallel-region renderer %q is already registered", getRendererID)
	}
	if parseErr := runtime2.RegisterRenderer(getRendererID, func() {}, runtime2.RendererMetadata{}); parseErr != nil {
		return parseErr
	}
	cacheParallelRegionRendererByID[getRendererID] = parallelRegionRendererEntry{
		getRender: parseRender,
	}
	return nil
}

// ParallelRegion renders one public parallel region through a stable component wrapper.
func ParallelRegion[Props any](parseSpec ParallelRegionSpec[Props]) Node {
	return CreateElement(renderParallelRegionComponent[Props], parseSpec)
}

// renderParallelRegionComponent renders one parallel-region shell with immediate local content and runtime2 shell markers.
func renderParallelRegionComponent[Props any](parseSpec ParallelRegionSpec[Props]) Node {
	getRuntimeSpec, parseRuntimeSpecErr := buildParallelRegionRuntimeSpec(parseSpec)
	if parseRuntimeSpecErr != nil {
		panic(fmt.Sprintf("ui: parallel-region spec is invalid: %v", parseRuntimeSpecErr))
	}
	getRender, parseResolveErr := resolveParallelRegionRenderer(string(getRuntimeSpec.RendererID))
	if parseResolveErr != nil {
		panic(fmt.Sprintf("ui: parallel-region renderer resolution failed: %v", parseResolveErr))
	}
	UseEffect(func() func() {
		getRegionInstanceID := string(getRuntimeSpec.RegionInstanceID)
		return func() {
			_ = handleParallelRegionOwnerRemove(getRegionInstanceID)
		}
	}, string(getRuntimeSpec.RegionInstanceID))
	return buildParallelRegionReactiveNode(getRuntimeSpec, getRender)
}

// buildParallelRegionReactiveNode wraps one public parallel-region shell in fine-grained source subscriptions when declared sources exist.
func buildParallelRegionReactiveNode(parseRuntimeSpec runtime2.ParallelRegionSpec, parseRender any) Node {
	if len(parseRuntimeSpec.SourceIDs) == 0 {
		return buildParallelRegionRenderedNode(parseRuntimeSpec, parseRender)
	}
	return ReactiveRegion(func() Node {
		return buildParallelRegionRenderedNode(parseRuntimeSpec, parseRender)
	}, buildParallelRegionReactiveSources(parseRuntimeSpec.SourceIDs)...)
}

// buildParallelRegionRenderedNode renders one public parallel-region shell and forwards browser lifecycle updates into runtime2.
func buildParallelRegionRenderedNode(parseRuntimeSpec runtime2.ParallelRegionSpec, parseRender any) Node {
	if canParallelRegionUseRuntime2Lifecycle() {
		getParallelRegionHostAdapter, hasParallelRegionMounted, parseHostAdapterErr := buildParallelRegionHostAdapter(parseRuntimeSpec)
		if parseHostAdapterErr != nil {
			panic(fmt.Sprintf("ui: parallel-region host adapter setup failed: %v", parseHostAdapterErr))
		}
		if parseSourceLookupErr := getParallelRegionHostAdapter.SetHostRegionSourceLookup(
			buildParallelRegionSourceLookup(parseRuntimeSpec.RegionInstanceID),
		); parseSourceLookupErr != nil {
			panic(fmt.Sprintf("ui: parallel-region source lookup setup failed: %v", parseSourceLookupErr))
		}
		getInputVersion := buildParallelRegionNextInputVersion(parseRuntimeSpec.RegionInstanceID)
		if !hasParallelRegionMounted {
			if parseDispatchErr := handleParallelRegionUpdateDispatch(getParallelRegionHostAdapter, parseRuntimeSpec, getInputVersion); parseDispatchErr != nil {
				panic(fmt.Sprintf("ui: parallel-region update dispatch failed: %v", parseDispatchErr))
			}
		}
	}
	getChild, parseRenderErr := buildParallelRegionLocalNode(parseRender, parseRuntimeSpec.Props)
	if parseRenderErr != nil {
		panic(fmt.Sprintf("ui: parallel-region local render failed: %v", parseRenderErr))
	}
	getShellMarker, parseMarkerErr := runtime2.BuildSSRShellMarkerAttributeValue(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: parseRuntimeSpec.RegionInstanceID,
		RendererID:       parseRuntimeSpec.RendererID,
	})
	if parseMarkerErr != nil {
		panic(fmt.Sprintf("ui: parallel-region shell marker build failed: %v", parseMarkerErr))
	}
	storeParallelRegionHydrationMarker(parseRuntimeSpec.RegionInstanceID, runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: parseRuntimeSpec.RegionInstanceID,
		RendererID:       parseRuntimeSpec.RendererID,
	})
	getShellProps := map[string]interface{}{
		runtime2.SSRShellMarkerAttribute: getShellMarker,
	}
	if getChild == nil {
		return runtime.CreateElement("div", getShellProps)
	}
	return runtime.CreateElement("div", getShellProps, getChild)
}

// buildParallelRegionReactiveSources converts canonical source IDs into public reactive-region sources.
func buildParallelRegionReactiveSources(parseSourceIDs []string) []ReactiveSource {
	getReactiveSources := make([]ReactiveSource, 0, len(parseSourceIDs))
	for _, getSourceID := range parseSourceIDs {
		getReactiveSources = append(getReactiveSources, parallelRegionReactiveSource{
			getSourceIDs: []string{getSourceID},
		})
	}
	return getReactiveSources
}

// BuildParallelRegionSourceIDs flattens declared reactive sources into a stable public source-ID list.
func BuildParallelRegionSourceIDs(parseSources ...ReactiveSource) ([]string, error) {
	getSourceIDs := make([]string, 0, len(parseSources))
	parseSeenSourceIDs := make(map[string]bool, len(parseSources))
	for _, parseSource := range parseSources {
		if parseSource == nil {
			continue
		}
		for _, parseSourceID := range parseSource.ReactiveRegionSourceIDs() {
			getNormalizedSourceIDs, parseNormalizeErr := runtime2.NormalizeSourceIDs([]string{parseSourceID})
			if parseNormalizeErr != nil {
				return nil, parseNormalizeErr
			}
			if len(getNormalizedSourceIDs) == 0 {
				continue
			}
			getSourceID := getNormalizedSourceIDs[0]
			if parseSeenSourceIDs[getSourceID] {
				continue
			}
			parseSeenSourceIDs[getSourceID] = true
			getSourceIDs = append(getSourceIDs, getSourceID)
		}
	}
	return getSourceIDs, nil
}

// ReactiveRegionSourceIDs reports the stable source IDs owned by one internal parallel-region reactive source binding.
func (parseSource parallelRegionReactiveSource) ReactiveRegionSourceIDs() []string {
	return append([]string(nil), parseSource.getSourceIDs...)
}

// resolveParallelRegionRenderer resolves one registered public parallel-region renderer by stable ID.
func resolveParallelRegionRenderer(parseRendererID string) (any, error) {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return nil, parseRendererIDErr
	}
	storeParallelRegionRendererMu.RLock()
	defer storeParallelRegionRendererMu.RUnlock()
	getParallelRegionRendererEntry, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]
	if !hasParallelRegionRenderer {
		return nil, fmt.Errorf("ui: parallel-region renderer %q is not registered", getRendererID)
	}
	return getParallelRegionRendererEntry.getRender, nil
}

// buildParallelRegionHostAdapter creates and mounts one runtime2 host adapter for a public parallel region when browser lifecycle support is active.
func buildParallelRegionHostAdapter(parseRuntimeSpec runtime2.ParallelRegionSpec) (*runtime2.HostRegionAdapter, bool, error) {
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	if getParallelRegionHostAdapter := cacheParallelRegionAdapterByID[parseRuntimeSpec.RegionInstanceID]; getParallelRegionHostAdapter != nil {
		getHasRemounted, parseRemountErr := handleParallelRegionStructuralRemount(getParallelRegionHostAdapter, parseRuntimeSpec)
		if parseRemountErr != nil {
			return nil, false, parseRemountErr
		}
		return getParallelRegionHostAdapter, getHasRemounted, nil
	}
	getParallelRegionHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(
		parseRuntimeSpec.RegionInstanceID,
		[]runtime2.SchedulerShardID{"ui-parallel-region"},
	)
	if parseHostAdapterErr != nil {
		return nil, false, parseHostAdapterErr
	}
	if _, parseMountErr := getParallelRegionHostAdapter.HandleHostRegionMount(parseRuntimeSpec, 1); parseMountErr != nil {
		return nil, false, parseMountErr
	}
	cacheParallelRegionAdapterByID[parseRuntimeSpec.RegionInstanceID] = getParallelRegionHostAdapter
	return getParallelRegionHostAdapter, true, nil
}

// handleParallelRegionStructuralRemount compares cached public region state and remounts runtime2 ownership when renderer or shell ownership changed.
func handleParallelRegionStructuralRemount(parseHostRegionAdapter *runtime2.HostRegionAdapter, parseRuntimeSpec runtime2.ParallelRegionSpec) (bool, error) {
	if parseHostRegionAdapter == nil {
		return false, fmt.Errorf("ui: parallel-region host adapter is required")
	}
	getCoordinator := parseHostRegionAdapter.GetHostRegionCoordinator()
	if getCoordinator == nil {
		return false, fmt.Errorf("ui: parallel-region coordinator is required")
	}
	getCoordinatorEntry, hasCoordinatorEntry := getCoordinator.GetEntry(parseRuntimeSpec.RegionInstanceID)
	if !hasCoordinatorEntry {
		return false, fmt.Errorf("ui: parallel-region %q is not mounted", parseRuntimeSpec.RegionInstanceID)
	}
	getIsLocalShellOwned := true
	if getCoordinatorEntry.RendererID == parseRuntimeSpec.RendererID &&
		parseHostRegionAdapter.IsHostRegionLocalShellOwnership() == getIsLocalShellOwned {
		return false, nil
	}
	_, parseRemountErr := parseHostRegionAdapter.HandleHostRegionStructuralRemount(parseRuntimeSpec, getIsLocalShellOwned)
	if parseRemountErr != nil {
		return false, parseRemountErr
	}
	return true, nil
}

// buildParallelRegionNextInputVersion increments and returns the next browser-side input version for one public parallel region instance.
func buildParallelRegionNextInputVersion(parseRegionInstanceID runtime2.RegionInstanceID) uint64 {
	if parseRegionInstanceID == "" {
		return 0
	}
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	getInputVersion := cacheParallelRegionInputVersionByID[parseRegionInstanceID] + 1
	cacheParallelRegionInputVersionByID[parseRegionInstanceID] = getInputVersion
	return getInputVersion
}

// buildParallelRegionSourceLookup builds one runtime2 declared-source lookup bridge backed by the public runtime atom registry.
func buildParallelRegionSourceLookup(parseRegionInstanceID runtime2.RegionInstanceID) runtime2.HostRegionSourceLookup {
	return func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
		return buildParallelRegionSourceSnapshot(parseRegionInstanceID, parseSourceIDs)
	}
}

// buildParallelRegionSourceSnapshot reads declared source values from the shared runtime atom registry and stamps them with the current public input version.
func buildParallelRegionSourceSnapshot(parseRegionInstanceID runtime2.RegionInstanceID, parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
	getSourceIDs, parseSourceIDsErr := runtime2.NormalizeSourceIDs(parseSourceIDs)
	if parseSourceIDsErr != nil {
		return nil, nil, parseSourceIDsErr
	}
	if len(getSourceIDs) == 0 {
		return map[string]any{}, map[string]uint64{}, nil
	}
	getInputVersion := resolveParallelRegionInputVersion(string(parseRegionInstanceID))
	if getInputVersion == 0 {
		return nil, nil, fmt.Errorf("ui: parallel-region input version is not tracked for %q", parseRegionInstanceID)
	}
	getRuntime := runtime.GetGlobalRuntime()
	getSourceValues := make(map[string]any, len(getSourceIDs))
	getSourceVersions := make(map[string]uint64, len(getSourceIDs))
	for _, getSourceID := range getSourceIDs {
		getSourceValue, hasSourceValue := getRuntime.GetAtomValue(getSourceID)
		if !hasSourceValue {
			return nil, nil, fmt.Errorf("ui: parallel-region declared source %q is not available", getSourceID)
		}
		getSourceValues[getSourceID] = getSourceValue
		getSourceVersions[getSourceID] = getInputVersion
	}
	return getSourceValues, getSourceVersions, nil
}

// handleParallelRegionUpdateDispatch forwards one public rerender into runtime2 host update dispatch using the currently available snapshot transport tier.
func handleParallelRegionUpdateDispatch(parseHostRegionAdapter *runtime2.HostRegionAdapter, parseRuntimeSpec runtime2.ParallelRegionSpec, parseInputVersion uint64) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("ui: parallel-region host adapter is required")
	}
	getDispatchPriority := runtime2.HostRegionDispatchPriorityUrgent
	if runtime.IsCurrentFiberTransitionUpdate() {
		getDispatchPriority = runtime2.HostRegionDispatchPriorityDeferred
	}
	_, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransportPriority(
		parseRuntimeSpec,
		parseInputVersion,
		getDispatchPriority,
		runtime2.GetCapabilityReport(),
		nil,
	)
	return parseDispatchErr
}

// setParallelRegionHydrationObserver wires one public parallel-region shell bridge into the next hydration observer callback.
func setParallelRegionHydrationObserver(
	parseRt *runtime.Runtime,
	parseCorrelationID string,
	parseBridge func() error,
	parseNotify func(runtime.HydrationMetrics),
) {
	if parseRt == nil {
		return
	}
	parseRt.SetNextHydrationObserver(parseCorrelationID, func(parseMetrics runtime.HydrationMetrics) {
		if !parseMetrics.Failed && parseBridge != nil {
			if parseBridgeErr := parseBridge(); parseBridgeErr != nil {
				runtime.ReportDiagnostic("ui", runtime.DiagnosticError, fmt.Sprintf("parallel-region hydration bridge failed: %v", parseBridgeErr))
			}
		}
		if parseNotify != nil {
			parseNotify(parseMetrics)
		}
	})
}

// storeParallelRegionHydrationMarker caches one runtime2 shell marker for one hydrated-shell mapping.
func storeParallelRegionHydrationMarker(parseRegionInstanceID runtime2.RegionInstanceID, parseMarker runtime2.SSRShellMarker) {
	if parseRegionInstanceID == "" {
		return
	}
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	cacheParallelRegionHydrationMarkerByID[parseRegionInstanceID] = parseMarker
}

// discardParallelRegionHydrationMarker clears one cached runtime2 shell marker.
func discardParallelRegionHydrationMarker(parseRegionInstanceID runtime2.RegionInstanceID) {
	if parseRegionInstanceID == "" {
		return
	}
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	delete(cacheParallelRegionHydrationMarkerByID, parseRegionInstanceID)
}

// snapshotParallelRegionHydrationMarkers returns a copy of the cached runtime2 shell markers.
func snapshotParallelRegionHydrationMarkers() map[runtime2.RegionInstanceID]runtime2.SSRShellMarker {
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	if len(cacheParallelRegionHydrationMarkerByID) == 0 {
		return nil
	}
	getShellMarkers := make(map[runtime2.RegionInstanceID]runtime2.SSRShellMarker, len(cacheParallelRegionHydrationMarkerByID))
	for getRegionInstanceID, getShellMarker := range cacheParallelRegionHydrationMarkerByID {
		getShellMarkers[getRegionInstanceID] = getShellMarker
	}
	return getShellMarkers
}

// buildParallelRegionLocalNode invokes one registered public renderer with validated props and returns its local-first node.
func buildParallelRegionLocalNode(parseRender any, parseProps any) (Node, error) {
	getRenderValue := reflect.ValueOf(parseRender)
	if !getRenderValue.IsValid() || getRenderValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("ui: parallel-region renderer is not callable")
	}
	if getRenderValue.Type().NumIn() != 1 || getRenderValue.Type().NumOut() != 1 {
		return nil, fmt.Errorf("ui: parallel-region renderer must accept one props argument and return one ui.Node")
	}
	getArgValue, parseArgErr := buildParallelRegionRenderArg(getRenderValue.Type().In(0), parseProps)
	if parseArgErr != nil {
		return nil, parseArgErr
	}
	getResults := getRenderValue.Call([]reflect.Value{getArgValue})
	if len(getResults) != 1 {
		return nil, fmt.Errorf("ui: parallel-region renderer returned %d values", len(getResults))
	}
	if getResults[0].IsNil() {
		return nil, nil
	}
	getNode, hasNode := getResults[0].Interface().(Node)
	if !hasNode {
		return nil, fmt.Errorf("ui: parallel-region renderer returned %T, want ui.Node", getResults[0].Interface())
	}
	return getNode, nil
}

// buildParallelRegionRenderArg maps validated public props into the registered renderer input type.
func buildParallelRegionRenderArg(parseArgType reflect.Type, parseProps any) (reflect.Value, error) {
	if parseProps == nil {
		return reflect.Zero(parseArgType), nil
	}
	getPropsValue := reflect.ValueOf(parseProps)
	if getPropsValue.Type().AssignableTo(parseArgType) {
		return getPropsValue, nil
	}
	if getPropsValue.Type().ConvertibleTo(parseArgType) {
		return getPropsValue.Convert(parseArgType), nil
	}
	return reflect.Value{}, fmt.Errorf(
		"ui: parallel-region props type %s does not match registered renderer input %s",
		getPropsValue.Type(),
		parseArgType,
	)
}

// handleParallelRegionOwnerRemove routes one public parallel-region owner removal into runtime2 cleanup and clears cached adapter state.
func handleParallelRegionOwnerRemove(parseRegionInstanceID string) error {
	getRegionInstanceID, parseRegionInstanceIDErr := runtime2.ParseRegionInstanceID(parseRegionInstanceID)
	if parseRegionInstanceIDErr != nil {
		return parseRegionInstanceIDErr
	}
	storeParallelRegionAdapterMu.Lock()
	getParallelRegionHostAdapter := cacheParallelRegionAdapterByID[getRegionInstanceID]
	delete(cacheParallelRegionAdapterByID, getRegionInstanceID)
	delete(cacheParallelRegionInputVersionByID, getRegionInstanceID)
	discardParallelRegionHydrationMarker(getRegionInstanceID)
	storeParallelRegionAdapterMu.Unlock()
	if getParallelRegionHostAdapter == nil {
		return nil
	}
	_, parseOwnerRemoveErr := getParallelRegionHostAdapter.HandleHostRegionOwnerRemove()
	return parseOwnerRemoveErr
}

// handleParallelRegionHydrationSelector discovers hydrated shell markers under one selector target and reattaches cached runtime2 regions.
func handleParallelRegionHydrationSelector(parseSelector string) error {
	if !canParallelRegionUseRuntime2Lifecycle() {
		return nil
	}
	return handleParallelRegionHydrationNodes(
		runtime.GetGlobalRuntime().FindNodesWithAttributeInSelector(parseSelector, runtime2.SSRShellMarkerAttribute),
	)
}

// handleParallelRegionHydrationTarget discovers hydrated shell markers under one explicit target node and reattaches cached runtime2 regions.
func handleParallelRegionHydrationTarget(parseTarget interface{}) error {
	if !canParallelRegionUseRuntime2Lifecycle() {
		return nil
	}
	return handleParallelRegionHydrationNodes(
		runtime.GetGlobalRuntime().FindNodesWithAttributeInTarget(parseTarget, runtime2.SSRShellMarkerAttribute),
	)
}

// handleParallelRegionHydrationNodes parses hydrated shell markers and completes public runtime2 shell attach for cached regions.
func handleParallelRegionHydrationNodes(parseNodes []runtime.DOMNode) error {
	if len(parseNodes) == 0 {
		return nil
	}
	getRuntime := runtime.GetGlobalRuntime()
	parseSeenRegionInstanceIDs := make(map[string]bool, len(parseNodes))
	for _, parseNode := range parseNodes {
		getMarkerPayload, hasMarkerPayload := getRuntime.GetAttributeValue(parseNode, runtime2.SSRShellMarkerAttribute)
		if !hasMarkerPayload || getMarkerPayload == "" {
			continue
		}
		getShellMarker, parseMarkerErr := runtime2.ParseSSRShellMarkerAttributeValue(getMarkerPayload)
		if parseMarkerErr != nil {
			return parseMarkerErr
		}
		getRegionInstanceID := string(getShellMarker.RegionInstanceID)
		if parseSeenRegionInstanceIDs[getRegionInstanceID] {
			continue
		}
		parseSeenRegionInstanceIDs[getRegionInstanceID] = true
		getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter(getRegionInstanceID)
		if !hasParallelRegionHostAdapter {
			continue
		}
		getMismatchResult, parseMismatchErr := getParallelRegionHostAdapter.HandleHostRegionShellIdentityMismatchDetection(getShellMarker)
		if parseMismatchErr != nil {
			return parseMismatchErr
		}
		if getMismatchResult.HasMismatch {
			return fmt.Errorf(
				"ui: hydrated parallel-region shell marker mismatch for %q (region mismatch=%t renderer mismatch=%t)",
				getRegionInstanceID,
				getMismatchResult.HasRegionIDMismatch,
				getMismatchResult.HasRendererIDMismatch,
			)
		}
		getAnchorNodeID, getAnchorTag, parseAnchorBuildErr := buildParallelRegionHydratedShellAnchor(getRuntime, parseNode)
		if parseAnchorBuildErr != nil {
			return parseAnchorBuildErr
		}
		if !getParallelRegionHostAdapter.GetHostRegionIsHydrationComplete() {
			if parseHydrationErr := getParallelRegionHostAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
				return parseHydrationErr
			}
		}
		if !getParallelRegionHostAdapter.HasHostRegionHydratedShellAnchor() {
			if parseAnchorErr := getParallelRegionHostAdapter.HandleHostRegionRegisterHydratedShellAnchor(getAnchorNodeID, getAnchorTag); parseAnchorErr != nil {
				return parseAnchorErr
			}
		}
		if !getParallelRegionHostAdapter.HasHostRegionPostHydrationAttached() {
			if _, parseAttachErr := getParallelRegionHostAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
				return parseAttachErr
			}
		}
	}
	return nil
}

// buildParallelRegionHydratedShellAnchor validates one resumed public shell node and maps it onto the canonical runtime2 root anchor identity.
func buildParallelRegionHydratedShellAnchor(parseRt *runtime.Runtime, parseNode runtime.DOMNode) (uint64, string, error) {
	if parseRt == nil {
		return 0, "", fmt.Errorf("ui: parallel-region runtime is required for hydration anchor registration")
	}
	if parseNode == nil || parseNode.IsNull() {
		return 0, "", fmt.Errorf("ui: hydrated parallel-region shell node is required")
	}
	getTagName, hasTagName := parseRt.GetTagName(parseNode)
	if !hasTagName {
		return 0, "", fmt.Errorf("ui: hydrated parallel-region shell tag is unavailable")
	}
	if getTagName == "#text" {
		return 0, "", fmt.Errorf("ui: hydrated parallel-region shell must be a host element, got %q", getTagName)
	}
	if getTagName != "div" {
		return 0, "", fmt.Errorf("ui: hydrated parallel-region shell tag %q does not match expected public shell tag %q", getTagName, "div")
	}
	return 1, getTagName, nil
}

// resolveParallelRegionInputVersion reports the current browser-side input version tracked for one public parallel region instance.
func resolveParallelRegionInputVersion(parseRegionInstanceID string) uint64 {
	getRegionInstanceID, parseRegionInstanceIDErr := runtime2.ParseRegionInstanceID(parseRegionInstanceID)
	if parseRegionInstanceIDErr != nil {
		return 0
	}
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	return cacheParallelRegionInputVersionByID[getRegionInstanceID]
}

// resolveParallelRegionHostAdapter reports one cached runtime2 host adapter for deterministic tests.
func resolveParallelRegionHostAdapter(parseRegionInstanceID string) (*runtime2.HostRegionAdapter, bool) {
	getRegionInstanceID, parseRegionInstanceIDErr := runtime2.ParseRegionInstanceID(parseRegionInstanceID)
	if parseRegionInstanceIDErr != nil {
		return nil, false
	}
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := cacheParallelRegionAdapterByID[getRegionInstanceID]
	return getParallelRegionHostAdapter, hasParallelRegionHostAdapter
}

// resetParallelRegionRegistry clears the public parallel-region registry and the bridged runtime2 registry for deterministic tests.
func resetParallelRegionRegistry() {
	storeParallelRegionRendererMu.Lock()
	cacheParallelRegionRendererByID = map[runtime2.RendererID]parallelRegionRendererEntry{}
	storeParallelRegionRendererMu.Unlock()
	storeParallelRegionAdapterMu.Lock()
	cacheParallelRegionAdapterByID = map[runtime2.RegionInstanceID]*runtime2.HostRegionAdapter{}
	cacheParallelRegionInputVersionByID = map[runtime2.RegionInstanceID]uint64{}
	cacheParallelRegionHydrationMarkerByID = map[runtime2.RegionInstanceID]runtime2.SSRShellMarker{}
	storeParallelRegionAdapterMu.Unlock()
	runtime2.ResetRendererRegistry()
}
