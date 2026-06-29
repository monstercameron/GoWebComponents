//go:build js && wasm

package ui

import (
	"fmt"
	"reflect"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// buildParallelRegionBridgedNode strips bridge-only event slot markers, wraps local click handlers, and returns discovered slot metadata.
func buildParallelRegionBridgedNode(
	parseRuntimeSpec runtime2.ParallelRegionSpec,
	parseNode Node,
) (Node, runtime2.EventSlotMetadata, error) {
	getSlotRecords := make([]runtime2.EventSlotRecord, 0)
	parseBridgeErr := handleParallelRegionEachNode(parseNode, func(parseCurrentNode Node) error {
		if parseCurrentNode == nil || len(parseCurrentNode.Props) == 0 {
			return nil
		}
		getSlotValueRaw, hasSlotValue := parseCurrentNode.Props[parallelRegionClickSlotProp]
		delete(parseCurrentNode.Props, parallelRegionClickSlotProp)
		if !hasSlotValue {
			runtime.RefreshElementHostProps(parseCurrentNode)
			return nil
		}
		getSlotID, hasSlotID := getSlotValueRaw.(string)
		if !hasSlotID {
			return fmt.Errorf("ui: parallel-region click slot marker expects string value, got %T", getSlotValueRaw)
		}
		getSlotID = strings.TrimSpace(getSlotID)
		if getSlotID == "" {
			return fmt.Errorf("ui: parallel-region click slot marker is required")
		}
		getWrappedClickHandler := parseCurrentNode.Props["onclick"]
		if getWrappedClickHandler == nil {
			return fmt.Errorf("ui: parallel-region click slot %q requires an onclick handler", getSlotID)
		}
		getBridgedClickHandler, parseHandlerErr := buildParallelRegionClickBridgeHandler(
			parseRuntimeSpec.RegionInstanceID,
			getSlotID,
			getWrappedClickHandler,
		)
		if parseHandlerErr != nil {
			return parseHandlerErr
		}
		parseCurrentNode.Props["onclick"] = getBridgedClickHandler
		runtime.RefreshElementHostProps(parseCurrentNode)
		getSlotRecords = append(getSlotRecords, runtime2.EventSlotRecord{
			SlotID:    getSlotID,
			EventType: parallelRegionClickEventType,
		})
		return nil
	})
	if parseBridgeErr != nil {
		return nil, runtime2.EventSlotMetadata{}, parseBridgeErr
	}
	getEventSlotMetadata, parseMetadataErr := buildParallelRegionEventSlotMetadata(getSlotRecords)
	if parseMetadataErr != nil {
		return nil, runtime2.EventSlotMetadata{}, parseMetadataErr
	}
	return parseNode, getEventSlotMetadata, nil
}

// buildParallelRegionClickBridgeHandler wraps one local click handler so it also dispatches through runtime2 event transport.
func buildParallelRegionClickBridgeHandler(
	parseRegionInstanceID runtime2.RegionInstanceID,
	parseSlotID string,
	parseWrappedHandler interface{},
) (interface{}, error) {
	if parseWrappedHandler == nil {
		return nil, fmt.Errorf("ui: parallel-region click handler is required")
	}
	return runtime.BuildDOMWrappedFunctionGlobal(func(parseEvent runtime.GoEvent) {
		handleParallelRegionWrappedClick(parseWrappedHandler, parseEvent)
		if parseDispatchErr := handleParallelRegionClickDispatch(parseRegionInstanceID, parseSlotID); parseDispatchErr != nil {
			reportParallelRegionDiagnosticError("parallel-region click bridge failed: " + parseDispatchErr.Error())
		}
	}), nil
}

// handleParallelRegionWrappedClick invokes one already-wrapped local click handler using the same event shapes the DOM adapter supports.
func handleParallelRegionWrappedClick(parseWrappedHandler interface{}, parseEvent runtime.GoEvent) {
	switch getWrappedHandler := parseWrappedHandler.(type) {
	case nil:
		return
	case func():
		getWrappedHandler()
	case func(string):
		getWrappedHandler(parseEvent.GetValue())
	case func(js.Value):
		getWrappedHandler(parseEvent.JSValue())
	case func() error:
		_ = getWrappedHandler()
	case func(js.Value) error:
		_ = getWrappedHandler(parseEvent.JSValue())
	case func(runtime.GoEvent):
		getWrappedHandler(parseEvent)
	case func(runtime.GoEvent) error:
		_ = getWrappedHandler(parseEvent)
	case js.Func:
		getWrappedHandler.Invoke(parseEvent.JSValue())
	default:
		getWrappedValue := reflect.ValueOf(parseWrappedHandler)
		if !getWrappedValue.IsValid() || getWrappedValue.Kind() != reflect.Func {
			return
		}
		getWrappedType := getWrappedValue.Type()
		switch getWrappedType.NumIn() {
		case 0:
			getWrappedValue.Call(nil)
		case 1:
			getEventValue := reflect.ValueOf(parseEvent)
			getArgType := getWrappedType.In(0)
			if getEventValue.Type().AssignableTo(getArgType) {
				getWrappedValue.Call([]reflect.Value{getEventValue})
				return
			}
			getJSValue := reflect.ValueOf(parseEvent.JSValue())
			if getJSValue.Type().AssignableTo(getArgType) {
				getWrappedValue.Call([]reflect.Value{getJSValue})
			}
		}
	}
}

// handleParallelRegionClickDispatch routes one bridged click event through the mounted worker region and commits any resulting patch.
func handleParallelRegionClickDispatch(parseRegionInstanceID runtime2.RegionInstanceID, parseSlotID string) error {
	getParallelRegionHostAdapter, hasParallelRegionHostAdapter := resolveParallelRegionHostAdapter(string(parseRegionInstanceID))
	if !hasParallelRegionHostAdapter {
		return nil
	}
	getInputVersion := buildParallelRegionNextInputVersion(parseRegionInstanceID)
	parseEventResult, parseEventErr := cacheParallelRegionWorkerRuntime.HandleWorkerRegionEvent(runtime2.WorkerRegionEventSpec{
		RegionID:     string(parseRegionInstanceID),
		InputVersion: getInputVersion,
		EventSlot: runtime2.EventSlotDispatch{
			SlotID:    parseSlotID,
			EventType: parallelRegionClickEventType,
		},
	})
	if parseEventErr != nil {
		return parseEventErr
	}
	if !parseEventResult.HasPatchReady {
		return nil
	}
	getTransportTier, parseTierErr := runtime2.SelectPatchTransportTier(buildParallelRegionWorkerCapabilityReport(), false)
	if parseTierErr != nil {
		return parseTierErr
	}
	getPatchPayload, parsePayloadErr := buildParallelRegionEventPatchPayload(getTransportTier, parseEventResult.PatchIR)
	if parsePayloadErr != nil {
		return parsePayloadErr
	}
	if parseTransportErr := getParallelRegionHostAdapter.SetHostRegionPatchTransportTier(getTransportTier); parseTransportErr != nil {
		return parseTransportErr
	}
	_, parseConsumeErr := getParallelRegionHostAdapter.HandleHostRegionPatchConsume(getTransportTier, getPatchPayload, nil, nil)
	return parseConsumeErr
}

// buildParallelRegionEventPatchPayload encodes one worker event patch using the selected transport tier.
func buildParallelRegionEventPatchPayload(
	parseTransportTier runtime2.TransportTier,
	parsePatchIR runtime2.PatchStreamRaw,
) ([]byte, error) {
	switch parseTransportTier {
	case runtime2.TransportTierBinary:
		return runtime2.BuildBinaryPatchPayload(parsePatchIR)
	case runtime2.TransportTierStructuredClone:
		return runtime2.BuildStructuredClonePatchPayloadJSON(parsePatchIR)
	default:
		return nil, fmt.Errorf("ui: unsupported parallel-region event patch transport tier %q", parseTransportTier)
	}
}
