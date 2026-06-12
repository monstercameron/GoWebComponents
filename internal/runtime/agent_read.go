package runtime

import (
	"strings"
)

// AgentNodeSnapshot is a JSON-serializable summary of one fiber node built by
// BuildAgentSnapshot. It carries the component name, kind, stable agent ref,
// hook count, dirty flags, hook value previews, and a children slice. Nodes
// beyond the depth or node budget are omitted and counted in the parent
// AgentSnapshot's TruncatedNodes field.
type AgentNodeSnapshot struct {
	// Name is the component or element name from describeFiber.
	Name string `json:"name"`
	// Kind is "component", "host", "text", "root", etc. from describeFiber.
	Kind string `json:"kind"`
	// AgentRef is the stable agent ref for this node; empty on the root fiber.
	AgentRef string `json:"agentRef,omitempty"`
	// HookCount is the number of hook slots on this fiber.
	HookCount int `json:"hookCount"`
	// Dirty reports whether this fiber has pending updates.
	Dirty bool `json:"dirty,omitempty"`
	// NeedsUpdate reports whether an ancestor queued an update for this fiber.
	NeedsUpdate bool `json:"needsUpdate,omitempty"`
	// Hooks carries hook value previews; values are redacted when a panic-report
	// redaction policy is active (fail-closed: any configured policy silences
	// hook values).
	Hooks []AgentHookPreview `json:"hooks,omitempty"`
	// Children is the (possibly truncated) list of child nodes in tree order.
	Children []AgentNodeSnapshot `json:"children,omitempty"`
}

// AgentHookPreview is a JSON-serializable hook value preview produced by
// BuildAgentSnapshot. It reuses the string previews from inspectHooks.
type AgentHookPreview struct {
	// Slot is the hook slot index.
	Slot int `json:"slot"`
	// Kind is the hook kind string (state, memo, ref, effect, …).
	Kind string `json:"kind"`
	// Value is the preview string; "[redacted]" when a redaction policy is active.
	Value string `json:"value"`
}

// AgentSnapshot is the top-level result returned by BuildAgentSnapshot.
type AgentSnapshot struct {
	// Root is the budgeted, redacted fiber tree starting from the runtime root.
	Root *AgentNodeSnapshot `json:"root,omitempty"`
	// TruncatedNodes is the count of nodes omitted to satisfy the depth or node
	// budget; zero when the full tree fit within the budget.
	TruncatedNodes int `json:"truncatedNodes,omitempty"`
	// BudgetApplied is true whenever either budget (maxDepth or maxNodes) was
	// reached and at least one node was omitted. Callers must check this field
	// rather than inferring truncation from the shape of Root.
	BudgetApplied bool `json:"budgetApplied,omitempty"`
}

// AgentQuerySelector carries the optional selector fields recognised by
// QueryAgentNodes. All fields are optional; a match requires every non-empty
// field to match.
type AgentQuerySelector struct {
	// Role matches the computed ARIA role derived from the element tag or the
	// "role" / "aria-role" prop (case-insensitive, same mapping as testkit
	// ByRole).
	Role string
	// Label matches the accessible label from "aria-label" prop (substring,
	// case-insensitive). When Role is also set, only nodes that have both the
	// correct role AND a matching label are returned.
	Label string
	// Text matches the concatenated text content of the fiber subtree
	// (substring, case-insensitive), the same text that testkit ByText uses.
	Text string
	// ID matches the "id" prop exactly (case-sensitive, same rule as testkit
	// ByID).
	ID string
	// Tag matches the host element tag (case-insensitive), same rule as testkit
	// AllByTag.
	Tag string
}

// AgentQueryMatch describes one node returned by QueryAgentNodes.
type AgentQueryMatch struct {
	// AgentRef is the stable agent ref for the matched node.
	AgentRef string `json:"agentRef"`
	// Name is the component or element name.
	Name string `json:"name"`
	// Kind is "component", "host", "text", "root", etc.
	Kind string `json:"kind"`
	// Tag is the host element tag; empty for non-host nodes.
	Tag string `json:"tag,omitempty"`
}

// BuildAgentSnapshot wraps the full Inspect() output into a
// JSON-serializable tree (AgentSnapshot) that is safe to send over the agent
// bridge wire. It enforces a depth budget (parseMaxDepth, 0 = unlimited) and a
// node-count budget (parseMaxNodes, 0 = unlimited). Whenever either budget is
// reached the result carries explicit truncation metadata in TruncatedNodes and
// BudgetApplied – truncation is never silent. Hook value previews are redacted
// when a PanicReportRedactionPolicy is configured (fail-closed: presence of
// any policy silences hook values to "[redacted]" rather than risk leaking
// sensitive data).
//
// The panic-report redaction hook (ConfigurePanicReportRedaction) is the
// runtime-local redaction mechanism used here because the telemetry redaction
// policy in internal/telemetryredaction is package-inaccessible from
// internal/runtime (it would create an import cycle through logging/).
// Using the panic-report state is therefore the documented pattern for
// runtime-local redaction, not a workaround.
func BuildAgentSnapshot(parseRt *Runtime, parseMaxDepth int, parseMaxNodes int) AgentSnapshot {
	parseInsp := parseRt.Inspect()
	if parseInsp.Root == nil {
		return AgentSnapshot{}
	}

	parseRedact := readHookRedactionActive()
	parseCounter := &readNodeCounter{limit: parseMaxNodes}
	parseRoot, parseTruncated := readBuildNode(parseInsp.Root, 0, parseMaxDepth, parseCounter, parseRedact)
	parseTotalTruncated := parseTruncated + parseCounter.truncated
	return AgentSnapshot{
		Root:           parseRoot,
		TruncatedNodes: parseTotalTruncated,
		BudgetApplied:  parseTotalTruncated > 0,
	}
}

// readNodeCounter tracks how many nodes have been emitted against an optional
// cap, accumulating the truncated count.
type readNodeCounter struct {
	limit     int
	emitted   int
	truncated int
}

// readClaimNode reports whether another node may be emitted. It increments
// emitted on success.
func (parseC *readNodeCounter) readClaimNode() bool {
	if parseC.limit <= 0 {
		parseC.emitted++
		return true
	}
	if parseC.emitted >= parseC.limit {
		return false
	}
	parseC.emitted++
	return true
}

// readBuildNode converts one FiberSnapshot into an AgentNodeSnapshot,
// recursing into children while respecting the depth and node budgets.
// It returns the node and the count of nodes in this subtree that were
// truncated due to the depth budget (node-budget truncations are counted in
// the counter directly).
func readBuildNode(parseFib *FiberSnapshot, parseDepth int, parseMaxDepth int, parseCounter *readNodeCounter, parseRedact bool) (*AgentNodeSnapshot, int) {
	if parseFib == nil {
		return nil, 0
	}
	if !parseCounter.readClaimNode() {
		// Count this node as truncated due to node budget.
		parseTruncated := readCountSubtreeSize(parseFib)
		parseCounter.truncated += parseTruncated
		return nil, 0
	}

	parseHooks := readBuildHookPreviews(parseFib.Hooks, parseRedact)
	parseNode := &AgentNodeSnapshot{
		Name:        parseFib.Name,
		Kind:        parseFib.Kind,
		AgentRef:    parseFib.AgentRef,
		HookCount:   parseFib.HookCount,
		Dirty:       parseFib.Dirty,
		NeedsUpdate: parseFib.NeedsUpdate,
		Hooks:       parseHooks,
	}

	parseDepthTruncated := 0
	parseNextDepth := parseDepth + 1
	for parseIndex := range parseFib.Children {
		// Depth budget: if max depth is set and we've reached it, count the
		// entire child subtree as truncated instead of recursing.
		if parseMaxDepth > 0 && parseNextDepth > parseMaxDepth {
			parseSubSize := readCountSubtreeSize(&parseFib.Children[parseIndex])
			parseDepthTruncated += parseSubSize
			continue
		}
		parseChildNode, parseChildTruncated := readBuildNode(&parseFib.Children[parseIndex], parseNextDepth, parseMaxDepth, parseCounter, parseRedact)
		parseDepthTruncated += parseChildTruncated
		if parseChildNode != nil {
			parseNode.Children = append(parseNode.Children, *parseChildNode)
		}
	}
	return parseNode, parseDepthTruncated
}

// readCountSubtreeSize counts the total number of FiberSnapshot nodes in a
// subtree (including the root), used to populate TruncatedNodes accurately.
func readCountSubtreeSize(parseFib *FiberSnapshot) int {
	if parseFib == nil {
		return 0
	}
	parseCount := 1
	for parseIndex := range parseFib.Children {
		parseCount += readCountSubtreeSize(&parseFib.Children[parseIndex])
	}
	return parseCount
}

// readBuildHookPreviews converts a slice of HookSnapshot values into
// AgentHookPreview values. When parseRedact is true every value is replaced
// with "[redacted]" (fail-closed).
func readBuildHookPreviews(parseHooks []HookSnapshot, parseRedact bool) []AgentHookPreview {
	if len(parseHooks) == 0 {
		return nil
	}
	parseResult := make([]AgentHookPreview, 0, len(parseHooks))
	for _, parseH := range parseHooks {
		parseValue := parseH.Value
		if parseRedact {
			parseValue = "[redacted]"
		}
		parseResult = append(parseResult, AgentHookPreview{
			Slot:  parseH.Slot,
			Kind:  parseH.Kind,
			Value: parseValue,
		})
	}
	return parseResult
}

// readHookRedactionActive reports whether a panic-report redaction policy is
// currently configured. Any configured policy causes hook value previews to be
// redacted (fail-closed). See the BuildAgentSnapshot GoDoc for rationale.
func readHookRedactionActive() bool {
	parsePolicy := CurrentPanicReportRedaction()
	return len(parsePolicy.Fields) > 0 || parsePolicy.Redact != nil
}

// QueryAgentNodes walks the live fiber tree under schedulerMu
// and returns all fibers that match the selector in tree (pre-order) order.
// It mirrors the testkit ByRole/ByLabel/ByText/ByID/AllByTag semantics:
//
//   - Role: derived from the host element tag via the same implicit role
//     mapping as testkit (button→button, a[href]→link, input[type=*]→…, etc.)
//     and from the "role" or "aria-role" prop (checked first); comparison is
//     case-insensitive after normalisation.
//   - Label: matched against the "aria-label" prop as a case-insensitive
//     substring.
//   - Text: matched against the concatenated textContent of the fiber subtree
//     as a case-insensitive substring (same depth-first concatenation as
//     testkit nodeText).
//   - ID: matched against the "id" prop exactly (case-sensitive).
//   - Tag: matched against the host element tag case-insensitively.
//
// When multiple selector fields are set, all must match (logical AND). An
// empty selector returns all non-root fibers. A nil or empty runtime returns
// an empty slice, never an error.
func QueryAgentNodes(parseRt *Runtime, parseSel AgentQuerySelector) []AgentQueryMatch {
	if parseRt == nil {
		return nil
	}

	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if parseRt.currentRoot == nil {
		return nil
	}

	parseMatches := make([]AgentQueryMatch, 0)
	readWalkFibers(parseRt.currentRoot, "", func(parseFiber *Fiber, parseRef string) {
		if readMatchesFiber(parseFiber, parseSel) {
			parseKind, parseName := describeFiber(parseFiber)
			parseTag := readFiberTag(parseFiber)
			parseMatches = append(parseMatches, AgentQueryMatch{
				AgentRef: parseRef,
				Name:     parseName,
				Kind:     parseKind,
				Tag:      parseTag,
			})
		}
	})
	return parseMatches
}

// readWalkFibers performs a pre-order depth-first walk of the fiber tree
// starting at parseFiber, calling parseVisit for every fiber including the
// root. parseRef is the accumulated agent ref for parseFiber (empty for the
// root fiber itself).
func readWalkFibers(parseFiber *Fiber, parseRef string, parseVisit func(*Fiber, string)) {
	if parseFiber == nil {
		return
	}
	parseVisit(parseFiber, parseRef)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseSegment := hotReloadFiberPathSegment(parseChild)
		parseChildRef := joinAgentRefPath(parseRef, parseSegment)
		readWalkFibers(parseChild, parseChildRef, parseVisit)
	}
}

// readMatchesFiber reports whether parseFiber satisfies all non-empty fields
// of parseSel. An empty selector always returns true.
func readMatchesFiber(parseFiber *Fiber, parseSel AgentQuerySelector) bool {
	if parseFiber == nil {
		return false
	}

	if parseSel.ID != "" {
		parseID := readPropString(parseFiber, "id")
		if parseID != parseSel.ID {
			return false
		}
	}

	if parseSel.Tag != "" {
		parseTag := readFiberTag(parseFiber)
		if !strings.EqualFold(parseTag, parseSel.Tag) {
			return false
		}
	}

	if parseSel.Role != "" {
		parseComputedRole := readFiberRole(parseFiber)
		if !strings.EqualFold(parseComputedRole, strings.TrimSpace(parseSel.Role)) {
			return false
		}
	}

	if parseSel.Label != "" {
		parseLabel := readPropString(parseFiber, "aria-label")
		if !strings.Contains(strings.ToLower(parseLabel), strings.ToLower(strings.TrimSpace(parseSel.Label))) {
			return false
		}
	}

	if parseSel.Text != "" {
		parseText := readFiberText(parseFiber)
		if !strings.Contains(strings.ToLower(parseText), strings.ToLower(strings.TrimSpace(parseSel.Text))) {
			return false
		}
	}

	return true
}

// readFiberRole computes the implicit ARIA role for a fiber using the same
// mapping as the testkit nodeRole function. The "role" or "aria-role" prop is
// checked first; if neither is set the host element tag drives the mapping.
func readFiberRole(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	// Explicit role prop overrides the implicit tag mapping (testkit: Attrs["role"]).
	if parseRole := readPropString(parseFiber, "role"); parseRole != "" {
		return strings.TrimSpace(parseRole)
	}
	if parseRole := readPropString(parseFiber, "aria-role"); parseRole != "" {
		return strings.TrimSpace(parseRole)
	}

	parseTag := strings.ToLower(strings.TrimSpace(readFiberTag(parseFiber)))
	switch parseTag {
	case "button":
		return "button"
	case "a":
		if strings.TrimSpace(readPropString(parseFiber, "href")) != "" {
			return "link"
		}
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "img":
		return "img"
	case "form":
		return "form"
	case "input":
		parseInputType := strings.ToLower(strings.TrimSpace(readPropString(parseFiber, "type")))
		switch parseInputType {
		case "button", "submit", "reset":
			return "button"
		case "checkbox":
			return "checkbox"
		case "radio":
			return "radio"
		case "range":
			return "slider"
		case "email", "password", "search", "tel", "text", "url", "":
			return "textbox"
		}
	}
	return ""
}

// readFiberTag returns the host element tag for a fiber that is a host
// element (typeOf is a string other than "ROOT" and "TEXT_ELEMENT"). Returns
// an empty string for component fibers, text fibers, and the root.
func readFiberTag(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	parseTag, parseOk := parseFiber.typeOf.(string)
	if !parseOk {
		return ""
	}
	if parseTag == "ROOT" || parseTag == "TEXT_ELEMENT" {
		return ""
	}
	return parseTag
}

// readFiberText concatenates the text content of the fiber and its entire
// subtree in pre-order depth-first order, mirroring the testkit nodeText
// function. Text is normalised (trimmed, collapsed whitespace) before joining.
func readFiberText(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	parseParts := make([]string, 0, 4)
	readCollectFiberText(parseFiber, &parseParts)
	return strings.Join(parseParts, " ")
}

// readCollectFiberText appends non-empty text segments from parseFiber and its
// children into parseParts.
func readCollectFiberText(parseFiber *Fiber, parseParts *[]string) {
	if parseFiber == nil {
		return
	}
	if parseFiber.textContent != "" {
		parseText := strings.TrimSpace(parseFiber.textContent)
		if parseText != "" {
			*parseParts = append(*parseParts, parseText)
		}
	}
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		readCollectFiberText(parseChild, parseParts)
	}
}

// readPropString reads one string prop from a fiber's props map. It returns
// the empty string when the prop is absent, nil, or not a string.
func readPropString(parseFiber *Fiber, parsePropName string) string {
	if parseFiber == nil || parseFiber.props == nil {
		return ""
	}
	parseVal, parseOk := parseFiber.props[parsePropName]
	if !parseOk || parseVal == nil {
		return ""
	}
	parseStr, parseOk2 := parseVal.(string)
	if !parseOk2 {
		return ""
	}
	return parseStr
}
