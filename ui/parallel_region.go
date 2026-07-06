package ui

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// ParallelRegionSpec stores the public serializable input contract for one parallel region instance.
type ParallelRegionSpec[Props any] struct {
	RendererID        string
	RegionInstanceID  string
	Props             Props
	SourceIDs         []string
	SchedulerShardIDs []string
}

// ParallelRegionStatus reports one read-only public runtime snapshot for a tracked parallel region instance.
type ParallelRegionStatus struct {
	GetRegionInstanceID            string
	GetRegionMode                  string
	GetAssignedWorkerShard         string
	GetRendererID                  string
	GetEpoch                       uint64
	GetIsHydrationComplete         bool
	HasHydratedShellAnchor         bool
	HasPostHydrationAttached       bool
	GetLastSnapshotVersion         uint64
	GetLastDispatchedVersion       uint64
	GetLastCommittedVersion        uint64
	GetTransportTier               string
	GetDroppedStalePatchCount      uint64
	GetIgnoredStaleDiagnosticCount uint64
	GetFallbackReason              string
}

// ParallelRegionWorkerRendererConfig stores optional trust and validation settings for one public worker-native renderer.
type ParallelRegionWorkerRendererConfig struct {
	IsTrusted                        bool
	HasUpdateValidationOverride      bool
	ShouldValidateUpdateRenderOutput bool
}

type parallelRegionRendererEntry struct {
	getRender       any
	getWorkerRender runtime2.WorkerRegionRenderer
}

type parallelRegionReactiveSource struct {
	getSourceIDs []string
}

var (
	storeParallelRegionRendererMu          sync.RWMutex
	cacheParallelRegionRendererByID        = map[runtime2.RendererID]parallelRegionRendererEntry{}
	storeParallelRegionAdapterMu           sync.RWMutex
	cacheParallelRegionAdapterByID         = map[runtime2.RegionInstanceID]*runtime2.HostRegionAdapter{}
	cacheParallelRegionRenderedNodeByID    = map[runtime2.RegionInstanceID]Node{}
	cacheParallelRegionInputVersionByID    = map[runtime2.RegionInstanceID]uint64{}
	cacheParallelRegionSchedulerShardsByID = map[runtime2.RegionInstanceID][]runtime2.SchedulerShardID{}
	cacheParallelRegionHydrationMarkerByID = map[runtime2.RegionInstanceID]runtime2.SSRShellMarker{}
	cacheParallelRegionWorkerRuntime       = runtime2.BuildWorkerRegionRuntime()
	// cacheParallelRegionBridgeFallbackByID records a UI-side worker-bridge failure
	// (mount/update/click) per region so it is SURFACED on ParallelRegionStatus
	// .GetFallbackReason instead of only logged. Without it a bridge failure left the
	// region WorkerAttached with no worker state — every later click hit "unknown
	// region ID" and the content silently never updated, with no visible error.
	cacheParallelRegionBridgeFallbackByID = map[runtime2.RegionInstanceID]string{}
)

// recordParallelRegionBridgeFallback stores a worker-bridge failure reason for one
// region so it surfaces on the public status. Ignores an empty region/reason.
func recordParallelRegionBridgeFallback(parseRegionInstanceID runtime2.RegionInstanceID, parseReason string) {
	if parseRegionInstanceID == "" || parseReason == "" {
		return
	}
	storeParallelRegionAdapterMu.Lock()
	cacheParallelRegionBridgeFallbackByID[parseRegionInstanceID] = parseReason
	storeParallelRegionAdapterMu.Unlock()
}

// clearParallelRegionBridgeFallback drops a region's recorded bridge failure after
// it recovers (a successful mount/update), so a stale reason does not linger.
func clearParallelRegionBridgeFallback(parseRegionInstanceID runtime2.RegionInstanceID) {
	if parseRegionInstanceID == "" {
		return
	}
	storeParallelRegionAdapterMu.Lock()
	delete(cacheParallelRegionBridgeFallbackByID, parseRegionInstanceID)
	storeParallelRegionAdapterMu.Unlock()
}

// resolveParallelRegionBridgeFallback returns a region's recorded bridge failure reason.
func resolveParallelRegionBridgeFallback(parseRegionInstanceID runtime2.RegionInstanceID) string {
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	return cacheParallelRegionBridgeFallbackByID[parseRegionInstanceID]
}

// buildParallelRegionRendererMetadata returns the current runtime2 capability metadata for public display-only regions.
func buildParallelRegionRendererMetadata(parseEventSlotMetadata runtime2.EventSlotMetadata) runtime2.RendererMetadata {
	return runtime2.RendererMetadata{
		FeatureFlags:      []string{"display-only"},
		EventSlotMetadata: parseEventSlotMetadata,
	}
}

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
	if parseErr := runtime2.RegisterRenderer(
		getRendererID,
		func() {},
		buildParallelRegionRendererMetadata(runtime2.EventSlotMetadata{}),
	); parseErr != nil {
		return parseErr
	}
	if parseErr := cacheParallelRegionWorkerRuntime.RegisterWorkerRegionRendererWithRegisteredMetadata(
		string(getRendererID),
		func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
			return buildParallelRegionWorkerRenderOutput(parseMount)
		},
	); parseErr != nil {
		return parseErr
	}
	cacheParallelRegionRendererByID[getRendererID] = parallelRegionRendererEntry{
		getRender: parseRender,
	}
	return nil
}

// SetParallelRegionWorkerRenderer registers one explicit worker-native renderer for an already-registered public parallel region.
func SetParallelRegionWorkerRenderer(parseRendererID string, parseRender runtime2.WorkerRegionRenderer) error {
	return SetParallelRegionWorkerRendererWithConfig(parseRendererID, parseRender, ParallelRegionWorkerRendererConfig{})
}

// SetParallelRegionWorkerRendererWithConfig registers one explicit worker-native renderer with optional trust and validation settings.
func SetParallelRegionWorkerRendererWithConfig(
	parseRendererID string,
	parseRender runtime2.WorkerRegionRenderer,
	parseConfig ParallelRegionWorkerRendererConfig,
) error {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return parseRendererIDErr
	}
	if parseRender == nil {
		return fmt.Errorf("ui: parallel-region worker renderer is required")
	}
	storeParallelRegionRendererMu.Lock()
	defer storeParallelRegionRendererMu.Unlock()
	getParallelRegionRendererEntry, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]
	if !hasParallelRegionRenderer {
		return fmt.Errorf("ui: parallel-region renderer %q is not registered", getRendererID)
	}
	if getParallelRegionRendererEntry.getWorkerRender != nil {
		return fmt.Errorf("ui: parallel-region worker renderer %q is already registered", getRendererID)
	}
	getParallelRegionRendererEntry.getWorkerRender = parseRender
	cacheParallelRegionRendererByID[getRendererID] = getParallelRegionRendererEntry
	if parseTrustErr := cacheParallelRegionWorkerRuntime.SetWorkerRegionRendererTrusted(string(getRendererID), parseConfig.IsTrusted); parseTrustErr != nil {
		return parseTrustErr
	}
	if parseConfig.HasUpdateValidationOverride {
		if parseValidationErr := cacheParallelRegionWorkerRuntime.SetWorkerRegionRendererUpdateValidationEnabled(
			string(getRendererID),
			parseConfig.ShouldValidateUpdateRenderOutput,
		); parseValidationErr != nil {
			return parseValidationErr
		}
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
	getSchedulerShardIDs, parseSchedulerShardIDsErr := buildParallelRegionSchedulerShardIDs(parseSpec.SchedulerShardIDs)
	if parseSchedulerShardIDsErr != nil {
		panic(fmt.Sprintf("ui: parallel-region scheduler shard setup failed: %v", parseSchedulerShardIDsErr))
	}
	return buildParallelRegionReactiveNode(getRuntimeSpec, getRender, getSchedulerShardIDs)
}

// buildParallelRegionReactiveNode wraps one public parallel-region shell in fine-grained source subscriptions when declared sources exist.
func buildParallelRegionReactiveNode(parseRuntimeSpec runtime2.ParallelRegionSpec, parseRender any, parseSchedulerShardIDs []runtime2.SchedulerShardID) Node {
	if len(parseRuntimeSpec.SourceIDs) == 0 {
		return buildParallelRegionRenderedNode(parseRuntimeSpec, parseRender, parseSchedulerShardIDs)
	}
	return ReactiveRegion(func() Node {
		return buildParallelRegionRenderedNode(parseRuntimeSpec, parseRender, parseSchedulerShardIDs)
	}, buildParallelRegionReactiveSources(parseRuntimeSpec.SourceIDs)...)
}

// buildParallelRegionRenderedNode renders one public parallel-region shell and forwards browser lifecycle updates into runtime2.
func buildParallelRegionRenderedNode(parseRuntimeSpec runtime2.ParallelRegionSpec, parseRender any, parseSchedulerShardIDs []runtime2.SchedulerShardID) Node {
	getChild, parseRenderErr := buildParallelRegionLocalNode(parseRender, parseRuntimeSpec.Props)
	if parseRenderErr != nil {
		panic(fmt.Sprintf("ui: parallel-region local render failed: %v", parseRenderErr))
	}
	getChild, getEventSlotMetadata, parseBridgeErr := buildParallelRegionBridgedNode(parseRuntimeSpec, getChild)
	if parseBridgeErr != nil {
		panic(fmt.Sprintf("ui: parallel-region bridge setup failed: %v", parseBridgeErr))
	}
	if canParallelRegionUseRuntime2Lifecycle() {
		if parseMetadataErr := handleParallelRegionRendererMetadata(parseRuntimeSpec.RendererID, getEventSlotMetadata); parseMetadataErr != nil {
			panic(fmt.Sprintf("ui: parallel-region renderer metadata sync failed: %v", parseMetadataErr))
		}
		getParallelRegionHostAdapter, hasParallelRegionMounted, parseHostAdapterErr := buildParallelRegionHostAdapter(parseRuntimeSpec, parseSchedulerShardIDs)
		if parseHostAdapterErr != nil {
			panic(fmt.Sprintf("ui: parallel-region host adapter setup failed: %v", parseHostAdapterErr))
		}
		if parseSourceLookupErr := getParallelRegionHostAdapter.SetHostRegionSourceLookup(
			buildParallelRegionSourceLookup(parseRuntimeSpec.RegionInstanceID),
		); parseSourceLookupErr != nil {
			panic(fmt.Sprintf("ui: parallel-region source lookup setup failed: %v", parseSourceLookupErr))
		}
		storeParallelRegionRenderedNode(parseRuntimeSpec.RegionInstanceID, getChild)
		getInputVersion := buildParallelRegionNextInputVersion(parseRuntimeSpec.RegionInstanceID)
		if hasParallelRegionMounted {
			if parseWorkerMountErr := handleParallelRegionWorkerMount(getParallelRegionHostAdapter, parseRuntimeSpec, getInputVersion); parseWorkerMountErr != nil {
				reportParallelRegionDiagnosticError("parallel-region worker mount bridge failed: " + parseWorkerMountErr.Error())
				// Surface on the public status: a failed mount leaves the region
				// without worker state, so later clicks silently fail otherwise.
				recordParallelRegionBridgeFallback(parseRuntimeSpec.RegionInstanceID, "worker mount bridge failed: "+parseWorkerMountErr.Error())
			} else {
				clearParallelRegionBridgeFallback(parseRuntimeSpec.RegionInstanceID)
			}
		} else {
			getDispatchResult, parseDispatchErr := handleParallelRegionUpdateDispatch(getParallelRegionHostAdapter, parseRuntimeSpec, getInputVersion)
			if parseDispatchErr != nil {
				panic(fmt.Sprintf("ui: parallel-region update dispatch failed: %v", parseDispatchErr))
			}
			if parseWorkerUpdateErr := handleParallelRegionWorkerUpdate(
				getParallelRegionHostAdapter,
				parseRuntimeSpec,
				getInputVersion,
				getDispatchResult,
			); parseWorkerUpdateErr != nil {
				reportParallelRegionDiagnosticError("parallel-region worker update bridge failed: " + parseWorkerUpdateErr.Error())
				recordParallelRegionBridgeFallback(parseRuntimeSpec.RegionInstanceID, "worker update bridge failed: "+parseWorkerUpdateErr.Error())
			} else {
				clearParallelRegionBridgeFallback(parseRuntimeSpec.RegionInstanceID)
			}
		}
		if parseAttachErr := handleParallelRegionPostRenderAttachByID(parseRuntimeSpec.RegionInstanceID); parseAttachErr != nil {
			reportParallelRegionDiagnosticError("parallel-region post-render attach failed: " + parseAttachErr.Error())
		}
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
	getShellProps := map[string]any{
		runtime2.SSRShellMarkerAttribute: getShellMarker,
	}
	return renderParallelRegionShellNode(getShellProps, getChild)
}

// handleParallelRegionRendererMetadata merges one rendered public event-slot declaration set into shared runtime2 renderer metadata.
func handleParallelRegionRendererMetadata(
	parseRendererID runtime2.RendererID,
	parseEventSlotMetadata runtime2.EventSlotMetadata,
) error {
	if len(parseEventSlotMetadata.Slots) == 0 {
		return nil
	}
	getRendererMetadata, parseResolveErr := runtime2.ResolveRendererMetadata(parseRendererID)
	if parseResolveErr != nil {
		return parseResolveErr
	}
	getMergedEventSlotMetadata, parseMergeErr := buildParallelRegionMergedEventSlotMetadata(
		getRendererMetadata.EventSlotMetadata,
		parseEventSlotMetadata,
	)
	if parseMergeErr != nil {
		return parseMergeErr
	}
	getRendererMetadata.EventSlotMetadata = getMergedEventSlotMetadata
	if parseSetErr := runtime2.SetRendererMetadata(parseRendererID, getRendererMetadata); parseSetErr != nil {
		return parseSetErr
	}
	return cacheParallelRegionWorkerRuntime.SetWorkerRegionRendererMetadata(string(parseRendererID), getRendererMetadata)
}

// handleParallelRegionPostRenderAttachByID marks one mounted parallel-region adapter worker-attached after commit.
func handleParallelRegionPostRenderAttachByID(parseRegionInstanceID runtime2.RegionInstanceID) error {
	if parseRegionInstanceID == "" {
		return nil
	}
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter(string(parseRegionInstanceID))
	if !hasParallelRegionHostAdapter {
		return nil
	}
	getRegionDOMIndex := getParallelRegionHostAdapter.GetHostRegionDOMIndex()
	if getRegionDOMIndex == nil {
		return fmt.Errorf("ui: parallel-region DOM index is required for post-render attach")
	}
	getAnchorNode, parseAnchorLookupErr := getRegionDOMIndex.GetRegionDOMNode(string(parseRegionInstanceID), 1)
	if parseAnchorLookupErr != nil {
		if parseAnchorErr := getRegionDOMIndex.SetRegionDOMNode(
			string(parseRegionInstanceID),
			1,
			&runtime2.RegionDOMNode{
				GetNodeID: 1,
				GetTag:    "div",
			},
		); parseAnchorErr != nil {
			return parseAnchorErr
		}
	} else if getAnchorNode.GetTag != "" && getAnchorNode.GetTag != "div" {
		return fmt.Errorf("ui: parallel-region shell anchor tag %q does not match expected public shell tag %q", getAnchorNode.GetTag, "div")
	}
	return getParallelRegionHostAdapter.HandleHostRegionPostRenderAttach()
}

// buildParallelRegionSchedulerShardIDs validates the public scheduler shard list and applies the default single-shard path.
func buildParallelRegionSchedulerShardIDs(parseSchedulerShardIDs []string) ([]runtime2.SchedulerShardID, error) {
	if len(parseSchedulerShardIDs) == 0 {
		return []runtime2.SchedulerShardID{"ui-parallel-region"}, nil
	}
	getSchedulerShardIDs := make([]runtime2.SchedulerShardID, 0, len(parseSchedulerShardIDs))
	parseSeenSchedulerShards := make(map[string]bool, len(parseSchedulerShardIDs))
	for _, parseSchedulerShardID := range parseSchedulerShardIDs {
		getSchedulerShardID := strings.TrimSpace(parseSchedulerShardID)
		if getSchedulerShardID == "" {
			return nil, fmt.Errorf("ui: parallel-region scheduler shard ID is required")
		}
		if parseSeenSchedulerShards[getSchedulerShardID] {
			return nil, fmt.Errorf("ui: parallel-region scheduler shard ID %q is duplicated", getSchedulerShardID)
		}
		parseSeenSchedulerShards[getSchedulerShardID] = true
		getSchedulerShardIDs = append(getSchedulerShardIDs, runtime2.SchedulerShardID(getSchedulerShardID))
	}
	return getSchedulerShardIDs, nil
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
	getParallelRegionRendererEntry, parseResolveErr := resolveParallelRegionRendererEntry(parseRendererID)
	if parseResolveErr != nil {
		return nil, parseResolveErr
	}
	return getParallelRegionRendererEntry.getRender, nil
}

// resolveParallelRegionRendererEntry resolves one registered public parallel-region renderer entry by stable ID.
func resolveParallelRegionRendererEntry(parseRendererID string) (parallelRegionRendererEntry, error) {
	getRendererID, parseRendererIDErr := runtime2.ParseRendererID(parseRendererID)
	if parseRendererIDErr != nil {
		return parallelRegionRendererEntry{}, parseRendererIDErr
	}
	storeParallelRegionRendererMu.RLock()
	defer storeParallelRegionRendererMu.RUnlock()
	getParallelRegionRendererEntry, hasParallelRegionRenderer := cacheParallelRegionRendererByID[getRendererID]
	if !hasParallelRegionRenderer {
		return parallelRegionRendererEntry{}, fmt.Errorf("ui: parallel-region renderer %q is not registered", getRendererID)
	}
	return getParallelRegionRendererEntry, nil
}

// buildParallelRegionHostAdapter creates and mounts one runtime2 host adapter for a public parallel region when browser lifecycle support is active.
func buildParallelRegionHostAdapter(parseRuntimeSpec runtime2.ParallelRegionSpec, parseSchedulerShardIDs []runtime2.SchedulerShardID) (*runtime2.HostRegionAdapter, bool, error) {
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	if getParallelRegionHostAdapter := cacheParallelRegionAdapterByID[parseRuntimeSpec.RegionInstanceID]; getParallelRegionHostAdapter != nil {
		if hasParallelRegionSchedulerShardChange(cacheParallelRegionSchedulerShardsByID[parseRuntimeSpec.RegionInstanceID], parseSchedulerShardIDs) {
			if _, parseOwnerRemoveErr := getParallelRegionHostAdapter.HandleHostRegionOwnerRemove(); parseOwnerRemoveErr != nil {
				return nil, false, parseOwnerRemoveErr
			}
			delete(cacheParallelRegionAdapterByID, parseRuntimeSpec.RegionInstanceID)
		} else {
			getHasRemounted, parseRemountErr := handleParallelRegionStructuralRemount(getParallelRegionHostAdapter, parseRuntimeSpec)
			if parseRemountErr != nil {
				return nil, false, parseRemountErr
			}
			return getParallelRegionHostAdapter, getHasRemounted, nil
		}
	}
	getParallelRegionHostAdapter, parseHostAdapterErr := runtime2.BuildHostRegionAdapter(
		parseRuntimeSpec.RegionInstanceID,
		parseSchedulerShardIDs,
	)
	if parseHostAdapterErr != nil {
		return nil, false, parseHostAdapterErr
	}
	if _, parseMountErr := getParallelRegionHostAdapter.HandleHostRegionMount(parseRuntimeSpec, 1); parseMountErr != nil {
		return nil, false, parseMountErr
	}
	cacheParallelRegionAdapterByID[parseRuntimeSpec.RegionInstanceID] = getParallelRegionHostAdapter
	cacheParallelRegionSchedulerShardsByID[parseRuntimeSpec.RegionInstanceID] = append([]runtime2.SchedulerShardID(nil), parseSchedulerShardIDs...)
	return getParallelRegionHostAdapter, true, nil
}

// hasParallelRegionSchedulerShardChange reports whether the public scheduler shard configuration changed between renders.
func hasParallelRegionSchedulerShardChange(parseCurrent []runtime2.SchedulerShardID, parseNext []runtime2.SchedulerShardID) bool {
	if len(parseCurrent) != len(parseNext) {
		return true
	}
	for parseShardIndex := range parseCurrent {
		if parseCurrent[parseShardIndex] != parseNext[parseShardIndex] {
			return true
		}
	}
	return false
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
	getSourceValues := make(map[string]any, len(getSourceIDs))
	getSourceVersions := make(map[string]uint64, len(getSourceIDs))
	for _, getSourceID := range getSourceIDs {
		getSourceValue, hasSourceValue := getParallelRegionSourceAtomValue(getSourceID)
		if !hasSourceValue {
			return nil, nil, fmt.Errorf("ui: parallel-region declared source %q is not available", getSourceID)
		}
		getSourceValues[getSourceID] = getSourceValue
		getSourceVersions[getSourceID] = getInputVersion
	}
	return getSourceValues, getSourceVersions, nil
}

// handleParallelRegionUpdateDispatch forwards one public rerender into runtime2 host update dispatch using the currently available snapshot transport tier.
func handleParallelRegionUpdateDispatch(
	parseHostRegionAdapter *runtime2.HostRegionAdapter,
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseInputVersion uint64,
) (runtime2.HostRegionUpdateDispatchTransportResult, error) {
	if parseHostRegionAdapter == nil {
		return runtime2.HostRegionUpdateDispatchTransportResult{}, fmt.Errorf("ui: parallel-region host adapter is required")
	}
	getDispatchPriority := runtime2.HostRegionDispatchPriorityUrgent
	if isParallelRegionTransitionUpdate() {
		getDispatchPriority = runtime2.HostRegionDispatchPriorityDeferred
	}
	getCapabilityReport := buildParallelRegionCapabilityReport()
	return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransportPriority(
		parseRuntimeSpec,
		parseInputVersion,
		getDispatchPriority,
		getCapabilityReport,
		nil,
	)
}

// buildParallelRegionCapabilityReport resolves runtime2 transport capabilities and lazily initializes live runtime detection when unset.
func buildParallelRegionCapabilityReport() runtime2.CapabilityReport {
	getCapabilityReport := runtime2.GetCapabilityReport()
	if getCapabilityReport.HasWorkerSupport {
		return getCapabilityReport
	}
	getInitializedCapabilityReport, parseInitErr := runtime2.InitCapabilityReportFromRuntime()
	if parseInitErr != nil {
		return getCapabilityReport
	}
	return getInitializedCapabilityReport
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
	if isParallelRegionRenderResultNil(getResults[0]) {
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

// isParallelRegionRenderResultNil reports whether one renderer return value is nil on a nilable reflect kind.
func isParallelRegionRenderResultNil(parseValue reflect.Value) bool {
	switch parseValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return parseValue.IsNil()
	default:
		return false
	}
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
	delete(cacheParallelRegionRenderedNodeByID, getRegionInstanceID)
	delete(cacheParallelRegionInputVersionByID, getRegionInstanceID)
	delete(cacheParallelRegionSchedulerShardsByID, getRegionInstanceID)
	delete(cacheParallelRegionHydrationMarkerByID, getRegionInstanceID)
	delete(cacheParallelRegionBridgeFallbackByID, getRegionInstanceID)
	storeParallelRegionAdapterMu.Unlock()
	cacheParallelRegionWorkerRuntime.HandleWorkerRegionDispose(parseRegionInstanceID)
	if getParallelRegionHostAdapter == nil {
		return nil
	}
	_, parseOwnerRemoveErr := getParallelRegionHostAdapter.HandleHostRegionOwnerRemove()
	return parseOwnerRemoveErr
}

// GetParallelRegionRuntimeStatus reports one read-only public runtime snapshot for one tracked parallel region instance.
func GetParallelRegionRuntimeStatus(parseRegionInstanceID string) (ParallelRegionStatus, bool, error) {
	getRegionInstanceID, parseRegionInstanceIDErr := runtime2.ParseRegionInstanceID(parseRegionInstanceID)
	if parseRegionInstanceIDErr != nil {
		return ParallelRegionStatus{}, false, parseRegionInstanceIDErr
	}
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter(string(getRegionInstanceID))
	if !hasParallelRegionHostAdapter || getParallelRegionHostAdapter == nil {
		return ParallelRegionStatus{}, false, nil
	}
	getRuntimeStatus, hasRuntimeStatus := getParallelRegionHostAdapter.GetHostRegionRuntimeStatus()
	if !hasRuntimeStatus {
		return ParallelRegionStatus{}, false, nil
	}
	getStatus := buildParallelRegionStatus(getRuntimeStatus)
	// Surface a UI-side worker-bridge failure that the runtime status does not carry
	// (the runtime never learned of it because the bridge failed before dispatch).
	if getStatus.GetFallbackReason == "" {
		getStatus.GetFallbackReason = resolveParallelRegionBridgeFallback(getRegionInstanceID)
	}
	return getStatus, true, nil
}

// buildParallelRegionStatus converts one internal runtime2 status snapshot into the public read-only ui status surface.
func buildParallelRegionStatus(parseRuntimeStatus runtime2.HostRegionRuntimeStatus) ParallelRegionStatus {
	return ParallelRegionStatus{
		GetRegionInstanceID:            string(parseRuntimeStatus.GetRegionInstanceID),
		GetRegionMode:                  string(parseRuntimeStatus.GetRegionMode),
		GetAssignedWorkerShard:         parseRuntimeStatus.GetAssignedWorkerShard,
		GetRendererID:                  string(parseRuntimeStatus.GetRendererID),
		GetEpoch:                       parseRuntimeStatus.GetEpoch,
		GetIsHydrationComplete:         parseRuntimeStatus.GetIsHydrationComplete,
		HasHydratedShellAnchor:         parseRuntimeStatus.HasHydratedShellAnchor,
		HasPostHydrationAttached:       parseRuntimeStatus.HasPostHydrationAttached,
		GetLastSnapshotVersion:         parseRuntimeStatus.GetLastSnapshotVersion,
		GetLastDispatchedVersion:       parseRuntimeStatus.GetLastDispatchedVersion,
		GetLastCommittedVersion:        parseRuntimeStatus.GetLastCommittedVersion,
		GetTransportTier:               string(parseRuntimeStatus.GetTransportTier),
		GetDroppedStalePatchCount:      parseRuntimeStatus.GetDroppedStalePatchCount,
		GetIgnoredStaleDiagnosticCount: parseRuntimeStatus.GetIgnoredStaleDiagnosticCount,
		GetFallbackReason:              parseRuntimeStatus.GetFallbackReason,
	}
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
	cacheParallelRegionRenderedNodeByID = map[runtime2.RegionInstanceID]Node{}
	cacheParallelRegionInputVersionByID = map[runtime2.RegionInstanceID]uint64{}
	cacheParallelRegionSchedulerShardsByID = map[runtime2.RegionInstanceID][]runtime2.SchedulerShardID{}
	cacheParallelRegionHydrationMarkerByID = map[runtime2.RegionInstanceID]runtime2.SSRShellMarker{}
	cacheParallelRegionBridgeFallbackByID = map[runtime2.RegionInstanceID]string{}
	storeParallelRegionAdapterMu.Unlock()
	cacheParallelRegionWorkerRuntime = runtime2.BuildWorkerRegionRuntime()
	runtime2.ResetRendererRegistry()
}
