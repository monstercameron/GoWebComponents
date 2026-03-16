package runtime

import (
	"fmt"
	"strings"
)

func (rt *Runtime) queueHydrationSubscription(atomID string, fiber *Fiber, subscribe bool) {
	if rt == nil || atomID == "" || fiber == nil {
		return
	}
	rt.deferredHydrationSubscriptions = append(rt.deferredHydrationSubscriptions, hydrationSubscriptionAction{
		atomID:    atomID,
		fiber:     fiber,
		subscribe: subscribe,
	})
}

func (rt *Runtime) flushHydrationSubscriptions() {
	if rt == nil || rt.atomRegistry == nil || len(rt.deferredHydrationSubscriptions) == 0 {
		return
	}
	for _, action := range rt.deferredHydrationSubscriptions {
		if action.fiber == nil || action.atomID == "" {
			continue
		}
		if action.subscribe {
			rt.atomRegistry.Subscribe(action.atomID, action.fiber)
			continue
		}
		rt.atomRegistry.Unsubscribe(action.atomID, action.fiber)
	}
	rt.deferredHydrationSubscriptions = rt.deferredHydrationSubscriptions[:0]
}

func (rt *Runtime) flushDeferredHydrationUpdates() {
	if rt == nil || len(rt.deferredHydrationUpdates) == 0 {
		return
	}
	pending := make([]*Fiber, 0, len(rt.deferredHydrationUpdates))
	for fiber := range rt.deferredHydrationUpdates {
		pending = append(pending, fiber)
	}
	clear(rt.deferredHydrationUpdates)
	for _, fiber := range pending {
		rt.ScheduleUpdateForFiber(fiber)
	}
}

func newHydrationBoundary(parent DOMNode, cursor DOMNode) *hydrationBoundary {
	return &hydrationBoundary{
		parent: parent,
		cursor: cursor,
		active: true,
	}
}

func (rt *Runtime) claimHydrationNode(fiber *Fiber) (DOMNode, bool) {
	if fiber == nil || fiber.hydration == nil || !fiber.hydration.active {
		return nil, false
	}

	boundary := fiber.hydration
	candidate := rt.nextHydrationCandidate(boundary.cursor)
	boundary.cursor = candidate
	if candidate == nil || candidate.IsNull() {
		rt.abortHydrationBoundary(boundary, fiber, "missing DOM node for hydrated subtree")
		return nil, false
	}

	if !rt.matchesHydrationNode(fiber, candidate) {
		rt.abortHydrationBoundary(boundary, fiber, fmt.Sprintf("DOM node %s did not match expected %s", rt.describeHydrationNode(candidate), expectedHydrationFiberName(fiber)))
		return nil, false
	}

	boundary.cursor = rt.domAdapter.GetNextSibling(candidate)
	if textWarning := rt.detectHydrationTextMismatch(fiber, candidate); textWarning != "" {
		ReportDiagnostic("runtime", DiagnosticWarning, textWarning)
	}
	for _, warning := range rt.detectHydrationAttributeMismatches(fiber, candidate) {
		ReportDiagnostic("runtime", DiagnosticWarning, warning)
	}
	return candidate, true
}

func (rt *Runtime) finalizeHydrationBoundary(boundary *hydrationBoundary, owner *Fiber) {
	if boundary == nil || !boundary.active {
		return
	}

	extra := rt.nextHydrationCandidate(boundary.cursor)
	if extra == nil || extra.IsNull() {
		boundary.cursor = nil
		return
	}

	ownerName := expectedHydrationFiberName(owner)
	if ownerName == "" {
		ownerName = "hydrated subtree"
	}
	ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("hydration discarded unexpected DOM nodes under %s", ownerName))
	for node := extra; node != nil && !node.IsNull(); {
		next := rt.domAdapter.GetNextSibling(node)
		if boundary.parent != nil && !boundary.parent.IsNull() {
			rt.domAdapter.RemoveChild(boundary.parent, node)
		}
		node = next
	}
	boundary.active = false
	boundary.cursor = nil
}

func (rt *Runtime) abortHydrationBoundary(boundary *hydrationBoundary, owner *Fiber, reason string) {
	if boundary == nil || !boundary.active {
		return
	}

	ownerName := expectedHydrationFiberName(owner)
	if ownerName == "" {
		ownerName = "hydrated subtree"
	}
	ReportDiagnostic("runtime", DiagnosticWarning, fmt.Sprintf("hydration fell back to client rendering for %s: %s", ownerName, strings.TrimSpace(reason)))

	for node := rt.nextHydrationCandidate(boundary.cursor); node != nil && !node.IsNull(); {
		next := rt.domAdapter.GetNextSibling(node)
		if boundary.parent != nil && !boundary.parent.IsNull() {
			rt.domAdapter.RemoveChild(boundary.parent, node)
		}
		node = next
	}

	boundary.active = false
	boundary.cursor = nil
	boundary.fallback = true
}

func (rt *Runtime) nextHydrationCandidate(node DOMNode) DOMNode {
	for node != nil && !node.IsNull() {
		if !rt.isIgnorableHydrationNode(node) {
			return node
		}
		node = rt.domAdapter.GetNextSibling(node)
	}
	return nil
}

func (rt *Runtime) isIgnorableHydrationNode(node DOMNode) bool {
	if rt.domNodeType(node) != 3 {
		return false
	}
	return strings.TrimSpace(rt.domNodeText(node)) == ""
}

func (rt *Runtime) matchesHydrationNode(fiber *Fiber, node DOMNode) bool {
	if fiber == nil || node == nil || node.IsNull() {
		return false
	}

	typ, ok := fiber.typeOf.(string)
	if !ok {
		return false
	}

	switch typ {
	case "TEXT_ELEMENT":
		return rt.domNodeType(node) == 3
	case "FRAGMENT":
		return false
	default:
		if rt.domNodeType(node) != 1 {
			return false
		}
		return strings.EqualFold(rt.domNodeTag(node), typ)
	}
}

func (rt *Runtime) detectHydrationTextMismatch(fiber *Fiber, node DOMNode) string {
	if fiber == nil || node == nil || node.IsNull() {
		return ""
	}
	if typ, ok := fiber.typeOf.(string); !ok || typ != "TEXT_ELEMENT" {
		return ""
	}
	expected := fiber.textContent
	if expected == "" && fiber.props != nil {
		expected, _ = fiber.props["nodeValue"].(string)
	}
	actual := rt.domNodeText(node)
	if actual == expected {
		return ""
	}
	return fmt.Sprintf("hydration text mismatch for %s: server %q client %q", expectedHydrationFiberName(fiber), actual, expected)
}

func (rt *Runtime) detectHydrationAttributeMismatches(fiber *Fiber, node DOMNode) []string {
	if fiber == nil || node == nil || node.IsNull() || fiber.props == nil {
		return nil
	}
	typ, ok := fiber.typeOf.(string)
	if !ok || typ == "TEXT_ELEMENT" || typ == "FRAGMENT" {
		return nil
	}

	warnings := make([]string, 0)
	for name, value := range fiber.props {
		if shouldSkipSSRProp(name, value) {
			continue
		}
		if name == "style" {
			continue
		}

		actual, comparable := rt.readHydrationComparableValue(node, name)
		if !comparable {
			continue
		}
		if hydrationValuesEqual(name, value, actual) {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("hydration attribute mismatch for %s on <%s>: server %q client %q", name, typ, actual, stringifyHydrationValue(value)))
	}
	return warnings
}

func (rt *Runtime) readHydrationComparableValue(node DOMNode, name string) (string, bool) {
	propName := name
	switch name {
	case "class":
		propName = "className"
	case "className":
		propName = "className"
	case "htmlFor":
		propName = "htmlFor"
	}

	if rt.domAdapter == nil {
		return "", false
	}
	value := rt.domAdapter.GetProperty(node, propName)
	if value == nil {
		if propName == name {
			return "", false
		}
		value = rt.domAdapter.GetProperty(node, name)
		if value == nil {
			return "", false
		}
	}
	return stringifyHydrationValue(value), true
}

func hydrationValuesEqual(name string, expected interface{}, actual string) bool {
	switch typed := expected.(type) {
	case bool:
		if typed {
			return actual == "true" || actual == name || actual == ""
		}
		return actual == "false" || actual == "<nil>" || actual == ""
	default:
		return actual == stringifyHydrationValue(expected)
	}
}

func stringifyHydrationValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return "<nil>"
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(value)
	}
}

func expectedHydrationFiberName(fiber *Fiber) string {
	if fiber == nil {
		return ""
	}
	if typ, ok := fiber.typeOf.(string); ok {
		switch typ {
		case "TEXT_ELEMENT":
			return "text node"
		case "FRAGMENT":
			return "fragment"
		default:
			return "<" + typ + ">"
		}
	}
	_, name := describeFiber(fiber)
	return name
}

func (rt *Runtime) describeHydrationNode(node DOMNode) string {
	if node == nil || node.IsNull() {
		return "null"
	}
	switch rt.domNodeType(node) {
	case 3:
		return fmt.Sprintf("text(%q)", rt.domNodeText(node))
	case 1:
		tag := rt.domNodeTag(node)
		if tag == "" {
			return "element"
		}
		return "<" + strings.ToLower(tag) + ">"
	default:
		return fmt.Sprintf("nodeType(%d)", rt.domNodeType(node))
	}
}

func (rt *Runtime) domNodeType(node DOMNode) int {
	value := rt.domAdapter.GetProperty(node, "nodeType")
	if number, ok := normalizeHydrationInt(value); ok {
		return number
	}
	return 0
}

func (rt *Runtime) domNodeTag(node DOMNode) string {
	if value := rt.domAdapter.GetProperty(node, "tagName"); value != nil {
		if text, ok := normalizeHydrationString(value); ok {
			return text
		}
	}
	if value := rt.domAdapter.GetProperty(node, "nodeName"); value != nil {
		if text, ok := normalizeHydrationString(value); ok {
			return text
		}
	}
	return ""
}

func (rt *Runtime) domNodeText(node DOMNode) string {
	if value := rt.domAdapter.GetProperty(node, "textContent"); value != nil {
		if text, ok := normalizeHydrationString(value); ok {
			return text
		}
	}
	return ""
}

func normalizeHydrationInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case string:
		switch typed {
		case "1", "element":
			return 1, true
		case "3", "text":
			return 3, true
		}
	}
	return 0, false
}

func normalizeHydrationString(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		if value == nil {
			return "", false
		}
		return fmt.Sprint(value), true
	}
}
