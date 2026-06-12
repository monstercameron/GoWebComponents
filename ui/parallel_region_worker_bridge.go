package ui

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// storeParallelRegionRenderedNode caches one rendered public region node for worker-side canonical conversion.
func storeParallelRegionRenderedNode(parseRegionInstanceID runtime2.RegionInstanceID, parseNode Node) {
	if parseRegionInstanceID == "" {
		return
	}
	storeParallelRegionAdapterMu.Lock()
	defer storeParallelRegionAdapterMu.Unlock()
	cacheParallelRegionRenderedNodeByID[parseRegionInstanceID] = parseNode
}

// resolveParallelRegionRenderedNode resolves one cached rendered public region node for worker-side canonical conversion.
func resolveParallelRegionRenderedNode(parseRegionInstanceID runtime2.RegionInstanceID) (Node, bool) {
	if parseRegionInstanceID == "" {
		return nil, false
	}
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	parseNode, hasNode := cacheParallelRegionRenderedNodeByID[parseRegionInstanceID]
	return parseNode, hasNode
}

// handleParallelRegionWorkerMount seeds worker-side region state from the already-rendered public node without re-executing the renderer.
func handleParallelRegionWorkerMount(
	parseHostRegionAdapter *runtime2.HostRegionAdapter,
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseInputVersion uint64,
) error {
	parseSnapshotEnvelope, parseSnapshotErr := buildParallelRegionWorkerSnapshot(parseHostRegionAdapter, parseRuntimeSpec, parseInputVersion)
	if parseSnapshotErr != nil {
		return parseSnapshotErr
	}
	cacheParallelRegionWorkerRuntime.HandleWorkerRegionDispose(string(parseRuntimeSpec.RegionInstanceID))
	parseWorkerRegionState, parseMountErr := cacheParallelRegionWorkerRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     string(parseRuntimeSpec.RegionInstanceID),
		RendererID:   string(parseRuntimeSpec.RendererID),
		Epoch:        parseSnapshotEnvelope.Epoch,
		InputVersion: parseInputVersion,
		Snapshot:     parseSnapshotEnvelope,
	})
	if parseMountErr != nil {
		return parseMountErr
	}
	_, parseApplyErr := runtime2.ApplyRegionDOMCanonicalSnapshot(
		parseHostRegionAdapter.GetHostRegionDOMIndex(),
		string(parseRuntimeSpec.RegionInstanceID),
		parseWorkerRegionState.RenderIR,
	)
	return parseApplyErr
}

// handleParallelRegionWorkerUpdate advances one mounted public region through worker-side diff and host-side patch commit when snapshot dispatch schedules work.
func handleParallelRegionWorkerUpdate(
	parseHostRegionAdapter *runtime2.HostRegionAdapter,
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseInputVersion uint64,
	parseDispatchResult runtime2.HostRegionUpdateDispatchTransportResult,
) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("ui: parallel-region host adapter is required")
	}
	if _, hasWorkerRegionState := cacheParallelRegionWorkerRuntime.GetWorkerRegionState(string(parseRuntimeSpec.RegionInstanceID)); !hasWorkerRegionState {
		return handleParallelRegionWorkerMount(parseHostRegionAdapter, parseRuntimeSpec, parseInputVersion)
	}
	if !parseDispatchResult.GetDispatchResult.HasScheduled {
		return nil
	}
	parseSnapshotEnvelope := parseDispatchResult.GetDispatchResult.GetSnapshotEnvelope
	if parseSnapshotEnvelope.RegionInstanceID == "" {
		return fmt.Errorf("ui: parallel-region scheduled dispatch snapshot is required")
	}
	parseWorkerUpdateResult, parseWorkerUpdateErr := cacheParallelRegionWorkerRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     string(parseRuntimeSpec.RegionInstanceID),
			RendererID:   string(parseRuntimeSpec.RendererID),
			Epoch:        parseSnapshotEnvelope.Epoch,
			InputVersion: parseInputVersion,
			Snapshot:     parseSnapshotEnvelope,
		},
		buildParallelRegionWorkerCapabilityReport(),
	)
	if parseWorkerUpdateErr != nil {
		return parseWorkerUpdateErr
	}
	if !parseWorkerUpdateResult.HasPatchPayload {
		return nil
	}
	if parseTierErr := parseHostRegionAdapter.SetHostRegionPatchTransportTier(parseWorkerUpdateResult.GetTransportTier); parseTierErr != nil {
		return parseTierErr
	}
	_, parseConsumeErr := parseHostRegionAdapter.HandleHostRegionPatchConsume(
		parseWorkerUpdateResult.GetTransportTier,
		parseWorkerUpdateResult.GetPatchPayload,
		nil,
		nil,
	)
	return parseConsumeErr
}

// buildParallelRegionWorkerSnapshot builds one worker mount snapshot from the current public props and declared source values.
func buildParallelRegionWorkerSnapshot(
	parseHostRegionAdapter *runtime2.HostRegionAdapter,
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseInputVersion uint64,
) (runtime2.SnapshotEnvelope, error) {
	if parseHostRegionAdapter == nil {
		return runtime2.SnapshotEnvelope{}, fmt.Errorf("ui: parallel-region host adapter is required")
	}
	parseCoordinator := parseHostRegionAdapter.GetHostRegionCoordinator()
	if parseCoordinator == nil {
		return runtime2.SnapshotEnvelope{}, fmt.Errorf("ui: parallel-region coordinator is required")
	}
	parseCoordinatorEntry, hasCoordinatorEntry := parseCoordinator.GetEntry(parseRuntimeSpec.RegionInstanceID)
	if !hasCoordinatorEntry {
		return runtime2.SnapshotEnvelope{}, fmt.Errorf("ui: parallel-region %q is not mounted", parseRuntimeSpec.RegionInstanceID)
	}
	parseSourceValues, parseSourceVersions, parseSourceErr := buildParallelRegionSourceSnapshot(
		parseRuntimeSpec.RegionInstanceID,
		parseRuntimeSpec.SourceIDs,
	)
	if parseSourceErr != nil {
		return runtime2.SnapshotEnvelope{}, parseSourceErr
	}
	return runtime2.BuildSnapshotEnvelope(
		parseRuntimeSpec.RegionInstanceID,
		parseCoordinatorEntry.Epoch,
		parseInputVersion,
		parseRuntimeSpec.Props,
		parseRuntimeSpec.SourceIDs,
		parseSourceValues,
		parseSourceVersions,
	)
}

// buildParallelRegionWorkerCapabilityReport resolves one patch-transport capability report for the public in-process bridge.
func buildParallelRegionWorkerCapabilityReport() runtime2.CapabilityReport {
	parseCapabilityReport := buildParallelRegionCapabilityReport()
	if parseCapabilityReport.HasStructuredCloneSupport ||
		parseCapabilityReport.HasBinaryTransportSupport ||
		parseCapabilityReport.HasSharedMemoryTransportSupport {
		return parseCapabilityReport
	}
	// The public bridge runs in-process and only needs one serializable patch envelope.
	// When runtime2 worker capability detection is unset, fall back to structured-clone
	// semantics instead of disabling worker patch commit entirely.
	return runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	}
}

// buildParallelRegionWorkerRenderOutput converts one already-rendered public node into display-only runtime2 render output.
func buildParallelRegionWorkerRenderOutput(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
	getParallelRegionRendererEntry := parallelRegionRendererEntry{}
	if parseMount.RendererID != "" {
		getResolvedParallelRegionRendererEntry, parseResolveErr := resolveParallelRegionRendererEntry(parseMount.RendererID)
		if parseResolveErr != nil {
			return nil, parseResolveErr
		}
		getParallelRegionRendererEntry = getResolvedParallelRegionRendererEntry
	}
	if getParallelRegionRendererEntry.getWorkerRender != nil {
		parseChildOutput, parseRenderErr := getParallelRegionRendererEntry.getWorkerRender(parseMount)
		if parseRenderErr != nil {
			return nil, parseRenderErr
		}
		return buildParallelRegionWorkerShellOutputFromRenderOutput(parseChildOutput), nil
	}
	if parseMount.RenderInput.GetEventSlot != nil {
		getRenderedNode, parseRenderErr := buildParallelRegionLocalNode(getParallelRegionRendererEntry.getRender, parseMount.Snapshot.Props)
		if parseRenderErr != nil {
			return nil, parseRenderErr
		}
		getRenderedNode, _, parseBridgeErr := buildParallelRegionBridgedNode(runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID(parseMount.RendererID),
			RegionInstanceID: runtime2.RegionInstanceID(parseMount.RegionID),
			Props:            parseMount.Snapshot.Props,
		}, getRenderedNode)
		if parseBridgeErr != nil {
			return nil, parseBridgeErr
		}
		storeParallelRegionRenderedNode(runtime2.RegionInstanceID(parseMount.RegionID), getRenderedNode)
	}
	parseNode, hasNode := resolveParallelRegionRenderedNode(runtime2.RegionInstanceID(parseMount.RegionID))
	if !hasNode {
		return nil, fmt.Errorf("ui: rendered parallel-region node %q is not cached", parseMount.RegionID)
	}
	return buildParallelRegionWorkerShellOutput(parseNode)
}

// buildParallelRegionWorkerShellOutput wraps one rendered public node in the stable shell element used by the public parallel-region path.
func buildParallelRegionWorkerShellOutput(parseNode Node) (any, error) {
	parseChildOutput, parseChildErr := buildParallelRegionWorkerNodeOutput(parseNode)
	if parseChildErr != nil {
		return nil, parseChildErr
	}
	return buildParallelRegionWorkerShellOutputFromRenderOutput(parseChildOutput), nil
}

// buildParallelRegionWorkerShellOutputFromRenderOutput wraps one worker-rendered child output in the stable shell element used by the public parallel-region path.
func buildParallelRegionWorkerShellOutputFromRenderOutput(parseChildOutput any) any {
	parseShellOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
	}
	if parseChildOutput != nil {
		parseShellOutput["children"] = []any{parseChildOutput}
	}
	return parseShellOutput
}

// buildParallelRegionWorkerNodeOutput converts one rendered public node subtree into runtime2 display-only render output.
func buildParallelRegionWorkerNodeOutput(parseNode Node) (any, error) {
	if parseNode == nil {
		return nil, nil
	}
	switch getNodeType := parseNode.Type.(type) {
	case string:
		switch getNodeType {
		case "TEXT_ELEMENT":
			return map[string]any{
				"kind": "text",
				"text": parseNode.TextContent,
			}, nil
		case "FRAGMENT":
			parseChildrenOutput, parseChildrenErr := buildParallelRegionWorkerChildrenOutput(parseNode.Children)
			if parseChildrenErr != nil {
				return nil, parseChildrenErr
			}
			parseFragmentOutput := map[string]any{
				"kind": "fragment",
			}
			if len(parseChildrenOutput) > 0 {
				parseFragmentOutput["children"] = parseChildrenOutput
			}
			return parseFragmentOutput, nil
		default:
			parsePropsOutput, parseNodeKey, parsePropsErr := buildParallelRegionWorkerPropsOutput(parseNode.Props)
			if parsePropsErr != nil {
				return nil, parsePropsErr
			}
			parseChildrenOutput, parseChildrenErr := buildParallelRegionWorkerChildrenOutput(parseNode.Children)
			if parseChildrenErr != nil {
				return nil, parseChildrenErr
			}
			parseElementOutput := map[string]any{
				"kind": "host-element",
				"tag":  getNodeType,
			}
			if len(parsePropsOutput) > 0 {
				parseElementOutput["props"] = parsePropsOutput
			}
			if parseNodeKey != "" {
				parseElementOutput["key"] = parseNodeKey
			}
			if len(parseChildrenOutput) > 0 {
				parseElementOutput["children"] = parseChildrenOutput
			}
			return parseElementOutput, nil
		}
	case *runtime.PortalElementType:
		return nil, fmt.Errorf("ui: public parallel-region worker bridge does not support portal nodes")
	case *runtime.ReactiveRegionElementType:
		return nil, fmt.Errorf("ui: public parallel-region worker bridge does not support reactive-region nodes")
	case *runtime.ReactiveTextElementType:
		return nil, fmt.Errorf("ui: public parallel-region worker bridge does not support reactive-text nodes")
	default:
		return nil, fmt.Errorf("ui: public parallel-region worker bridge does not support node type %T", parseNode.Type)
	}
}

// buildParallelRegionWorkerChildrenOutput converts one rendered public child list into runtime2 render-output children.
func buildParallelRegionWorkerChildrenOutput(parseChildren []any) ([]any, error) {
	if len(parseChildren) == 0 {
		return nil, nil
	}
	parseChildrenOutput := make([]any, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		switch getChild := parseChild.(type) {
		case nil:
			continue
		case Node:
			parseChildOutput, parseChildErr := buildParallelRegionWorkerNodeOutput(getChild)
			if parseChildErr != nil {
				return nil, parseChildErr
			}
			if parseChildOutput != nil {
				parseChildrenOutput = append(parseChildrenOutput, parseChildOutput)
			}
		case string:
			parseChildrenOutput = append(parseChildrenOutput, map[string]any{
				"kind": "text",
				"text": getChild,
			})
		default:
			return nil, fmt.Errorf("ui: public parallel-region worker bridge does not support child type %T", parseChild)
		}
	}
	return parseChildrenOutput, nil
}

// buildParallelRegionWorkerPropsOutput converts one rendered public host props map into runtime2 display-only props and an optional keyed identity.
func buildParallelRegionWorkerPropsOutput(parseProps map[string]any) (map[string]any, string, error) {
	if len(parseProps) == 0 {
		return nil, "", nil
	}
	parsePropsOutput := make(map[string]any, len(parseProps))
	parseNodeKey := ""
	for getPropKey, getPropValue := range parseProps {
		if shouldParallelRegionStripWorkerProp(getPropKey) {
			continue
		}
		switch getPropKey {
		case "", "children":
			continue
		case "key":
			if getKeyValue, hasKeyValue := getPropValue.(string); hasKeyValue {
				parseNodeKey = getKeyValue
				continue
			}
			return nil, "", fmt.Errorf("ui: public parallel-region worker bridge expects string key props, got %T", getPropValue)
		default:
			parsePropsOutput[getPropKey] = getPropValue
		}
	}
	if len(parsePropsOutput) == 0 {
		return nil, parseNodeKey, nil
	}
	return parsePropsOutput, parseNodeKey, nil
}
