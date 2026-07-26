package runtime

import (
	"maps"
	"strconv"
	"strings"
)

func GetCurrentFiber() *Fiber {
	return currentFiber
}

// Hook-ownership guard state. computeHookGoroutineID is a full runtime.Stack
// traceback (microseconds per call), so the guard verifies lazily:
// hookGuardVerifiedOwnerID records the owner goroutine id for which some hook
// call already proved the calling goroutine matches. It persists across
// SetCurrentFiber(nil)/re-arm cycles for the same owner id (goroutine ids are
// never reused within a process), so steady-state render passes on one
// goroutine verify once instead of once per component. These are package
// globals like currentFiber itself; the reconciler is single-goroutine per
// render pass by construction.
var (
	hookGuardVerifiedOwnerID uint64
	hookGuardCheckCounter    uint32
)

// SetCurrentFiber sets the current fiber (used during component rendering)
func SetCurrentFiber(parseFiber *Fiber) {
	currentFiber = parseFiber
	if parseFiber == nil || !hookThreadingGuardEnabled {
		currentFiberOwnerGoroutineID = 0
		return
	}
	currentFiberOwnerGoroutineID = computeHookGoroutineID()
}

// renderPassOwnerID returns the goroutine id that owns the current component
// render: the work loop pass id (captured lazily on the pass's first component
// render), or a freshly computed id when no pass is active. It must only be
// called from the render's own synchronous call stack (renderFunctionComponent,
// strictPreviewRender), which is what makes the cached id exact.
func (parseRt *Runtime) renderPassOwnerID() uint64 {
	if !hookThreadingGuardEnabled {
		return 0
	}
	if parseRt == nil || !parseRt.renderPassActive {
		return computeHookGoroutineID()
	}
	if parseRt.renderPassOwnerGoroutineID == 0 {
		parseRt.renderPassOwnerGoroutineID = computeHookGoroutineID()
	}
	return parseRt.renderPassOwnerGoroutineID
}

// setCurrentFiberOwned installs a fiber whose owner goroutine id was already
// computed by the enclosing render pass (see Runtime.workLoop). The owner id
// is exact because the reconciler calls this on the same synchronous call
// stack that captured it; passing 0 falls back to computing the id.
func setCurrentFiberOwned(parseFiber *Fiber, parseOwnerID uint64) {
	if parseOwnerID == 0 || parseFiber == nil {
		SetCurrentFiber(parseFiber)
		return
	}
	currentFiber = parseFiber
	if !hookThreadingGuardEnabled {
		currentFiberOwnerGoroutineID = 0
		return
	}
	currentFiberOwnerGoroutineID = parseOwnerID
}

// runtimeForFiber returns the owning runtime for a fiber subtree, falling back
// to the global runtime for legacy tests that install a bare current fiber.
// ResolveRuntime returns the runtime that owns the component currently
// rendering, falling back to the global runtime outside a render (v5 P2.6).
//
// Packages that reach for GetGlobalRuntime to read or write atoms should call
// this instead. During a render it resolves to the runtime that actually owns
// the tree, so a second Runtime in the same process keeps its own atom state —
// which is what P3.7 requires, since projections are published into the atom
// registry and would otherwise all land in whichever runtime happened to be
// global.
func ResolveRuntime() *Runtime {
	return runtimeForFiber(currentFiber)
}

func runtimeForFiber(parseFiber *Fiber) *Runtime {
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.parent {
		if parseCursor.ownerRuntime != nil {
			return parseCursor.ownerRuntime
		}
	}
	if parseFiber != nil && parseFiber.alternate != nil {
		for parseCursor := parseFiber.alternate; parseCursor != nil; parseCursor = parseCursor.parent {
			if parseCursor.ownerRuntime != nil {
				return parseCursor.ownerRuntime
			}
		}
	}
	return GetGlobalRuntime()
}

// requireCurrentHookFiber returns the current render fiber or panics with the
// hook-specific development diagnostic.
func requireCurrentHookFiber(parseName string) *Fiber {
	parseFiber := GetCurrentFiber()
	if parseFiber == nil {
		panic(actionableHookUsagePanic(parseName))
	}
	if !isCurrentHookGoroutineOwner() {
		reportHookThreadingViolation(parseName, parseFiber)
		panic(actionableHookThreadingPanic(parseName, parseFiber))
	}
	return parseFiber
}

// isCurrentHookGoroutineOwner reports whether a hook call is running on the
// goroutine that claimed the current render fiber.
//
// The definitive check costs a full runtime.Stack traceback, so it runs on the
// first hook call under a not-yet-verified owner id and is then re-sampled
// every 64th call; verified steady-state hook calls take the cheap path. A
// hook call from a goroutine other than the one that armed the fiber is
// always caught while that owner id is unverified (goroutine ids are unique
// for the life of the process, so a stale verified id never matches a new
// owner); a goroutine spawned mid-render after verification is caught by the
// periodic re-check instead of on its first call.
func isCurrentHookGoroutineOwner() bool {
	if !hookThreadingGuardEnabled || currentFiberOwnerGoroutineID == 0 {
		return true
	}
	hookGuardCheckCounter++
	// Resample interval trade-off: a rogue goroutine calling hooks after the
	// owner was verified is caught within this many hook calls. 4096 (was 64)
	// keeps detection while making hook-dense renders affordable — a 2400-hook
	// pass paid ~37 runtime.Stack tracebacks per render at 64 (~65% of total
	// hooks-scenario CPU, measured); at 4096 it pays at most one.
	if hookGuardVerifiedOwnerID == currentFiberOwnerGoroutineID && hookGuardCheckCounter&4095 != 0 {
		return true
	}
	parseCurrentID := computeHookGoroutineID()
	if parseCurrentID == 0 || parseCurrentID == currentFiberOwnerGoroutineID {
		hookGuardVerifiedOwnerID = currentFiberOwnerGoroutineID
		return true
	}
	return false
}

// reportHookThreadingViolation records a structured diagnostic before the hook
// panic is raised. The panic report includes its own stack detail.
func reportHookThreadingViolation(parseName string, parseFiber *Fiber) {
	parseFields := map[string]string{
		"hook":            strings.TrimSpace(parseName),
		"renderGoroutine": strconv.FormatUint(currentFiberOwnerGoroutineID, 10),
		"callGoroutine":   strconv.FormatUint(computeHookGoroutineID(), 10),
	}
	reportDiagnosticWithContextDetails(
		"runtime",
		DiagnosticError,
		hookThreadingViolationMessage(parseName),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		"",
		"hook state is render-goroutine-owned; continuing would corrupt component state",
		parseFields,
	)
}

// IsCurrentFiberTransitionUpdate reports whether the current fiber render originated from deferred transition work.
func IsCurrentFiberTransitionUpdate() bool {
	for parseFiber := GetCurrentFiber(); parseFiber != nil; parseFiber = parseFiber.parent {
		if strings.HasPrefix(parseFiber.updateOrigin, "transition") {
			return true
		}
	}
	return false
}

// CreateElement creates a new virtual DOM element.
func CreateElement(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	return buildElement(parseTyp, cloneElementProps(parseProps), parseChildren...)
}

// CreateElementOwned creates a new virtual DOM element and takes ownership of the provided props map.
func CreateElementOwned(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	return buildElement(parseTyp, parseProps, parseChildren...)
}

// CreateElementCompactHostOwned creates one typed fast-lane host element from a
// reconciliation key and a caller-normalized, deterministic compact
// string-attribute view. Fast-lane elements carry no props map at all; cold
// readers materialize one on demand (see ensureElementProps).
func CreateElementCompactHostOwned(parseTag string, parseKey string, parseAttrs []HostAttr, parseChildren ...any) *Element {
	if parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		parseElem := buildElement(parseTag, nil, parseChildren...)
		parseElem.Key = parseKey
		return parseElem
	}
	parseElem := buildElementWithHostProps(parseTag, nil, nil, parseAttrs, true, true, parseChildren...)
	parseElem.Key = parseKey
	return parseElem
}

// CreateElementCompactHostOwnedText creates one typed fast-lane host element
// whose only child is plain text, skipping child-slice normalization entirely.
func CreateElementCompactHostOwnedText(parseTag string, parseKey string, parseAttrs []HostAttr, parseText string) *Element {
	return &Element{
		Type:               parseTag,
		Children:           emptyChildren,
		Key:                parseKey,
		TextContent:        parseText,
		getHostAttrs:       parseAttrs,
		isCompactHostProps: true,
		hasDirectText:      true,
	}
}

// PlainTextContent reports the text carried by one plain text node (as built
// by ui.Text / html.Text â€” no props, no key), so construction sugar can route
// single text children through the direct-text fast path without allocating a
// child slice.
func PlainTextContent(parseElem *Element) (string, bool) {
	if parseElem == nil || parseElem.hasDirectText || len(parseElem.Props) != 0 {
		return "", false
	}
	if parseTag, parseOk := parseElem.Type.(string); parseOk && parseTag == "TEXT_ELEMENT" {
		return parseElem.TextContent, true
	}
	return "", false
}

// buildElementHostProps creates one host-only props map and optional compact string attrs for one public element payload.
func buildElementHostProps(parseTyp any, parseProps map[string]any) (map[string]any, []HostAttr, bool) {
	parseTag, parseOk := parseTyp.(string)
	if !parseOk || parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return nil, nil, false
	}
	if len(parseProps) == 0 {
		return nil, nil, true
	}

	// Host fibers alias the element's props map directly instead of building a
	// separate host-only copy: every entry point clones or owns the map before
	// reaching here, and the DOM differ skips propKindSkip entries (including
	// "children"), so the copy only added one map allocation per host element
	// per render â€” the single largest allocation site in component updates.
	getHostAttrs := make([]HostAttr, 0, len(parseProps))
	isCompactHostProps := true
	for parseName, parseValue := range parseProps {
		if parseName == "children" {
			continue
		}
		if parseName == "key" || parseValue == nil {
			continue
		}
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseAttrName := parseMeta.attrName
		if parseAttrName == "" {
			parseAttrName = parseName
		}
		switch parseMeta.kind {
		case propKindSpecialProperty:
			isCompactHostProps = false
		case propKindStyle:
			parseTextValue, parseTextOk := parseValue.(string)
			if !parseTextOk {
				isCompactHostProps = false
				continue
			}
			getHostAttrs = append(getHostAttrs, HostAttr{Name: parseAttrName, Value: parseTextValue})
		default:
			parseTextValue, parseTextOk := parseValue.(string)
			if !parseTextOk {
				isCompactHostProps = false
				continue
			}
			getHostAttrs = append(getHostAttrs, HostAttr{Name: parseAttrName, Value: parseTextValue})
		}
	}
	if !isCompactHostProps {
		getHostAttrs = nil
	}
	return parseProps, getHostAttrs, isCompactHostProps
}

// cloneElementProps clones one props map so callers can safely retain and reuse their original input.
func cloneElementProps(parseProps map[string]any) map[string]any {
	if len(parseProps) == 0 {
		return nil
	}

	getProps := make(map[string]any, len(parseProps)+1)
	maps.Copy(getProps, parseProps)
	return getProps
}

// canStoreElementDirectText reports whether one host element can carry its only
// text child directly on the host fiber. Both raw string children and plain
// TEXT_ELEMENT children (as produced by ui.Text / html.Text â€” no props, no key)
// qualify; carrying the text on the host fiber skips one fiber and one DOM
// text-node round trip per text leaf.
func canStoreElementDirectText(parseTyp any, parseChildren []any) (bool, string) {
	if len(parseChildren) != 1 {
		return false, ""
	}
	parseTag, parseOk := parseTyp.(string)
	if !parseOk || parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return false, ""
	}
	switch parseChild := parseChildren[0].(type) {
	case string:
		return true, parseChild
	case *Element:
		if parseChild != nil && !parseChild.hasDirectText && len(parseChild.Props) == 0 {
			if parseChildTag, parseTagOk := parseChild.Type.(string); parseTagOk && parseChildTag == "TEXT_ELEMENT" {
				return true, parseChild.TextContent
			}
		}
	}
	return false, ""
}

// DemoteDirectTextChild converts an element that carries its only text child
// directly on the host (the direct-text fast path) back to an explicit
// TEXT_ELEMENT child. Post-creation child appenders (html.WithChildren) must
// call this first: the direct-text representation renders TextContent and
// ignores structural children, so appending to Children alone would drop the
// new nodes.
func DemoteDirectTextChild(parseElem *Element) {
	if parseElem == nil || !parseElem.hasDirectText {
		return
	}
	parseElem.hasDirectText = false
	parseChildren := []any{&Element{
		Type:        "TEXT_ELEMENT",
		TextContent: parseElem.TextContent,
		Children:    emptyChildren,
	}}
	parseElem.TextContent = ""
	parseElem.Children = parseChildren
	if parseElem.Props == nil {
		if parseElem.isCompactHostProps {
			// Materialize the full attribute view: a bare {"children": ...}
			// map would make the SSR serializers treat the element as
			// map-built and silently drop every compact attribute.
			parseElem.Props = fastLanePropsView(parseElem.getHostAttrs, parseElem.Key, nil, false)
		} else {
			parseElem.Props = make(map[string]any, 1)
		}
	}
	parseElem.Props["children"] = parseChildren
}

// getElementChildren returns one element's structural children while tolerating legacy props-backed child storage.
func getElementChildren(parseElem *Element) []any {
	if parseElem == nil {
		return nil
	}
	if parseElem.hasDirectText {
		if parseElem.Children != nil {
			return parseElem.Children
		}
		return emptyChildren
	}
	if parseElem.Children != nil {
		if len(parseElem.Children) == 0 {
			if parseChildren, parseOk := parseElem.Props["children"].([]any); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseElem.Children
	}
	if parseChildren, parseOk := parseElem.Props["children"].([]any); parseOk {
		return parseChildren
	}
	return nil
}

// getFiberChildren returns one fiber's structural child slice while tolerating legacy props-backed child storage.
func getFiberChildren(parseFiber *Fiber) []any {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.hasDirectText {
		if parseFiber.children != nil {
			return parseFiber.children
		}
		return emptyChildren
	}
	if parseFiber.children != nil {
		if len(parseFiber.children) == 0 {
			if parseChildren, parseOk := parseFiber.props["children"].([]any); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseFiber.children
	}
	if parseChildren, parseOk := parseFiber.props["children"].([]any); parseOk {
		return parseChildren
	}
	return nil
}

// getElementFiberProps resolves one element's internal working props bag.
func getElementFiberProps(parseElem *Element) map[string]any {
	if parseElem == nil {
		return nil
	}
	if parseElem.getHostProps != nil || parseElem.isCompactHostProps {
		return parseElem.getHostProps
	}
	return parseElem.Props
}

// isFastLaneCompactFiber reports whether a fiber carries typed fast-lane host
// state: a deterministic compact attribute slice and no props map at all.
func isFastLaneCompactFiber(parseFiber *Fiber) bool {
	return parseFiber != nil && parseFiber.isCompactHostProps && parseFiber.props == nil
}

// hostAttrsEqual compares two deterministic compact attribute slices
// positionally; fast-lane construction guarantees a stable order per payload.
func hostAttrsEqual(parseA, parseB []HostAttr) bool {
	if len(parseA) != len(parseB) {
		return false
	}
	for parseIndex := range parseA {
		if parseA[parseIndex] != parseB[parseIndex] {
			return false
		}
	}
	return true
}

// compactAttrPropName maps one compact attribute name back to its legacy
// props-map key; only "for" differs from its prop spelling.
func compactAttrPropName(parseName string) string {
	if parseName == "for" {
		return "htmlFor"
	}
	return parseName
}

// fastLanePropsView materializes a legacy props-map view from typed fast-lane
// host state for cold readers (inspection, diagnostics, mixed-shape updates).
// Hot paths never call this.
func fastLanePropsView(parseAttrs []HostAttr, parseKey string, parseChildren []any, hasDirectText bool) map[string]any {
	parseProps := make(map[string]any, len(parseAttrs)+2)
	for _, parseAttr := range parseAttrs {
		parseProps[compactAttrPropName(parseAttr.Name)] = parseAttr.Value
	}
	if parseKey != "" {
		parseProps["key"] = parseKey
	}
	if !hasDirectText && len(parseChildren) != 0 {
		parseProps["children"] = parseChildren
	}
	return parseProps
}

// fiberPropsView returns a props-map view of one fiber, materializing a
// transient map for typed fast-lane fibers so map-shaped consumers keep
// working; the view is never cached on the fiber.
func fiberPropsView(parseFiber *Fiber) map[string]any {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.props != nil || !parseFiber.isCompactHostProps {
		return parseFiber.props
	}
	return fastLanePropsView(parseFiber.getHostAttrs, parseFiber.key, parseFiber.children, parseFiber.hasDirectText)
}

// EnsureElementProps materializes and caches a legacy props-map view on one
// typed fast-lane element. Public tooling that reads Element.Props directly
// should call this first; elements built through the map lane are returned
// unchanged.
func EnsureElementProps(parseElem *Element) map[string]any {
	if parseElem == nil {
		return nil
	}
	if parseElem.Props != nil || !parseElem.isCompactHostProps {
		return parseElem.Props
	}
	parseElem.Props = fastLanePropsView(parseElem.getHostAttrs, parseElem.Key, parseElem.Children, parseElem.hasDirectText)
	return parseElem.Props
}

// RefreshElementHostProps rebuilds one element's cached host-prop view after
// post-creation prop mutation. Typed fast-lane elements materialize their
// props map first, so a refresh never drops attributes that only lived in the
// compact slice.
func RefreshElementHostProps(parseElem *Element) {
	if parseElem == nil {
		return
	}
	EnsureElementProps(parseElem)
	parseElem.getHostProps, parseElem.getHostAttrs, parseElem.isCompactHostProps = buildElementHostProps(parseElem.Type, parseElem.Props)
}

// buildElement builds one virtual DOM element and stores the normalized children slice on the props map.
func buildElement(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	getHostProps, getHostAttrs, isCompactHostProps := buildElementHostProps(parseTyp, parseProps)
	return buildElementWithHostProps(parseTyp, parseProps, getHostProps, getHostAttrs, isCompactHostProps, false, parseChildren...)
}

// buildElementWithHostProps builds one virtual DOM element from an
// already-normalized host-prop view. isMapFree marks the typed fast lane:
// those elements never receive a props map at construction time (cold readers
// materialize one through EnsureElementProps).
func buildElementWithHostProps(parseTyp any, parseProps map[string]any, getHostProps map[string]any, getHostAttrs []HostAttr, isCompactHostProps bool, isMapFree bool, parseChildren ...any) *Element {
	if len(parseChildren) == 0 {
		parseChildren = emptyChildren
	}

	if isParseDirectText, parseDirectText := canStoreElementDirectText(parseTyp, parseChildren); isParseDirectText {
		// Typed fast-lane elements stay map-free; their children live on the
		// Children/text fields only.
		if parseProps == nil && !isMapFree {
			parseProps = make(map[string]any, 1)
		}
		if parseProps != nil {
			parseProps["children"] = parseChildren
		}
		return &Element{
			Type:               parseTyp,
			Props:              parseProps,
			Children:           emptyChildren,
			TextContent:        parseDirectText,
			getHostProps:       getHostProps,
			getHostAttrs:       getHostAttrs,
			isCompactHostProps: isCompactHostProps,
			hasDirectText:      true,
		}
	}

	// Normalize string children once so downstream reconciliation sees only
	// Elements.  All text elements for one parent share a single backing array
	// so N string children cost one allocation instead of N.
	parseTextCount := 0
	for _, parseChild := range parseChildren {
		if _, hasParseText := parseChild.(string); hasParseText {
			parseTextCount++
		}
	}
	if parseTextCount > 0 {
		parseTextElems := make([]Element, parseTextCount)
		parseTextIdx := 0
		for parseIndex, parseChild := range parseChildren {
			if parseText, hasParseText := parseChild.(string); hasParseText {
				parseTextElems[parseTextIdx] = Element{
					Type:        "TEXT_ELEMENT",
					TextContent: parseText,
					Children:    emptyChildren,
				}
				parseChildren[parseIndex] = &parseTextElems[parseTextIdx]
				parseTextIdx++
			}
		}
	}

	if parseProps == nil && !isMapFree {
		parseProps = make(map[string]any, 1)
	}
	if parseProps != nil {
		parseProps["children"] = parseChildren
	}

	return &Element{
		Type:               parseTyp,
		Props:              parseProps,
		Children:           parseChildren,
		getHostProps:       getHostProps,
		getHostAttrs:       getHostAttrs,
		isCompactHostProps: isCompactHostProps,
	}
}

// flattenFragments is an internal reconciler helper.
func flattenFragments(parseElements []any) ([]any, bool) {
	isParseNeedsFlatten := false
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			continue
		}
		if parseT, parseOk2 := parseElem.Type.(string); parseOk2 && parseT == "FRAGMENT" {
			isParseNeedsFlatten = true
			break
		}
	}

	if !isParseNeedsFlatten {
		return parseElements, false
	}

	parseFlattened := slicePool.get()
	for _, parseElement2 := range parseElements {
		parseElem2, parseOk3 := parseElement2.(*Element)
		if !parseOk3 {
			if parseElement2 != nil {
				parseFlattened = append(parseFlattened, parseElement2)
			}
			continue
		}
		if parseElem2 == nil {
			continue
		}
		if parseT2, parseOk4 := parseElem2.Type.(string); parseOk4 && parseT2 == "FRAGMENT" {
			if parseChildren := getElementChildren(parseElem2); parseChildren != nil {
				parseRes, parseAllocated := flattenFragments(parseChildren)
				parseFlattened = append(parseFlattened, parseRes...)
				if parseAllocated {
					slicePool.clear(parseRes)
				}
			}
			continue
		}
		parseFlattened = append(parseFlattened, parseElem2)
	}

	return parseFlattened, true
}

// cloneChildFibers clones the child fibers from the alternate to the current fiber
// This is used when skipping reconciliation for non-dirty fibers
func (parseRt *Runtime) cloneChildFibers(parseParent *Fiber) {
	if parseParent.alternate == nil || parseParent.alternate.child == nil {
		return
	}

	var parsePrevSibling *Fiber
	parseOldFiber := parseParent.alternate.child

	for parseOldFiber != nil {
		parseEffectTag := effectTagNone
		if parseOldFiber.dirty || parseOldFiber.needsUpdate {
			parseEffectTag = effectTagUpdate
		}
		parseNewFiber := acquireWorkInProgress(parseOldFiber)
		*parseNewFiber = Fiber{
			typeOf:   parseOldFiber.typeOf,
			props:    parseOldFiber.props,
			children: parseOldFiber.children,
			// key must survive the bailout clone: fast-lane fibers carry their
			// reconciliation key ONLY here (props is nil), and the keyed
			// reconciler treats key==""+props==nil as unkeyed — dropping it
			// destroyed row identity on the next keyed update after a bailout.
			key:                 parseOldFiber.key,
			portalUnresolved:    parseOldFiber.portalUnresolved,
			getHostAttrs:        parseOldFiber.getHostAttrs,
			textContent:         parseOldFiber.textContent,
			dom:                 parseOldFiber.dom,
			parent:              parseParent,
			alternate:           parseOldFiber,
			effectTag:           parseEffectTag,
			dirty:               parseOldFiber.dirty,
			subtreeDirty:        parseOldFiber.subtreeDirty,
			needsUpdate:         parseOldFiber.needsUpdate,
			needsChildReconcile: parseOldFiber.needsChildReconcile,
			hooks:               parseOldFiber.hooks, // Share hooks for non-updated components
			eventCallbacks:      parseOldFiber.eventCallbacks,
			contextValues:       parseOldFiber.contextValues,
			reactiveAtomID:      parseOldFiber.reactiveAtomID,
			reactiveSourceIDs:   parseOldFiber.reactiveSourceIDs,
			fineGrained:         parseOldFiber.fineGrained,
			hasDirectText:       parseOldFiber.hasDirectText,
			isCompactHostProps:  parseOldFiber.isCompactHostProps,
			updateOrigin:        parseOldFiber.updateOrigin,
			// Same reason as buildUpdatedFiber: the bailout clone is the OTHER
			// way a marked fiber reaches the work loop, so dropping the lane
			// here defeats P2.2 just as completely. The suspension fields travel
			// for the same reason — see the goroutine-per-render note there.
			updateLane:      parseOldFiber.updateLane,
			asyncSuspension: parseOldFiber.asyncSuspension,
			asyncWait:       parseOldFiber.asyncWait,
			ownerRuntime:    parseRt,
		}
		if parseNewFiber.hooks != nil {
			parseNewFiber.hooks.owner = parseNewFiber
		}
		ensureFineGrainedTwinLink(parseOldFiber, parseNewFiber)
		parseRt.handleClonedFiberSubscriptionMove(parseOldFiber, parseNewFiber)

		if parsePrevSibling == nil {
			parseParent.child = parseNewFiber
		} else {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
		parseOldFiber = parseOldFiber.sibling
	}
}

// reuseFiberChildSubtree relinks one committed child chain under the current fiber without cloning descendants.
func (parseRt *Runtime) reuseFiberChildSubtree(parseParent *Fiber) {
	if parseParent == nil || parseParent.alternate == nil {
		return
	}
	parseParent.child = parseParent.alternate.child
	if parseParent.child != nil {
		parseRt.sanitizeFiberSubtree(parseParent.child, parseParent)
	}
}

// sanitizeFiberSubtree relinks one reused committed subtree and clears stale
// work flags before commit traversal. A node that is already fully clean —
// correct parent link, no pending flags, hooks owned by itself — can only
// have clean descendants (scheduler dirtying bubbles subtreeDirty up through
// it, and any render below went through a dirty ancestor), so the walk stops
// descending there. The first bailout after a commit pays one full sweep;
// every later bailout of the same subtree touches only the shallow fringe.
func (parseRt *Runtime) sanitizeFiberSubtree(parseFiber *Fiber, parseParent *Fiber) {
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.sibling {
		if parseCurrent.parent == parseParent &&
			parseCurrent.effectTag == effectTagNone &&
			!parseCurrent.dirty && !parseCurrent.subtreeDirty &&
			!parseCurrent.needsUpdate && !parseCurrent.needsChildReconcile && !parseCurrent.needsChildOrder &&
			(parseCurrent.hooks == nil || parseCurrent.hooks.owner == parseCurrent) {
			continue
		}
		parseCurrent.parent = parseParent
		parseCurrent.effectTag = effectTagNone
		parseCurrent.dirty = false
		parseCurrent.subtreeDirty = false
		parseCurrent.needsUpdate = false
		parseCurrent.needsChildReconcile = false
		parseCurrent.needsChildOrder = false
		if parseCurrent.hooks != nil {
			parseCurrent.hooks.owner = parseCurrent
		}
		if parseCurrent.child != nil {
			parseRt.sanitizeFiberSubtree(parseCurrent.child, parseCurrent)
		}
	}
}

// restoreCommittedTreeLinks re-anchors the committed tree after a render pass is
// abandoned, so every fiber in it resolves back to currentRoot again.
//
// The bailout path SHARES fiber objects between the two trees: reuseFiberChildSubtree
// hands the work-in-progress parent the committed parent's own child chain, and
// sanitizeFiberSubtree then repoints those children's .parent at the WIP fiber.
// One object cannot have two parents, and the design accepts that because the WIP
// tree becomes the committed tree a moment later. Abandon the pass instead and the
// trade is never paid: the committed tree's descendants are left pointing into a
// fiber that was thrown away.
//
// That is not cosmetic. isFiberInCurrentTree decides whether an update is
// deliverable by walking .parent up to the root and comparing it with currentRoot,
// and resolveOwnedFiberTarget / resolveSubscribedFiberTarget return nil when it
// fails — so after one abandoned pass a component's setState reaches nothing, with
// no diagnostic and no crash. Measured before this fix: three of four fibers
// orphaned, and a state write afterwards never reached the DOM.
//
// O(tree) on a path that only runs when a pass is discarded, which is the right
// place to spend it. No short-circuit on an already-correct parent link: the
// abandoned pass may have re-anchored a fiber at any depth, so a matching link
// near the root says nothing about the subtree beneath it.
//
// What this does NOT restore: sanitizeFiberSubtree also clears dirty/needsUpdate
// on the shared fibers, so work marked before the abandoned pass is lost with it.
// Recovering that needs the flags to be captured before the pass, which is a
// larger change; re-anchoring at least stops the tree from being permanently
// undeliverable, which is the difference between one lost update and every
// future one.
func restoreCommittedTreeLinks(parseRoot *Fiber) {
	if parseRoot == nil {
		return
	}
	for parseChild := parseRoot.child; parseChild != nil; parseChild = parseChild.sibling {
		parseChild.parent = parseRoot
		restoreCommittedTreeLinks(parseChild)
	}
}

// handleClonedFiberSubscriptionMove moves atom subscriptions from one cloned fiber to its new current fiber.
func (parseRt *Runtime) handleClonedFiberSubscriptionMove(parseOldFiber *Fiber, parseNewFiber *Fiber) {
	if parseRt == nil || parseRt.atomRegistry == nil || parseOldFiber == nil || parseNewFiber == nil || parseOldFiber == parseNewFiber {
		return
	}
	getAtomIDs := buildClonedFiberSubscriptionAtomIDs(parseNewFiber)
	if len(getAtomIDs) == 0 {
		return
	}
	if parseRt.hydrating {
		for _, parseAtomID := range getAtomIDs {
			if parseAtomID == "" {
				continue
			}
			parseRt.queueHydrationSubscription(parseAtomID, parseOldFiber, false)
			parseRt.queueHydrationSubscription(parseAtomID, parseNewFiber, true)
		}
		return
	}
	parseRt.atomRegistry.MoveSubscriptions(getAtomIDs, parseOldFiber, parseNewFiber)
}

// buildClonedFiberSubscriptionAtomIDs returns one deduplicated atom ID list for cloned-fiber subscription transfer.
func buildClonedFiberSubscriptionAtomIDs(parseFiber *Fiber) []string {
	if parseFiber == nil {
		return nil
	}
	getCapacityHint := len(parseFiber.reactiveSourceIDs)
	if parseFiber.hooks != nil {
		getCapacityHint += len(parseFiber.hooks.atoms)
	}
	if getCapacityHint == 0 {
		return nil
	}
	getAtomIDs := make([]string, 0, getCapacityHint)
	hasAtomIDSeen := make(map[string]bool, getCapacityHint)
	storeAtomIDs := func(parseSourceIDs []string) {
		for _, parseAtomID := range parseSourceIDs {
			if parseAtomID == "" || hasAtomIDSeen[parseAtomID] {
				continue
			}
			hasAtomIDSeen[parseAtomID] = true
			getAtomIDs = append(getAtomIDs, parseAtomID)
		}
	}
	if parseFiber.hooks != nil && len(parseFiber.hooks.atoms) > 0 {
		storeAtomIDs(parseFiber.hooks.atoms)
	}
	if len(parseFiber.reactiveSourceIDs) > 0 {
		storeAtomIDs(parseFiber.reactiveSourceIDs)
	}
	return getAtomIDs
}
