package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

const parallelRegionClickSlotProp = "data-gwc-parallel-click-slot"
const parallelRegionClickEventType = "click"

// handleParallelRegionEachNode visits one rendered node subtree depth-first.
func handleParallelRegionEachNode(parseNode Node, parseVisit func(Node) error) error {
	if parseNode == nil {
		return nil
	}
	if parseVisit != nil {
		if parseVisitErr := parseVisit(parseNode); parseVisitErr != nil {
			return parseVisitErr
		}
	}
	for _, parseChild := range parseNode.Children {
		getChildNode, hasChildNode := parseChild.(Node)
		if !hasChildNode {
			continue
		}
		if parseChildErr := handleParallelRegionEachNode(getChildNode, parseVisit); parseChildErr != nil {
			return parseChildErr
		}
	}
	return nil
}

// buildParallelRegionEventSlotMetadata builds normalized event-slot metadata from discovered slot records.
func buildParallelRegionEventSlotMetadata(parseSlots []runtime2.EventSlotRecord) (runtime2.EventSlotMetadata, error) {
	if len(parseSlots) == 0 {
		return runtime2.EventSlotMetadata{}, nil
	}
	getMetadata := runtime2.EventSlotMetadata{
		Version: runtime2.EventSlotMetadataVersionV1,
		Slots:   append([]runtime2.EventSlotRecord(nil), parseSlots...),
	}
	if parseMetadataErr := runtime2.ValidateEventSlotMetadata(getMetadata); parseMetadataErr != nil {
		return runtime2.EventSlotMetadata{}, parseMetadataErr
	}
	return getMetadata, nil
}

// buildParallelRegionMergedEventSlotMetadata merges renderer slot declarations and preserves a normalized union.
func buildParallelRegionMergedEventSlotMetadata(
	parseCurrent runtime2.EventSlotMetadata,
	parseNext runtime2.EventSlotMetadata,
) (runtime2.EventSlotMetadata, error) {
	if len(parseCurrent.Slots) == 0 && len(parseNext.Slots) == 0 {
		return runtime2.EventSlotMetadata{}, nil
	}
	getMergedSlots := make([]runtime2.EventSlotRecord, 0, len(parseCurrent.Slots)+len(parseNext.Slots))
	cacheSeenSlotKeys := map[string]bool{}
	for _, getSlot := range parseCurrent.Slots {
		getSlotKey := getSlot.SlotID + "\x00" + getSlot.EventType
		if cacheSeenSlotKeys[getSlotKey] {
			continue
		}
		cacheSeenSlotKeys[getSlotKey] = true
		getMergedSlots = append(getMergedSlots, getSlot)
	}
	for _, getSlot := range parseNext.Slots {
		getSlotKey := getSlot.SlotID + "\x00" + getSlot.EventType
		if cacheSeenSlotKeys[getSlotKey] {
			continue
		}
		cacheSeenSlotKeys[getSlotKey] = true
		getMergedSlots = append(getMergedSlots, getSlot)
	}
	return buildParallelRegionEventSlotMetadata(getMergedSlots)
}

// hasParallelRegionWorkerEventProp reports whether one public host prop should be stripped from display-only worker output.
func hasParallelRegionWorkerEventProp(parsePropKey string) bool {
	switch parsePropKey {
	case "onclick", "oninput", "onchange", "onsubmit", "onkeydown", "onkeyup",
		"onmouseup", "onmousedown", "onmouseenter", "onmouseleave", "ondblclick",
		"oncontextmenu", "onwheel", "ontransitionend", "onanimationend", "onload",
		"onerror", "onpointerdown", "onpointermove", "onpointerup", "ontouchstart",
		"ontouchmove", "ontouchend", "ondragstart", "ondragover", "ondrop",
		"ondragend", "onfocus", "onblur", "onscroll":
		return true
	default:
		return false
	}
}

// shouldParallelRegionStripWorkerProp reports whether one public host prop is
// bridge-only and must not reach the display-only worker render output. Bridge
// event props (the click slot + on*-named handlers) are stripped by name; ANY
// function-valued prop is also stripped regardless of name. A function cannot be
// JSON-encoded into the worker props, so an arbitrarily-named callback (e.g.
// "onCustom" or a raw Go func) that slipped past the name allowlist would otherwise
// make the first patch encode fail and silently degrade the region.
func shouldParallelRegionStripWorkerProp(parsePropKey string, parsePropValue any) bool {
	if parsePropKey == parallelRegionClickSlotProp || hasParallelRegionWorkerEventProp(parsePropKey) {
		return true
	}
	return parsePropValue != nil && reflect.TypeOf(parsePropValue).Kind() == reflect.Func
}
