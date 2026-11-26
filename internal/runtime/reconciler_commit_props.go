package runtime

import "strings"

type preparedHostDOMAdapter interface {
	CreatePreparedElement(tag string, attrs []HostAttr, text string) DOMNode
}

type passiveEventDOMAdapter interface {
	AddPassiveEventListener(node DOMNode, eventType string, handler any)
	RemovePassiveEventListener(node DOMNode, eventType string, handler any)
}

func (parseRt *Runtime) createDom(parseFiber *Fiber) DOMNode {
	var parseDom DOMNode

	if parseT, parseOk := parseFiber.typeOf.(string); parseOk {
		switch parseT {
		case "TEXT_ELEMENT":
			parseText := parseFiber.textContent
			if parseText == "" && parseFiber.props != nil {
				parseText, _ = parseFiber.props["nodeValue"].(string)
			}
			parseDom = parseRt.domAdapter.CreateTextNode(parseText)
		case "FRAGMENT":
			// Fragments don't create DOM nodes - children are rendered directly
			return nil
		default:
			if parseFiber.isCompactHostProps {
				if parsePreparedAdapter, parsePreparedOk := parseRt.domAdapter.(preparedHostDOMAdapter); parsePreparedOk {
					parsePreparedText := ""
					if parseFiber.hasDirectText {
						parsePreparedText = parseFiber.textContent
					}
					parseDom = parsePreparedAdapter.CreatePreparedElement(parseT, getCreatedHostAttrs(parseFiber.getHostAttrs), parsePreparedText)
					if parseFiber.hasCompactSpecialProps && !IsDOMNodeNull(parseDom) {
						parseRt.applyCreatedDomProps(parseDom, parseFiber.props)
					}
				}
			}
			if IsDOMNodeNull(parseDom) {
				// Regular element (not TEXT_ELEMENT or FRAGMENT)
				parseDom = parseRt.domAdapter.CreateElement(parseT)
				if parseFiber.isCompactHostProps {
					parseRt.applyCompactHostAttrs(parseDom, getCreatedHostAttrs(parseFiber.getHostAttrs))
					if parseFiber.hasCompactSpecialProps {
						parseRt.applyCreatedDomProps(parseDom, parseFiber.props)
					}
				} else {
					// Apply properties only for non-text elements
					parseRt.applyCreatedDomProps(parseDom, parseFiber.props)
				}
				if parseFiber.hasDirectText {
					parseRt.domAdapter.SetTextContent(parseDom, parseFiber.textContent)
				}
			}
		}
	}
	if _, parseOk2 := parseFiber.typeOf.(*PortalElementType); parseOk2 {
		return nil
	}
	if _, parseOk3 := parseFiber.typeOf.(*ReactiveRegionElementType); parseOk3 {
		return nil
	}
	if _, parseOk4 := parseFiber.typeOf.(*ReactiveTextElementType); parseOk4 {
		return parseRt.domAdapter.CreateTextNode(reactiveTextValue(parseFiber))
	}
	// Function components don't have DOM nodes - they render their children

	return parseDom
}

// isEmptyEquivalentAttr reports whether writing an empty value for one
// attribute on a freshly created element is the same as writing nothing. Only
// class and style qualify; value="", boolean, and ARIA attributes carry meaning
// when empty and are never dropped.
func isEmptyEquivalentAttr(parseName string, parseValue string) bool {
	return parseValue == "" && (parseName == "class" || parseName == "style")
}

// getCreatedHostAttrs returns the compact attrs to write onto a freshly
// created element: the input slice itself unless it holds an empty class or
// style, in which case a filtered copy (the fiber's own slice is kept intact
// because the next render diffs against it).
func getCreatedHostAttrs(parseAttrs []HostAttr) []HostAttr {
	isParseFiltered := false
	for _, parseAttr := range parseAttrs {
		if isEmptyEquivalentAttr(parseAttr.Name, parseAttr.Value) {
			isParseFiltered = true
			break
		}
	}
	if !isParseFiltered {
		return parseAttrs
	}
	parseKept := make([]HostAttr, 0, len(parseAttrs))
	for _, parseAttr := range parseAttrs {
		if !isEmptyEquivalentAttr(parseAttr.Name, parseAttr.Value) {
			parseKept = append(parseKept, parseAttr)
		}
	}
	return parseKept
}

// applyCreatedDomProps writes the initial props of a freshly created element,
// skipping empty class and style strings that would only create an empty
// attribute.
func (parseRt *Runtime) applyCreatedDomProps(parseDom DOMNode, parseNewProps map[string]any) {
	if IsDOMNodeNull(parseDom) || len(parseNewProps) == 0 {
		return
	}
	parseRt.applyDomPropsInitial(parseDom, parseNewProps, false, true)
}

// applyCompactHostAttrs applies one pre-normalized string-attr slice without the generic prop meta walk.
//
// The attrs arrive as a slice precisely so the typed fast lane never allocates
// a props map; converting that slice back INTO a map to reach
// BatchSetAttributes therefore paid the allocation the fast lane exists to
// avoid, once per mounted element with three or more attributes. It bought
// nothing: BatchSetAttributes is a loop over SetAttribute, and SetAttribute is
// already where cross-node attribute batching happens.
func (parseRt *Runtime) applyCompactHostAttrs(parseDom DOMNode, parseAttrs []HostAttr) {
	if parseRt == nil || IsDOMNodeNull(parseDom) || len(parseAttrs) == 0 {
		return
	}
	for _, parseAttr := range parseAttrs {
		parseRt.domAdapter.SetAttribute(parseDom, parseAttr.Name, parseAttr.Value)
	}
}

// updateCompactHostAttrs diffs two deterministic compact attribute slices and
// applies only the changed attributes; it reports whether any DOM write
// happened. Slices are tiny (a handful of attributes), so the containment
// scans are linear rather than map-backed.
func (parseRt *Runtime) updateCompactHostAttrs(parseDom DOMNode, parseOldAttrs, parseNewAttrs []HostAttr) bool {
	if hostAttrsEqual(parseOldAttrs, parseNewAttrs) {
		return false
	}
	isChanged := false
	for _, parseNew := range parseNewAttrs {
		parseOldValue, hasOld := lookupHostAttr(parseOldAttrs, parseNew.Name)
		if !hasOld || parseOldValue != parseNew.Value {
			parseRt.domAdapter.SetAttribute(parseDom, parseNew.Name, parseNew.Value)
			isChanged = true
		}
	}
	for _, parseOld := range parseOldAttrs {
		if _, hasNew := lookupHostAttr(parseNewAttrs, parseOld.Name); !hasNew {
			parseRt.domAdapter.RemoveAttribute(parseDom, parseOld.Name)
			isChanged = true
		}
	}
	return isChanged
}

// lookupHostAttr finds one attribute by name in a compact attribute slice.
func lookupHostAttr(parseAttrs []HostAttr, parseName string) (string, bool) {
	for _, parseAttr := range parseAttrs {
		if parseAttr.Name == parseName {
			return parseAttr.Value, true
		}
	}
	return "", false
}

// updateDomProperties updates DOM properties with optimized batching when available
func (parseRt *Runtime) updateDomProperties(parseDom DOMNode, parseOldProps, parseNewProps map[string]any) {
	// Check if dom is nil (interface is nil) or if the concrete value is null
	if IsDOMNodeNull(parseDom) {
		return
	}

	// fmt.Printf("updateDomProperties: updating %d old props, %d new props\n", len(oldProps), len(newProps))

	// Optimization: Fast path for initial render (no old props)
	if len(parseOldProps) == 0 && len(parseNewProps) > 0 {
		parseRt.applyInitialDomProps(parseDom, parseNewProps, false)
		return
	}

	// Remove old properties
	for parseName := range parseOldProps {
		if _, parseExists := parseNewProps[parseName]; parseExists {
			continue
		}
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseTargetName := parseMeta.attrName
		if parseTargetName == "" {
			parseTargetName = parseName
		}
		if parseHandler, parseOk := passiveEventHandler(parseOldProps[parseName]); parseOk {
			parseRt.removePassiveEventHandler(parseDom, parseTargetName, parseHandler)
			continue
		}
		if parseMeta.shouldReset {
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseMeta.resetValue)
		} else if domPropUnsetsAsProperty(parseMeta, parseOldProps[parseName]) {
			// Set as a property, so it has to be cleared as one — removing an
			// attribute that was never written leaves the handler live. See
			// domPropUnsetsAsProperty.
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName, nil)
		} else {
			parseRt.domAdapter.RemoveAttribute(parseDom, parseMeta.attrName)
		}
	}

	// Set new properties
	for parseName2, parseValue := range parseNewProps {
		parseMeta2 := getPropMeta(parseName2)
		if parseMeta2.kind == propKindSkip {
			continue
		}
		parseTargetName2 := parseMeta2.attrName
		if parseTargetName2 == "" {
			parseTargetName2 = parseName2
		}

		// Optimization: Skip if value hasn't changed
		if parseOldValue, parseExists2 := parseOldProps[parseName2]; parseExists2 && fastEqual(parseOldValue, parseValue) {
			continue
		}
		parseHadOldNonPassiveHandler := false
		if parseOldValue, parseExists3 := parseOldProps[parseName2]; parseExists3 {
			if parseOldHandler, parseOldPassive := passiveEventHandler(parseOldValue); parseOldPassive {
				parseRt.removePassiveEventHandler(parseDom, parseTargetName2, parseOldHandler)
			} else if _, parseEventOk := eventTypeFromProp(parseTargetName2); parseEventOk {
				parseHadOldNonPassiveHandler = true
			}
		}
		if parseHandler, parseOk := passiveEventHandler(parseValue); parseOk {
			if parseHadOldNonPassiveHandler {
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, nil)
			}
			parseRt.addPassiveEventHandler(parseDom, parseTargetName2, parseHandler)
			continue
		}

		switch parseMeta2.kind {
		case propKindStyle:
			if parseStyles, parseOk := parseValue.(map[string]string); parseOk {
				parseRt.domAdapter.SetStyles(parseDom, parseStyles)
			} else if parseStr, parseOk2 := parseValue.(string); parseOk2 {
				// Allow inline style strings as attribute values
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr)
			}
		case propKindClass:
			if parseStr2, parseOk3 := parseValue.(string); parseOk3 {
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr2)
			}
		case propKindSpecialProperty:
			// Always set these as properties to ensure UI updates correctly
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, parseValue)
		default:
			if parseStr3, parseOk4 := parseValue.(string); parseOk4 {
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr3)
			} else {
				// Always update properties (especially event handlers which are closures)
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, parseValue)
			}
		}
	}
}

// applyInitialDomProps is an internal reconciler helper.
//
// Attribute writes go straight to the adapter. They used to be accumulated
// into a deferred batch — held in two scalar pairs for the one- and
// two-attribute cases, spilling into a map at three — which was flushed
// through BatchSetAttributes at each non-attribute write and at the end.
//
// All of that rested on BatchSetAttributes being one bridge call. It is not;
// it is a loop over SetAttribute, and SetAttribute is already where
// cross-node attribute batching happens (jsdom/attr_batch.go encodes every
// write in a commit into a single flush). So the deferral bought no fewer
// bridge crossings and cost one map per mounted element with three or more
// attributes — on the mount path, which is where this runtime allocates.
//
// Write ORDER is preserved more faithfully than before, not less: the flush
// calls existed to stop deferred attributes from landing after a property
// write issued later, and direct writes are in encounter order by
// construction.
func (parseRt *Runtime) applyInitialDomProps(parseDom DOMNode, parseNewProps map[string]any, isPreserveHydrationState bool) {
	parseRt.applyDomPropsInitial(parseDom, parseNewProps, isPreserveHydrationState, false)
}

// applyDomPropsInitial implements applyInitialDomProps; isCreate marks a
// freshly created element, where an empty class or style string is dropped.
func (parseRt *Runtime) applyDomPropsInitial(parseDom DOMNode, parseNewProps map[string]any, isPreserveHydrationState bool, isCreate bool) {
	parseStoreAttr := func(parseName string, parseValue string) {
		if isCreate && isEmptyEquivalentAttr(parseName, parseValue) {
			return
		}
		parseRt.domAdapter.SetAttribute(parseDom, parseName, parseValue)
	}

	for parseName, parseValue := range parseNewProps {
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseTargetName := parseMeta.attrName
		if parseTargetName == "" {
			parseTargetName = parseName
		}
		if isPreserveHydrationState && shouldPreserveHydrationInitialProperty(parseTargetName) {
			continue
		}
		if parseHandler, parseOk := passiveEventHandler(parseValue); parseOk {
			parseRt.addPassiveEventHandler(parseDom, parseTargetName, parseHandler)
			continue
		}

		switch parseMeta.kind {
		case propKindStyle:
			if parseStyles, parseOk := parseValue.(map[string]string); parseOk {
				parseRt.domAdapter.SetStyles(parseDom, parseStyles)
			} else if parseStr, parseOk2 := parseValue.(string); parseOk2 {
				parseStoreAttr(parseMeta.attrName, parseStr)
			}
		case propKindClass:
			if parseStr2, parseOk3 := parseValue.(string); parseOk3 {
				parseStoreAttr(parseMeta.attrName, parseStr2)
			}
		case propKindSpecialProperty:
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
		default:
			if parseStr3, parseOk4 := parseValue.(string); parseOk4 {
				parseStoreAttr(parseMeta.attrName, parseStr3)
			} else {
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
			}
		}
	}
}

func passiveEventHandler(parseValue any) (any, bool) {
	parsePassive, parseOk := parseValue.(PassiveEventHandler)
	if !parseOk || parsePassive.Handler == nil {
		return nil, false
	}
	return parsePassive.Handler, true
}

func eventTypeFromProp(parseName string) (string, bool) {
	if len(parseName) <= 2 || !strings.HasPrefix(parseName, "on") {
		return "", false
	}
	return parseName[2:], true
}

func (parseRt *Runtime) addPassiveEventHandler(parseDom DOMNode, parseName string, parseHandler any) {
	parseAdapter, parseOk := parseRt.domAdapter.(passiveEventDOMAdapter)
	parseEventType, parseEventOk := eventTypeFromProp(parseName)
	if !parseOk || !parseEventOk {
		parseRt.domAdapter.SetProperty(parseDom, parseName, parseHandler)
		return
	}
	parseAdapter.AddPassiveEventListener(parseDom, parseEventType, parseHandler)
}

func (parseRt *Runtime) removePassiveEventHandler(parseDom DOMNode, parseName string, parseHandler any) {
	parseAdapter, parseOk := parseRt.domAdapter.(passiveEventDOMAdapter)
	parseEventType, parseEventOk := eventTypeFromProp(parseName)
	if !parseOk || !parseEventOk {
		parseRt.domAdapter.SetProperty(parseDom, parseName, nil)
		return
	}
	parseAdapter.RemovePassiveEventListener(parseDom, parseEventType, parseHandler)
}

// shouldPreserveHydrationInitialProperty is an internal reconciler helper.
func shouldPreserveHydrationInitialProperty(parseName string) bool {
	switch parseName {
	case "value", "checked", "selected", "autofocus":
		return true
	default:
		return false
	}
}
