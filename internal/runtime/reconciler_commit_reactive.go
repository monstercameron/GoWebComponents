package runtime

import (
	"slices"
	"strings"
)

// oldFiberUpdateOrigin is an internal reconciler helper.
func oldFiberUpdateOrigin(parseOldFiber *Fiber, parseTypeOf any) string {
	if parseOldFiber != nil && parseOldFiber.updateOrigin != "" {
		return parseOldFiber.updateOrigin
	}
	if isFineGrainedType(parseTypeOf) {
		return "fine-grained"
	}
	return ""
}

// isReactiveTextType is an internal reconciler helper.
func isReactiveTextType(parseTypeOf any) bool {
	_, parseOk := parseTypeOf.(*ReactiveTextElementType)
	return parseOk
}

// isReactiveRegionType is an internal reconciler helper.
func isReactiveRegionType(parseTypeOf any) bool {
	_, parseOk := parseTypeOf.(*ReactiveRegionElementType)
	return parseOk
}

// isFineGrainedType is an internal reconciler helper.
func isFineGrainedType(parseTypeOf any) bool {
	return isReactiveTextType(parseTypeOf) || isReactiveRegionType(parseTypeOf)
}

// isHostFiber is an internal reconciler helper.
func isHostFiber(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	parseTyp, parseOk := parseFiber.typeOf.(string)
	if !parseOk {
		return false
	}
	switch parseTyp {
	case "ROOT", "FRAGMENT", "TEXT_ELEMENT":
		return false
	default:
		return true
	}
}

// hasFineGrainedAncestor is an internal reconciler helper.
func hasFineGrainedAncestor(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	if parseFiber.ancestorFlagsValid {
		return parseFiber.underFineGrained
	}
	for parseParent := parseFiber.parent; parseParent != nil; parseParent = parseParent.parent {
		if parseParent.fineGrained {
			return true
		}
	}
	return false
}

// recordFineGrainedDescendantCommit is an internal reconciler helper.
func (parseRt *Runtime) recordFineGrainedDescendantCommit(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || parseFiber.fineGrained || !hasFineGrainedAncestor(parseFiber) {
		return
	}
	if parseFiber.hasDirectText {
		parseRt.profiling.fineGrainedDescendantTextCommits++
	}
	if isTextLikeFiber(parseFiber) {
		parseRt.profiling.fineGrainedDescendantTextCommits++
		return
	}
	if isHostFiber(parseFiber) {
		parseRt.profiling.fineGrainedDescendantHostCommits++
	}
}

// isTextLikeFiber is an internal reconciler helper.
func isTextLikeFiber(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
		return true
	}
	parseTyp, parseOk2 := parseFiber.typeOf.(string)
	return parseOk2 && parseTyp == "TEXT_ELEMENT"
}

// textLikeFiberValue is an internal reconciler helper.
func textLikeFiberValue(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
		return parseFiber.textContent
	}
	parseValue := parseFiber.textContent
	if parseValue == "" && parseFiber.props != nil {
		parseValue, _ = parseFiber.props["nodeValue"].(string)
	}
	return parseValue
}

const (
	reactiveTextAtomIDProp      = "__gwc_reactive_text_atom_id"
	reactiveTextGetterProp      = "__gwc_reactive_text_getter"
	reactiveRegionSourceIDsProp = "__gwc_reactive_region_source_ids"
	reactiveRegionRenderProp    = "__gwc_reactive_region_render"
)

// reactiveTextValue is an internal reconciler helper.
func reactiveTextValue(parseFiber *Fiber) string {
	if parseFiber == nil || parseFiber.props == nil {
		return ""
	}
	parseGetter, _ := parseFiber.props[reactiveTextGetterProp].(func() string)
	if parseGetter == nil {
		return ""
	}
	return parseGetter()
}

// reactiveRegionValue is an internal reconciler helper.
func reactiveRegionValue(parseFiber *Fiber) *Element {
	if parseFiber == nil || parseFiber.props == nil {
		return nil
	}
	render, _ := parseFiber.props[reactiveRegionRenderProp].(func() *Element)
	if render == nil {
		return nil
	}
	return render()
}

// reactiveRegionSourceIDs is an internal reconciler helper.
func reactiveRegionSourceIDs(parseFiber *Fiber) []string {
	if parseFiber == nil || parseFiber.props == nil {
		return nil
	}
	parseRaw, _ := parseFiber.props[reactiveRegionSourceIDsProp].([]string)
	if len(parseRaw) == 0 {
		return nil
	}
	if len(parseRaw) == 1 {
		if strings.TrimSpace(parseRaw[0]) == "" {
			return nil
		}
		return parseRaw
	}
	parseIds := make([]string, 0, len(parseRaw))
	parseSeen := make(map[string]struct{}, len(parseRaw))
	for _, parseId := range parseRaw {
		if strings.TrimSpace(parseId) == "" {
			continue
		}
		if _, parseExists := parseSeen[parseId]; parseExists {
			continue
		}
		parseSeen[parseId] = struct{}{}
		parseIds = append(parseIds, parseId)
	}
	if len(parseIds) == 0 {
		return nil
	}
	return parseIds
}

// sameReactiveSourceIDs is an internal reconciler helper.
func sameReactiveSourceIDs(parseLeft []string, parseRight []string) bool {
	if len(parseLeft) != len(parseRight) {
		return false
	}
	for parseIndex := range parseLeft {
		if parseLeft[parseIndex] != parseRight[parseIndex] {
			return false
		}
	}
	return true
}

// syncFineGrainedSubscriptions is an internal reconciler helper.
func (parseRt *Runtime) syncFineGrainedSubscriptions(parseFiber *Fiber, parseSourceIDs []string) {
	if parseRt == nil || parseRt.atomRegistry == nil || parseFiber == nil {
		return
	}
	parsePrevious := parseFiber.reactiveSourceIDs
	// The joined form of parsePrevious, carried alongside it so the unchanged
	// path below can adopt the string instead of rebuilding it.
	parsePreviousJoined := parseFiber.reactiveAtomID
	if parseFiber.alternate != nil && parseFiber.alternate != parseFiber && len(parseFiber.alternate.reactiveSourceIDs) > 0 {
		parsePrevious = parseFiber.alternate.reactiveSourceIDs
		parsePreviousJoined = parseFiber.alternate.reactiveAtomID
	}
	if sameReactiveSourceIDs(parsePrevious, parseSourceIDs) {
		parseFiber.reactiveSourceIDs = parsePrevious
		parseFiber.reactiveAtomID = ""
		if len(parsePrevious) > 0 {
			// Nothing changed, so the previous generation's joined id is still
			// correct. Re-joining here allocated once per fine-grained fiber per
			// COMMIT — on the path whose whole purpose is to avoid work when the
			// subscription set is stable. The fallback covers a previous
			// generation that never recorded one.
			parseFiber.reactiveAtomID = parsePreviousJoined
			if parseFiber.reactiveAtomID == "" {
				parseFiber.reactiveAtomID = strings.Join(parsePrevious, ",")
			}
		}
		parseFiber.fineGrained = len(parsePrevious) > 0
		return
	}
	// For small subscription sets use inline linear search to avoid map allocation.
	if len(parsePrevious) <= 4 && len(parseSourceIDs) <= 4 {
		for _, parseOldID := range parsePrevious {
			parseKeep := slices.Contains(parseSourceIDs, parseOldID)
			if parseKeep {
				continue
			}
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseOldID, parseFiber, false)
				if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
					parseRt.queueHydrationSubscription(parseOldID, parseFiber.alternate, false)
				}
			} else {
				parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber)
				if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
					parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber.alternate)
				}
			}
		}
		for _, parseNewID := range parseSourceIDs {
			parseAlready := slices.Contains(parsePrevious, parseNewID)
			if parseAlready {
				continue
			}
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseNewID, parseFiber, true)
			} else {
				parseRt.atomRegistry.Subscribe(parseNewID, parseFiber)
			}
		}
	} else {
		parsePreviousSet := make(map[string]struct{}, len(parsePrevious))
		for _, parseId := range parsePrevious {
			parsePreviousSet[parseId] = struct{}{}
		}
		parseNextSet := make(map[string]struct{}, len(parseSourceIDs))
		for _, parseId2 := range parseSourceIDs {
			parseNextSet[parseId2] = struct{}{}
		}
		for _, parseOldID := range parsePrevious {
			if _, parseKeep := parseNextSet[parseOldID]; parseKeep {
				continue
			}
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseOldID, parseFiber, false)
				if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
					parseRt.queueHydrationSubscription(parseOldID, parseFiber.alternate, false)
				}
			} else {
				parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber)
				if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
					parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber.alternate)
				}
			}
		}
		for _, parseNewID := range parseSourceIDs {
			if _, parseAlready := parsePreviousSet[parseNewID]; parseAlready {
				continue
			}
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseNewID, parseFiber, true)
			} else {
				parseRt.atomRegistry.Subscribe(parseNewID, parseFiber)
			}
		}
	}
	parseFiber.reactiveSourceIDs = append([]string(nil), parseSourceIDs...)
	parseFiber.reactiveAtomID = ""
	if len(parseSourceIDs) > 0 {
		parseFiber.reactiveAtomID = strings.Join(parseSourceIDs, ",")
	}
	parseFiber.fineGrained = len(parseSourceIDs) > 0
}

// syncReactiveTextSubscription is an internal reconciler helper.
func (parseRt *Runtime) syncReactiveTextSubscription(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || parseFiber.props == nil {
		return
	}
	parseAtomID, _ := parseFiber.props[reactiveTextAtomIDProp].(string)
	// The atom-id prop may carry several comma-joined source ids (a multi-source computed),
	// so the text node subscribes to every dependency, not only the first.
	parseSourceIDs := []string{}
	for _, parseID := range strings.Split(parseAtomID, ",") {
		if strings.TrimSpace(parseID) != "" {
			parseSourceIDs = append(parseSourceIDs, parseID)
		}
	}
	parseRt.syncFineGrainedSubscriptions(parseFiber, parseSourceIDs)
	if parseAtomID != "" {
		parseFiber.reactiveAtomID = parseAtomID
	}
}

// syncReactiveRegionSubscription is an internal reconciler helper.
func (parseRt *Runtime) syncReactiveRegionSubscription(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil {
		return
	}
	parseRt.syncFineGrainedSubscriptions(parseFiber, reactiveRegionSourceIDs(parseFiber))
}
