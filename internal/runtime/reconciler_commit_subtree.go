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
	return isFastLaneCompactFiber(parseFiber) && !parseFiber.fineGrained &&
		len(parseFiber.eventCallbacks) == 0
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
	return true
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
	if _, isSVG := serializedMountSVGTags[strings.ToLower(parseTag)]; isSVG {
		return false
	}
	if !isFastLaneCompactFiber(parseFiber) || parseFiber.fineGrained ||
		len(parseFiber.eventCallbacks) != 0 || !IsDOMNodeNull(parseFiber.dom) ||
		parseFiber.hydration != nil || parseFiber.effectTag != effectTagPlacement {
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
		for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
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
