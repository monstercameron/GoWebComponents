package runtime

import (
	"fmt"
	"strings"
)

// queueHydrationSubscription is a core package helper.
func (parseRt *Runtime) queueHydrationSubscription(parseAtomID string, parseFiber *Fiber, isSubscribe bool) {
	if parseRt == nil || parseAtomID == "" || parseFiber == nil {
		return
	}
	parseRt.deferredHydrationSubscriptions = append(parseRt.deferredHydrationSubscriptions, hydrationSubscriptionAction{
		atomID:    parseAtomID,
		fiber:     parseFiber,
		subscribe: isSubscribe,
	})
}

// flushHydrationSubscriptions is a core package helper.
func (parseRt *Runtime) flushHydrationSubscriptions() {
	if parseRt == nil || parseRt.atomRegistry == nil || len(parseRt.deferredHydrationSubscriptions) == 0 {
		return
	}
	for _, parseAction := range parseRt.deferredHydrationSubscriptions {
		if parseAction.fiber == nil || parseAction.atomID == "" {
			continue
		}
		if parseAction.subscribe {
			parseRt.atomRegistry.Subscribe(parseAction.atomID, parseAction.fiber)
			continue
		}
		parseRt.atomRegistry.Unsubscribe(parseAction.atomID, parseAction.fiber)
	}
	parseRt.deferredHydrationSubscriptions = parseRt.deferredHydrationSubscriptions[:0]
}

// flushDeferredHydrationUpdates is a core package helper.
func (parseRt *Runtime) flushDeferredHydrationUpdates() {
	if parseRt == nil || len(parseRt.deferredHydrationUpdates) == 0 {
		return
	}
	parsePending := make([]*Fiber, 0, len(parseRt.deferredHydrationUpdates))
	for parseFiber := range parseRt.deferredHydrationUpdates {
		parsePending = append(parsePending, parseFiber)
	}
	clear(parseRt.deferredHydrationUpdates)
	for _, parseFiber2 := range parsePending {
		parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber2, "hydration")
	}
}

// newHydrationBoundary is a core package helper.
func newHydrationBoundary(parseParent DOMNode, parseCursor DOMNode) *hydrationBoundary {
	return &hydrationBoundary{
		parent: parseParent,
		cursor: parseCursor,
		active: true,
	}
}

// hydrationDiagnosticFiber is a core package helper.
func (parseRt *Runtime) hydrationDiagnosticFiber(parseFiber *Fiber) *Fiber {
	if parseFiber == nil {
		return nil
	}
	parseKind, _ := describeFiber(parseFiber)
	if parseKind == "root" && parseFiber.child != nil {
		return parseFiber.child
	}
	return parseFiber
}

// reportHydrationDiagnostic is a core package helper.
func (parseRt *Runtime) reportHydrationDiagnostic(parseFiber *Fiber, parseMessage string) {
	parseTarget := parseRt.hydrationDiagnosticFiber(parseFiber)
	parseSeverity := DiagnosticWarning
	if parseRt != nil && parseRt.strictHydration {
		parseSeverity = DiagnosticError
	}
	ReportDiagnosticWithContext(
		"runtime",
		parseSeverity,
		parseMessage,
		diagnosticPathForFiber(parseTarget),
		diagnosticComponentStack(parseTarget),
	)
	if parseRt != nil && parseRt.strictHydration {
		parseRt.finishHydrationMetrics(true, parseMessage)
		panic(markUnhandledPanicContext("runtime", PanicPhaseHydration, "hydration", diagnosticPathForFiber(parseTarget), diagnosticComponentStack(parseTarget), parseMessage))
	}
}

// claimHydrationNode is a core package helper.
func (parseRt *Runtime) claimHydrationNode(parseFiber *Fiber) (DOMNode, bool) {
	if parseFiber == nil || parseFiber.hydration == nil || !parseFiber.hydration.active {
		return nil, false
	}

	parseBoundary := parseFiber.hydration
	parseCandidate := parseRt.nextHydrationCandidate(parseBoundary.cursor)
	parseBoundary.cursor = parseCandidate
	if IsDOMNodeNull(parseCandidate) {
		parseRt.abortHydrationBoundary(parseBoundary, parseFiber, "missing DOM node for hydrated subtree")
		return nil, false
	}

	if !parseRt.matchesHydrationNode(parseFiber, parseCandidate) {
		parseRt.abortHydrationBoundary(parseBoundary, parseFiber, fmt.Sprintf("DOM node %s did not match expected %s", parseRt.describeHydrationNode(parseCandidate), expectedHydrationFiberName(parseFiber)))
		return nil, false
	}

	parseBoundary.cursor = parseRt.domAdapter.GetNextSibling(parseCandidate)
	if parseTextWarning := parseRt.detectHydrationTextMismatch(parseFiber, parseCandidate); parseTextWarning != "" {
		parseRt.recordHydrationMismatch()
		parseRt.reportHydrationDiagnostic(parseFiber, parseTextWarning)
		// Finding #61: in non-strict mode reportHydrationDiagnostic only logs;
		// the mismatched server node is adopted as-is and stale content persists.
		// Patch the DOM node's text to the client (expected) value so the
		// adopted node reflects what the virtual tree describes.
		if !parseRt.strictHydration && parseRt.domAdapter != nil {
			parseExpectedText := ""
			parseIsTextNodeFiber := false
			switch {
			case isTextLikeFiber(parseFiber):
				parseExpectedText = textLikeFiberValue(parseFiber)
				parseIsTextNodeFiber = true
			case parseFiber.hasDirectText:
				parseExpectedText = parseFiber.textContent
			}
			parseActualText := parseRt.domNodeText(parseCandidate)
			// Finding #79: adjacent text fibers (e.g. CreateElement("p", nil,
			// "Hello", "World")) serialize with NO separator, so the browser parses
			// them as ONE text node "HelloWorld". Overwriting that node to just this
			// fiber's value would DESTROY the following fibers' text. Instead, when
			// this is a text node whose content begins with the expected value, SPLIT
			// it: keep the expected prefix here and leave the remainder as a new
			// sibling text node for the next text fiber to claim. This resolves
			// greedily for any number of adjacent text fibers.
			if parseIsTextNodeFiber && len(parseExpectedText) > 0 &&
				len(parseActualText) > len(parseExpectedText) &&
				strings.HasPrefix(parseActualText, parseExpectedText) &&
				parseRt.domNodeType(parseCandidate) == 3 {
				parseRt.domAdapter.SetTextContent(parseCandidate, parseExpectedText)
				parseRemainderNode := parseRt.domAdapter.CreateTextNode(parseActualText[len(parseExpectedText):])
				parseParent := parseRt.domAdapter.GetParent(parseCandidate)
				if !IsDOMNodeNull(parseParent) {
					if IsDOMNodeNull(parseBoundary.cursor) {
						parseRt.domAdapter.AppendChild(parseParent, parseRemainderNode)
					} else {
						parseRt.domAdapter.InsertBefore(parseParent, parseRemainderNode, parseBoundary.cursor)
					}
					// The next text fiber claims the split-off remainder.
					parseBoundary.cursor = parseRemainderNode
				}
			} else {
				parseRt.domAdapter.SetTextContent(parseCandidate, parseExpectedText)
			}
		}
	}
	for _, parseWarning := range parseRt.detectHydrationAttributeMismatches(parseFiber, parseCandidate) {
		parseRt.recordHydrationMismatch()
		parseRt.reportHydrationDiagnostic(parseFiber, parseWarning)
	}
	return parseCandidate, true
}

// finalizeHydrationBoundary is a core package helper.
func (parseRt *Runtime) finalizeHydrationBoundary(parseBoundary *hydrationBoundary, parseOwner *Fiber) {
	if parseBoundary == nil || !parseBoundary.active {
		return
	}

	parseExtra := parseRt.nextHydrationCandidate(parseBoundary.cursor)
	if IsDOMNodeNull(parseExtra) {
		parseBoundary.cursor = nil
		return
	}

	parseOwnerName := expectedHydrationFiberName(parseOwner)
	if parseOwnerName == "" {
		parseOwnerName = "hydrated subtree"
	}
	parseRt.reportHydrationDiagnostic(parseOwner, fmt.Sprintf("hydration discarded unexpected DOM nodes under %s", parseOwnerName))
	parseRemoved := 0
	for parseNode := parseExtra; !IsDOMNodeNull(parseNode); {
		parseNext := parseRt.domAdapter.GetNextSibling(parseNode)
		if !IsDOMNodeNull(parseBoundary.parent) {
			parseRt.domAdapter.RemoveChild(parseBoundary.parent, parseNode)
		}
		parseRemoved++
		parseNode = parseNext
	}
	parseRt.recordHydrationDiscarded(parseRemoved)
	parseBoundary.active = false
	parseBoundary.cursor = nil
}

// abortHydrationBoundary is a core package helper.
func (parseRt *Runtime) abortHydrationBoundary(parseBoundary *hydrationBoundary, parseOwner *Fiber, parseReason string) {
	if parseBoundary == nil || !parseBoundary.active {
		return
	}

	parseOwnerName := expectedHydrationFiberName(parseOwner)
	if parseOwnerName == "" {
		parseOwnerName = "hydrated subtree"
	}
	parseRt.recordHydrationFallback()
	parseRt.reportHydrationDiagnostic(parseOwner, fmt.Sprintf("hydration fell back to client rendering for %s: %s", parseOwnerName, strings.TrimSpace(parseReason)))

	parseRemoved := 0
	for parseNode := parseRt.nextHydrationCandidate(parseBoundary.cursor); !IsDOMNodeNull(parseNode); {
		parseNext := parseRt.domAdapter.GetNextSibling(parseNode)
		if !IsDOMNodeNull(parseBoundary.parent) {
			parseRt.domAdapter.RemoveChild(parseBoundary.parent, parseNode)
		}
		parseRemoved++
		parseNode = parseNext
	}
	parseRt.recordHydrationDiscarded(parseRemoved)

	parseBoundary.active = false
	parseBoundary.cursor = nil
	parseBoundary.fallback = true
}

// nextHydrationCandidate is a core package helper.
func (parseRt *Runtime) nextHydrationCandidate(parseNode DOMNode) DOMNode {
	for !IsDOMNodeNull(parseNode) {
		if !parseRt.isIgnorableHydrationNode(parseNode) {
			return parseNode
		}
		parseNode = parseRt.domAdapter.GetNextSibling(parseNode)
	}
	return nil
}

// isIgnorableHydrationNode is a core package helper.
func (parseRt *Runtime) isIgnorableHydrationNode(parseNode DOMNode) bool {
	if parseRt.domNodeType(parseNode) != 3 {
		return false
	}
	return strings.TrimSpace(parseRt.domNodeText(parseNode)) == ""
}

// matchesHydrationNode is a core package helper.
func (parseRt *Runtime) matchesHydrationNode(parseFiber *Fiber, parseNode DOMNode) bool {
	if parseFiber == nil || IsDOMNodeNull(parseNode) {
		return false
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveRegionElementType); parseOk {
		return false
	}
	if _, parseOk2 := parseFiber.typeOf.(*ReactiveTextElementType); parseOk2 {
		return parseRt.domNodeType(parseNode) == 3
	}

	parseTyp, parseOk3 := parseFiber.typeOf.(string)
	if !parseOk3 {
		return false
	}

	switch parseTyp {
	case "TEXT_ELEMENT":
		return parseRt.domNodeType(parseNode) == 3
	case "FRAGMENT":
		return false
	default:
		if parseRt.domNodeType(parseNode) != 1 {
			return false
		}
		return strings.EqualFold(parseRt.domNodeTag(parseNode), parseTyp)
	}
}

// detectHydrationTextMismatch is a core package helper.
func (parseRt *Runtime) detectHydrationTextMismatch(parseFiber *Fiber, parseNode DOMNode) string {
	if parseFiber == nil || IsDOMNodeNull(parseNode) {
		return ""
	}
	parseExpected := ""
	switch {
	case isTextLikeFiber(parseFiber):
		parseExpected = textLikeFiberValue(parseFiber)
	case parseFiber.hasDirectText:
		parseExpected = parseFiber.textContent
	default:
		return ""
	}
	parseActual := parseRt.domNodeText(parseNode)
	if parseActual == parseExpected {
		return ""
	}
	return fmt.Sprintf("hydration text mismatch for %s: server %q client %q", expectedHydrationFiberName(parseFiber), parseActual, parseExpected)
}

// detectHydrationAttributeMismatches is a core package helper.
func (parseRt *Runtime) detectHydrationAttributeMismatches(parseFiber *Fiber, parseNode DOMNode) []string {
	if parseFiber == nil || IsDOMNodeNull(parseNode) {
		return nil
	}
	parseTyp, parseOk := parseFiber.typeOf.(string)
	if !parseOk || parseTyp == "TEXT_ELEMENT" || parseTyp == "FRAGMENT" {
		return nil
	}

	// Typed fast-lane fibers carry attributes in the compact slice instead of
	// a props map; compare those directly against the server-rendered node.
	if isFastLaneCompactFiber(parseFiber) {
		var parseWarnings []string
		for _, parseAttr := range parseFiber.getHostAttrs {
			parseName := compactAttrPropName(parseAttr.Name)
			parseActual, parseComparable := parseRt.readHydrationComparableValue(parseNode, parseName)
			if !parseComparable {
				continue
			}
			if hydrationValuesEqual(parseName, parseAttr.Value, parseActual) {
				continue
			}
			parseWarnings = append(parseWarnings, fmt.Sprintf("hydration attribute mismatch for %s on <%s>: server %q client %q", parseName, parseTyp, parseActual, parseAttr.Value))
		}
		return parseWarnings
	}
	if parseFiber.props == nil {
		return nil
	}

	parseWarnings := make([]string, 0)
	for parseName, parseValue := range parseFiber.props {
		if shouldSkipSSRProp(parseName, parseValue) {
			continue
		}
		if parseName == "style" {
			continue
		}

		parseActual, parseComparable := parseRt.readHydrationComparableValue(parseNode, parseName)
		if !parseComparable {
			continue
		}
		if hydrationValuesEqual(parseName, parseValue, parseActual) {
			continue
		}
		parseWarnings = append(parseWarnings, fmt.Sprintf("hydration attribute mismatch for %s on <%s>: server %q client %q", parseName, parseTyp, parseActual, stringifyHydrationValue(parseValue)))
	}
	return parseWarnings
}

// readHydrationComparableValue is a core package helper.
func (parseRt *Runtime) readHydrationComparableValue(parseNode DOMNode, parseName string) (string, bool) {
	parsePropName := parseName
	switch parseName {
	case "class":
		parsePropName = "className"
	case "className":
		parsePropName = "className"
	case "htmlFor":
		parsePropName = "htmlFor"
	}

	if parseRt.domAdapter == nil {
		return "", false
	}
	parseValue := normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, parsePropName))
	if parseValue == nil {
		if parsePropName == parseName {
			return "", false
		}
		parseValue = normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, parseName))
		if parseValue == nil {
			return "", false
		}
	}
	return stringifyHydrationValue(parseValue), true
}

// hydrationValuesEqual is a core package helper.
func hydrationValuesEqual(parseName string, parseExpected any, parseActual string) bool {
	switch parseTyped := parseExpected.(type) {
	case bool:
		if parseTyped {
			return parseActual == "true" || parseActual == parseName || parseActual == ""
		}
		return parseActual == "false" || parseActual == "<nil>" || parseActual == ""
	default:
		return parseActual == stringifyHydrationValue(parseExpected)
	}
}

// stringifyHydrationValue is a core package helper.
func stringifyHydrationValue(parseValue any) string {
	switch parseTyped := parseValue.(type) {
	case nil:
		return "<nil>"
	case string:
		return parseTyped
	case bool:
		if parseTyped {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(parseValue)
	}
}

// expectedHydrationFiberName is a core package helper.
func expectedHydrationFiberName(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveRegionElementType); parseOk {
		return "reactive region"
	}
	if _, parseOk2 := parseFiber.typeOf.(*ReactiveTextElementType); parseOk2 {
		return "reactive text node"
	}
	if parseTyp, parseOk3 := parseFiber.typeOf.(string); parseOk3 {
		switch parseTyp {
		case "TEXT_ELEMENT":
			return "text node"
		case "FRAGMENT":
			return "fragment"
		default:
			return "<" + parseTyp + ">"
		}
	}
	_, parseName := describeFiber(parseFiber)
	return parseName
}

// describeHydrationNode is a core package helper.
func (parseRt *Runtime) describeHydrationNode(parseNode DOMNode) string {
	if IsDOMNodeNull(parseNode) {
		return "null"
	}
	switch parseRt.domNodeType(parseNode) {
	case 3:
		return fmt.Sprintf("text(%q)", parseRt.domNodeText(parseNode))
	case 1:
		parseTag := parseRt.domNodeTag(parseNode)
		if parseTag == "" {
			return "element"
		}
		return "<" + strings.ToLower(parseTag) + ">"
	default:
		return fmt.Sprintf("nodeType(%d)", parseRt.domNodeType(parseNode))
	}
}

// domNodeType is a core package helper.
func (parseRt *Runtime) domNodeType(parseNode DOMNode) int {
	parseValue := normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, "nodeType"))
	if parseNumber, parseOk := normalizeHydrationInt(parseValue); parseOk {
		return parseNumber
	}
	return 0
}

// domNodeTag is a core package helper.
func (parseRt *Runtime) domNodeTag(parseNode DOMNode) string {
	if parseValue := normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, "tagName")); parseValue != nil {
		if parseText, parseOk := normalizeHydrationString(parseValue); parseOk {
			return parseText
		}
	}
	if parseValue2 := normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, "nodeName")); parseValue2 != nil {
		if parseText2, parseOk2 := normalizeHydrationString(parseValue2); parseOk2 {
			return parseText2
		}
	}
	return ""
}

// domNodeText is a core package helper.
func (parseRt *Runtime) domNodeText(parseNode DOMNode) string {
	if parseValue := normalizeForeignPropertyValue(parseRt.domAdapter.GetProperty(parseNode, "textContent")); parseValue != nil {
		if parseText, parseOk := normalizeHydrationString(parseValue); parseOk {
			return parseText
		}
	}
	return ""
}

// normalizeHydrationInt is a core package helper.
func normalizeHydrationInt(parseValue any) (int, bool) {
	switch parseTyped := parseValue.(type) {
	case int:
		return parseTyped, true
	case int32:
		return int(parseTyped), true
	case int64:
		return int(parseTyped), true
	case float64:
		return int(parseTyped), true
	case float32:
		return int(parseTyped), true
	case string:
		switch parseTyped {
		case "1", "element":
			return 1, true
		case "3", "text":
			return 3, true
		}
	}
	return 0, false
}

// normalizeHydrationString is a core package helper.
func normalizeHydrationString(parseValue any) (string, bool) {
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped, true
	case []byte:
		return string(parseTyped), true
	default:
		if parseValue == nil {
			return "", false
		}
		return fmt.Sprint(parseValue), true
	}
}
