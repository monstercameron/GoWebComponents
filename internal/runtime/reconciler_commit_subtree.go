package runtime

import (
	"html"
	"strings"
)

// htmlSubtreeDOMAdapter is the optional capability for mounting one serialized
// HTML subtree in a single bridge call (browser adapter only; the mock and
// node environments fall back to per-node creation).
type htmlSubtreeDOMAdapter interface {
	CreateHTMLSubtree(parseHTML string) DOMNode
}

// serializedMountMinHosts gates the serialized strategy to subtrees where one
// parse beats per-node template creation; tiny mounts stay on the per-node
// path, which is already one bridge call per node.
const serializedMountMinHosts = 3

// serializedMountSVGTags mirrors the adapter-side SVG routing: template
// innerHTML parses in the HTML namespace, so SVG subtrees must keep the
// per-node createElementNS path.
var serializedMountSVGTags = map[string]struct{}{
	"svg": {}, "g": {}, "defs": {}, "use": {}, "symbol": {}, "marker": {},
	"path": {}, "rect": {}, "circle": {}, "ellipse": {}, "line": {},
	"polyline": {}, "polygon": {}, "tspan": {}, "clippath": {}, "mask": {},
	"pattern": {}, "image": {}, "foreignobject": {},
	"lineargradient": {}, "radialgradient": {}, "stop": {},
}

// shouldDeferHostDomToCommit reports whether one render-phase host fiber may
// leave its DOM null so commit can attempt a serialized whole-subtree mount.
// Deferral is always safe: the commit placement branch creates per-node DOM
// for any fiber that still has none.
//
// NOTE(perf A/B 2026-07-04): measured against the per-node prepared+batched
// path in the Example 201 browser harness. End-to-end DOM-ready is within
// run noise; the phase probe shows diff+commit creation cost consistently
// 3-15% lower on every mount scenario, and a 200-row list mounts in one
// bridge call instead of one per node. Kept for the consistent direction
// and call-count scaling; do not expect large end-to-end wins from it.
func (parseRt *Runtime) shouldDeferHostDomToCommit(parseFiber *Fiber) bool {
	if parseRt == nil || parseFiber == nil || parseFiber.effectTag != effectTagPlacement {
		return false
	}
	if _, parseOk := parseRt.domAdapter.(htmlSubtreeDOMAdapter); !parseOk {
		return false
	}
	// Plain text nodes defer too: a serialized parent inlines them (mixed
	// text+element children are the deep-tree shape), and any that end up
	// outside a serialized subtree are created by the commit placement
	// branch instead — same bridge call, later.
	if parseTag, parseOk := parseFiber.typeOf.(string); parseOk && parseTag == "TEXT_ELEMENT" {
		return true
	}
	return isSerializableHostFiber(parseFiber) && !parseFiber.fineGrained &&
		len(parseFiber.eventCallbacks) == 0
}

// isSerializableHostFiber reports whether one host fiber's DOM state is fully
// described by getHostAttrs, so a serialized mount produces exactly what the
// per-node path would have produced.
//
// The typed fast lane (props == nil) qualifies by construction. The MAP lane
// qualifies too, and that is the case that matters: isCompactHostProps already
// means every prop resolved to a plain string attribute captured in
// getHostAttrs — a non-string value or a special property clears it. Gating on
// isFastLaneCompactFiber instead excluded every element built through
// html.Props{...}, which is nearly all real application markup and all of the
// Example 201 content render, so those subtrees never reached the one-call
// mount that already existed for them.
//
// Only two prop kinds are dropped without clearing isCompactHostProps:
// "children", which needs no attribute, and a DOM ref, which needs the node —
// a ref-bearing fiber must take the per-node path or its ref is never filled.
func isSerializableHostFiber(parseFiber *Fiber) bool {
	if parseFiber == nil || !parseFiber.isCompactHostProps {
		return false
	}
	if parseFiber.props == nil {
		return true
	}
	_, hasRef := parseFiber.props[DOMRefKey]
	return !hasRef
}

// serializedAttrsRoundTrip reports whether every attribute survives the SSR
// writer byte-for-byte.
//
// writeSSRCompactAttrs silently DROPS names it rejects and REWRITES url-bearing
// values. Both are correct for SSR and wrong here: the per-node mount path
// calls SetAttribute with the raw name and value, so a subtree containing
// either case would land in the DOM differently depending on which strategy
// commit happened to pick — and which it picks depends on host count and
// sibling grouping, not on anything the author wrote. Rejecting those subtrees
// keeps the two paths identical. Map-lane attribute names come from user maps
// (data-*, aria-*, spread props), so this is reachable input, not paranoia.
func serializedAttrsRoundTrip(parseAttrs []HostAttr) bool {
	for _, parseAttr := range parseAttrs {
		parseName := normalizeSSRAttrName(compactAttrPropName(parseAttr.Name))
		if !isValidSSRAttrName(parseName) {
			return false
		}
		if urlBearingSSRAttr(parseName) && sanitizeSSRURLValue(parseAttr.Value) != parseAttr.Value {
			return false
		}
	}
	return true
}

// isDeferredCommitHostFiber reports whether one null-dom placement fiber is a
// host that will materialize DOM during commit (deferred for the serialized
// mount attempt), so pre-commit walks like the append-batching count treat it
// as a DOM-bearing placement rather than a pass-through container.
func isDeferredCommitHostFiber(parseFiber *Fiber) bool {
	if parseFiber == nil || parseFiber.effectTag != effectTagPlacement || !IsDOMNodeNull(parseFiber.dom) {
		return false
	}
	parseTag, parseOk := parseFiber.typeOf.(string)
	return parseOk && parseTag != "FRAGMENT"
}

// tryCommitSerializedSubtree mounts one placement fiber's entire subtree from
// a single serialized HTML string when every node in it is a plain typed
// fast-lane host: one template parse plus a binding walk instead of one
// template parse per node. Returns false (with no side effects) when the
// adapter lacks the capability or any node is ineligible, in which case the
// caller uses the regular per-node path. On success the root fiber carries
// the mounted DOM (still to be appended and placement-processed by the
// caller) and every descendant is bound and cleared to effectTagNone.
func (parseRt *Runtime) tryCommitSerializedSubtree(parseFiber *Fiber) bool {
	if parseRt == nil || parseFiber == nil || parseFiber.child == nil {
		return false
	}
	parseSubtreeAdapter, parseOk := parseRt.domAdapter.(htmlSubtreeDOMAdapter)
	if !parseOk {
		return false
	}

	var parseBuilder strings.Builder
	parseHostCount := 0
	if !serializeMountSubtree(parseFiber, &parseBuilder, &parseHostCount) || parseHostCount < serializedMountMinHosts {
		return false
	}

	parseRoot := parseSubtreeAdapter.CreateHTMLSubtree(parseBuilder.String())
	if IsDOMNodeNull(parseRoot) {
		return false
	}
	parseRt.bindSerializedSubtree(parseFiber, parseRoot)
	parseRt.profiling.serializedMountRoots++
	return true
}

// SerializedMountRoots reports how many placement roots (single subtrees or
// sibling-run members) mounted through the serialized-HTML strategy — the
// observable signal that the fast mount paths are actually firing.
func (parseRt *Runtime) SerializedMountRoots() int {
	if parseRt == nil {
		return 0
	}
	return parseRt.profiling.serializedMountRoots
}

// htmlFragmentDOMAdapter is the optional capability for parsing SEVERAL
// serialized sibling subtrees in one call; the returned node is a container
// (template content / mock fragment) whose children are the parsed roots.
type htmlFragmentDOMAdapter interface {
	CreateHTMLFragment(parseHTML string) DOMNode
}

// prepareSerializedSiblingRuns pre-mounts maximal runs of consecutive
// eligible placement siblings from ONE combined HTML parse. Flat lists are
// the motivating shape: each row is a one-host subtree, too small for
// tryCommitSerializedSubtree, so a 200-row mount previously paid one
// template parse per row. Run members keep effectTagPlacement with their dom
// pre-bound — the normal commit recursion appends them in order through the
// existing batched append — while their descendants are bound and cleared
// exactly like the single-subtree path.
func (parseRt *Runtime) prepareSerializedSiblingRuns(parseParent *Fiber) {
	if parseRt == nil || parseParent == nil || parseParent.child == nil {
		return
	}
	parseFragmentAdapter, parseOk := parseRt.domAdapter.(htmlFragmentDOMAdapter)
	if !parseOk {
		return
	}

	var parseBuilder strings.Builder
	var parseRun []*Fiber
	parseHostCount := 0

	parseFlush := func() {
		if len(parseRun) >= 2 && parseHostCount >= serializedMountMinHosts && parseBuilder.Len() > 0 {
			parseFragment := parseFragmentAdapter.CreateHTMLFragment(parseBuilder.String())
			if !IsDOMNodeNull(parseFragment) {
				parseDomChild := parseRt.domAdapter.GetFirstChild(parseFragment)
				for _, parseMember := range parseRun {
					if IsDOMNodeNull(parseDomChild) {
						break
					}
					// Bind the member and its descendants; the member keeps
					// its placement tag so the commit walk appends it in
					// sibling order.
					parseRt.bindSerializedSubtree(parseMember, parseDomChild)
					parseMember.effectTag = effectTagPlacement
					parseRt.profiling.serializedMountRoots++
					parseDomChild = parseRt.domAdapter.GetNextSibling(parseDomChild)
				}
			}
		}
		parseBuilder.Reset()
		parseRun = parseRun[:0]
		parseHostCount = 0
	}

	for parseChild := parseParent.child; parseChild != nil; parseChild = parseChild.sibling {
		parsePreLen := parseBuilder.Len()
		parsePreHosts := parseHostCount
		if serializeMountSubtree(parseChild, &parseBuilder, &parseHostCount) {
			parseRun = append(parseRun, parseChild)
			continue
		}
		// Member ineligible: discard its partial serialization and close the
		// current run.
		parseTruncated := parseBuilder.String()[:parsePreLen]
		parseHostCount = parsePreHosts
		parseKept := parseTruncated
		parseBuilder.Reset()
		parseBuilder.WriteString(parseKept)
		parseFlush()
	}
	parseFlush()
}

// serializeMountSubtree writes one fiber subtree as HTML, reporting false as
// soon as any node falls outside the eligible shape: a mounted-from-scratch
// typed fast-lane host (no props map, no events, no refs) whose children are
// either direct text or exclusively more such hosts. Attribute serialization
// reuses the SSR compact writer for sanitization and escaping parity.
func serializeMountSubtree(parseFiber *Fiber, parseBuilder *strings.Builder, parseHostCount *int) bool {
	if parseFiber == nil {
		return false
	}
	parseTag, parseOk := parseFiber.typeOf.(string)
	if !parseOk || parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return false
	}
	// The tag is written raw into the parsed HTML string; anything outside the
	// conservative tag alphabet would break the positional bind zip (or, with
	// a hostile name, inject markup). Mirrors jsdom's CreatePreparedElement
	// guard, which this path bypasses.
	if !isSafeSerializedMountTag(parseTag) {
		return false
	}
	if _, isSVG := serializedMountSVGTags[strings.ToLower(parseTag)]; isSVG {
		return false
	}
	if !isSerializableHostFiber(parseFiber) || parseFiber.fineGrained ||
		len(parseFiber.eventCallbacks) != 0 || !IsDOMNodeNull(parseFiber.dom) ||
		parseFiber.hydration != nil || parseFiber.effectTag != effectTagPlacement {
		return false
	}
	if !serializedAttrsRoundTrip(parseFiber.getHostAttrs) {
		return false
	}

	*parseHostCount++
	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)
	writeSSRCompactAttrs(parseBuilder, parseFiber.getHostAttrs)
	parseBuilder.WriteByte('>')
	if isVoidElement(parseTag) {
		return parseFiber.child == nil && !parseFiber.hasDirectText
	}
	if parseFiber.hasDirectText {
		if parseFiber.child != nil {
			return false
		}
		parseBuilder.WriteString(html.EscapeString(parseFiber.textContent))
	} else {
		// Children may mix plain text nodes with nested hosts (e.g. a label
		// text plus a nested element per layer of a deep tree). Text children
		// serialize inline; the positional bind walk pairs them with the
		// parsed text nodes. Two guards keep the zip aligned: empty text
		// parses to NO node, and adjacent text runs merge into ONE node —
		// either case falls back to per-node mounting.
		wasTextChild := false
		for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
			if isSerializableTextChild(parseChild) {
				parseText := serializableTextChildValue(parseChild)
				if parseText == "" || wasTextChild {
					return false
				}
				parseBuilder.WriteString(html.EscapeString(parseText))
				wasTextChild = true
				continue
			}
			wasTextChild = false
			if !serializeMountSubtree(parseChild, parseBuilder, parseHostCount) {
				return false
			}
		}
	}
	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseTag)
	parseBuilder.WriteByte('>')
	return true
}

// isSafeSerializedMountTag reports whether one tag name can round-trip through
// a serialized HTML parse without escaping (letters, digits, hyphen).
func isSafeSerializedMountTag(parseTag string) bool {
	if parseTag == "" {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseTag); parseIndex++ {
		parseC := parseTag[parseIndex]
		switch {
		case parseC >= 'a' && parseC <= 'z':
		case parseC >= 'A' && parseC <= 'Z':
		case parseC >= '0' && parseC <= '9':
		case parseC == '-':
		default:
			return false
		}
	}
	return true
}

// isSerializableTextChild reports whether one child fiber is a plain
// mounted-from-scratch text node the serializer may inline.
func isSerializableTextChild(parseFiber *Fiber) bool {
	if parseFiber == nil || parseFiber.effectTag != effectTagPlacement ||
		!IsDOMNodeNull(parseFiber.dom) || parseFiber.hydration != nil || parseFiber.child != nil {
		return false
	}
	parseTag, parseOk := parseFiber.typeOf.(string)
	return parseOk && parseTag == "TEXT_ELEMENT"
}

// serializableTextChildValue mirrors createDom's TEXT_ELEMENT value lookup.
func serializableTextChildValue(parseFiber *Fiber) string {
	parseText := parseFiber.textContent
	if parseText == "" && parseFiber.props != nil {
		parseText, _ = parseFiber.props["nodeValue"].(string)
	}
	return parseText
}

// bindSerializedSubtree zips the fiber subtree against the freshly parsed DOM
// subtree — the serializer emitted exactly this structure, so fiber children
// and DOM children correspond positionally — assigning each fiber its node
// and clearing descendant placement tags so the commit walk does not re-append
// nodes that arrived with the parsed subtree.
func (parseRt *Runtime) bindSerializedSubtree(parseFiber *Fiber, parseDom DOMNode) {
	parseFiber.dom = parseDom
	if parseFiber.child == nil {
		return
	}
	parseDomChild := parseRt.domAdapter.GetFirstChild(parseDom)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		if IsDOMNodeNull(parseDomChild) {
			return
		}
		parseChild.effectTag = effectTagNone
		parseRt.bindSerializedSubtree(parseChild, parseDomChild)
		parseDomChild = parseRt.domAdapter.GetNextSibling(parseDomChild)
	}
}
