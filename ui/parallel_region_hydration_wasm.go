//go:build js && wasm

package ui

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

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
		getHydrationHelper, parseHydrationHelperErr := runtime2.BuildHostRegionHydrationAttachHelper(getParallelRegionHostAdapter)
		if parseHydrationHelperErr != nil {
			return parseHydrationHelperErr
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
		if !getHydrationHelper.GetHostRegionIsHydrationComplete() {
			if parseHydrationErr := getHydrationHelper.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
				return parseHydrationErr
			}
		}
		if !getParallelRegionHostAdapter.HasHostRegionHydratedShellAnchor() {
			if parseAnchorErr := getHydrationHelper.HandleHostRegionRegisterHydratedShellAnchor(getAnchorNodeID, getAnchorTag); parseAnchorErr != nil {
				return parseAnchorErr
			}
		}
		if !getHydrationHelper.HasHostRegionPostHydrationAttached() {
			if _, parseAttachErr := getHydrationHelper.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
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
