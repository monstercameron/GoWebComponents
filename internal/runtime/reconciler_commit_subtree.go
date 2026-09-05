package runtime

import "strings"

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

// serializedMountReshapedTags are tags whose CONTENT MODEL an HTML parser
// enforces by rewriting the tree, which makes them unsafe to mount from a
// serialized string.
//
// bindSerializedSubtree zips fiber children against parsed DOM children by
// POSITION, on the assumption that the parser returns exactly the structure the
// serializer emitted. A parser makes no such promise. It promises a CONFORMING
// tree, and it will insert, move, or drop nodes to get one. Three cases,
// measured against the per-node mount in
// TestSerializedMountSurvivesParserNormalization:
//
//	<table><tr>…      an implicit <tbody> is inserted, so every <tr> fiber binds
//	                  to the tbody and each later update writes the wrong node
//	<p><div>…         the <p> is closed early and the div promoted to a sibling;
//	                  the observed result duplicated the content
//	<select><div>…    the stray element is dropped and replaced by a text node
//
// None of these error. The DOM is simply not the one the component described,
// and every subsequent update compounds it.
//
// Verifying after the parse was the alternative and is rejected on cost: the
// bind walk already spends two bridge crossings per node, and reading each
// node's tag to check it would add a third, erasing the reason this path
// exists. Refusing the shapes costs nothing and cannot be wrong about the ones
// it names.
func isSerializedMountReshapedTag(parseTag string) bool {
	switch parseTag {
	case "table", "thead", "tbody", "tfoot", "tr", "td", "th", "caption", "colgroup", "col",
		"select", "optgroup", "option", "datalist", "html", "head", "body", "frameset", "frame", "template", "form":
		return true
	default:
		return false
	}
}

// serializedMountNonNestingTags cannot contain themselves at any depth.
//
// The parser implies an end tag for the open one and promotes the inner element
// to a sibling, so the fiber tree and the parsed tree disagree about depth from
// that point down. Found by sweeping parent/child combinations rather than by
// reading the spec — li>li and a>a both survived a hand-written blocklist that
// already covered tables, selects, and paragraphs, which is the argument for
// keeping that sweep.
//
// Depth matters: <a><div><a> is nested just as <a><a> is, so this is checked
// against every ancestor inside the serialized subtree rather than the parent.
// Ancestors OUTSIDE it are irrelevant — the fragment is parsed standalone, so
// an <a> already in the document cannot affect it.
func isSerializedMountNonNestingTag(parseTag string) bool {
	switch parseTag {
	case "a", "li", "dt", "dd", "button", "nobr", "p":
		return true
	default:
		return false
	}
}

// serializedMountSVGTags mirrors the adapter-side SVG routing: template
// innerHTML parses in the HTML namespace, so SVG subtrees must keep the
// per-node createElementNS path.
func isSerializedMountSVGTag(parseTag string) bool {
	switch parseTag {
	case "svg", "g", "defs", "use", "symbol", "marker", "path", "rect", "circle", "ellipse", "line",
		"polyline", "polygon", "tspan", "clippath", "mask", "pattern", "image", "foreignobject",
		"lineargradient", "radialgradient", "stop":
		return true
	default:
		return false
	}
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
	if parseFiber == nil || !parseFiber.isCompactHostProps || parseFiber.hasCompactSpecialProps {
		return false
	}
	if parseFiber.props == nil {
		return true
	}
	_, hasRef := parseFiber.props[DOMRefKey]
	return !hasRef
}

// writeSerializedAttrsRoundTrip writes compact attributes only when every one
// survives the SSR writer byte-for-byte.
//
// writeSSRCompactAttrs silently DROPS names it rejects and REWRITES url-bearing
// values. Both are correct for SSR and wrong here: the per-node mount path
// calls SetAttribute with the raw name and value, so a subtree containing
// either case would land in the DOM differently depending on which strategy
// commit happened to pick — and which it picks depends on host count and
// sibling grouping, not on anything the author wrote. Rejecting those subtrees
// keeps the two paths identical. Map-lane attribute names come from user maps
// (data-*, aria-*, spread props), so this is reachable input, not paranoia.
//
// Validation and emission deliberately share this pass. The serialized mount
// path used to validate here and then call writeSSRCompactAttrs, which sorted,
// normalized, and validated every name a second time. Large append runs spend
// enough time in this wasm loop for that duplicate safety check to dominate.
func writeSerializedAttrsRoundTrip(parseBuilder *strings.Builder, parseAttrs []HostAttr) bool {
	if len(parseAttrs) == 0 {
		return true
	}
	var parseStorage [16]HostAttr
	parsePairs := parseStorage[:0]
	for _, parseAttr := range parseAttrs {
		parsePairs = append(parsePairs, HostAttr{Name: compactAttrPropName(parseAttr.Name), Value: parseAttr.Value})
	}
	for parseIndex := 1; parseIndex < len(parsePairs); parseIndex++ {
		parsePair := parsePairs[parseIndex]
		parseSlot := parseIndex
		for parseSlot > 0 && parsePairs[parseSlot-1].Name > parsePair.Name {
			parsePairs[parseSlot] = parsePairs[parseSlot-1]
			parseSlot--
		}
		parsePairs[parseSlot] = parsePair
	}
	for parseIndex := range parsePairs {
		parseName := normalizeSSRAttrName(parsePairs[parseIndex].Name)
		if !isValidSSRAttrName(parseName) {
			return false
		}
		if urlBearingSSRAttr(parseName) && sanitizeSSRURLValue(parsePairs[parseIndex].Value) != parsePairs[parseIndex].Value {
			return false
		}
		parsePairs[parseIndex].Name = parseName
	}
	for _, parsePair := range parsePairs {
		parseBuilder.WriteByte(' ')
		parseBuilder.WriteString(parsePair.Name)
		parseBuilder.WriteString(`="`)
		writeSerializedEscapedString(parseBuilder, parsePair.Value)
		parseBuilder.WriteByte('"')
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
	parseRt.bindSerializedSubtreeRoot(parseFiber, parseRoot)
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

// htmlFragmentAppendDOMAdapter can parse and append a serialized sibling run
// without round-tripping every root handle through Go.
type htmlFragmentAppendDOMAdapter interface {
	AppendHTMLFragment(parseParent DOMNode, parseHTML string) bool
}

// prepareSerializedSiblingRuns pre-mounts maximal runs of consecutive
// eligible placement siblings from ONE combined HTML parse. Flat lists are
// the motivating shape: each row is a one-host subtree, too small for
// tryCommitSerializedSubtree, so a 200-row mount previously paid one
// template parse per row. Run members keep effectTagPlacement with their dom
// pre-bound — the normal commit recursion appends them in order through the
// existing batched append — while their descendants are bound and cleared
// exactly like the single-subtree path.
func (parseRt *Runtime) prepareSerializedSiblingRuns(parseParent *Fiber, parseDomParent DOMNode) {
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
	parseCanDirectAppend := true

	parseFlush := func(parseAllowDirectAppend bool) {
		if len(parseRun) >= 2 && parseHostCount >= serializedMountMinHosts && parseBuilder.Len() > 0 {
			if parseAllowDirectAppend && parseCanDirectAppend {
				if parseAppendAdapter, parseAppendOk := parseRt.domAdapter.(htmlFragmentAppendDOMAdapter); parseAppendOk &&
					!IsDOMNodeNull(parseDomParent) && parseAppendAdapter.AppendHTMLFragment(parseDomParent, parseBuilder.String()) {
					for _, parseMember := range parseRun {
						markSerializedRootUnbound(parseMember)
						parseRt.profiling.serializedMountRoots++
					}
					parseBuilder.Reset()
					parseRun = parseRun[:0]
					parseHostCount = 0
					return
				}
			}
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
					parseRt.bindSerializedSubtreeRoot(parseMember, parseDomChild)
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
		parseCanDirectAppend = false
		parseFlush(false)
	}
	parseFlush(true)
}

// markSerializedRootUnbound records that a directly appended serialized root
// and its descendants already exist in the DOM but do not yet have Go handles.
func markSerializedRootUnbound(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseFiber.dom = nil
	parseFiber.effectTag = effectTagNone
	parseFiber.serializedUnbound = true
	markSerializedDescendantsUnbound(parseFiber)
}

// serializeMountSubtree writes one fiber subtree as HTML, reporting false as
// soon as any node falls outside the eligible shape: a mounted-from-scratch
// typed fast-lane host (no props map, no events, no refs) whose children are
// either direct text or exclusively more such hosts. Attribute serialization
// reuses the SSR compact writer for sanitization and escaping parity.
func serializeMountSubtree(parseFiber *Fiber, parseBuilder *strings.Builder, parseHostCount *int) bool {
	return serializeMountSubtreeWithin(parseFiber, parseBuilder, parseHostCount, nil)
}

// serializeMountSubtreeWithin carries the enclosing tags so a non-nesting tag
// can be refused when it appears inside itself. parseAncestors holds only tags
// inside the serialized subtree; it starts empty at each serialization root.
func serializeMountSubtreeWithin(parseFiber *Fiber, parseBuilder *strings.Builder, parseHostCount *int, parseAncestors []string) bool {
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
	parseLowerTag := strings.ToLower(parseTag)
	if isSerializedMountSVGTag(parseLowerTag) {
		return false
	}
	if isSerializedMountReshapedTag(parseLowerTag) {
		return false
	}
	if isSerializedMountNonNestingTag(parseLowerTag) {
		for _, parseAncestor := range parseAncestors {
			if parseAncestor == parseLowerTag {
				return false
			}
		}
	}
	// A paragraph may only contain phrasing content. An element child closes the
	// <p> early and is promoted to a sibling, so the fiber tree and the parsed
	// tree stop agreeing about depth. Text-only paragraphs — the common case,
	// and the one the benchmark's cards use — are unaffected.
	if parseLowerTag == "p" && parseFiber.child != nil {
		for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
			if !isSerializableTextChild(parseChild) {
				return false
			}
		}
	}
	if !isSerializableHostFiber(parseFiber) || parseFiber.fineGrained ||
		len(parseFiber.eventCallbacks) != 0 || !IsDOMNodeNull(parseFiber.dom) ||
		parseFiber.hydration != nil || parseFiber.effectTag != effectTagPlacement {
		return false
	}
	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)
	if !writeSerializedAttrsRoundTrip(parseBuilder, parseFiber.getHostAttrs) {
		return false
	}
	*parseHostCount++
	parseBuilder.WriteByte('>')
	if isVoidElement(parseTag) {
		return parseFiber.child == nil && !parseFiber.hasDirectText
	}
	if parseFiber.hasDirectText {
		if parseFiber.child != nil {
			return false
		}
		writeSerializedEscapedString(parseBuilder, parseFiber.textContent)
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
				writeSerializedEscapedString(parseBuilder, parseText)
				wasTextChild = true
				continue
			}
			wasTextChild = false
			if !serializeMountSubtreeWithin(parseChild, parseBuilder, parseHostCount,
				append(parseAncestors, parseLowerTag)) {
				return false
			}
		}
	}
	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseTag)
	parseBuilder.WriteByte('>')
	return true
}

// writeSerializedEscapedString appends Go html.EscapeString-compatible output
// directly to the existing mount builder. The standard helper first counts
// replacements and then builds a second string; serialized mounts can emit the
// same five escapes in one pass and avoid that temporary entirely.
func writeSerializedEscapedString(parseBuilder *strings.Builder, parseValue string) {
	parseStart := 0
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		var parseEscape string
		switch parseValue[parseIndex] {
		case '&':
			parseEscape = "&amp;"
		case '\'':
			parseEscape = "&#39;"
		case '<':
			parseEscape = "&lt;"
		case '>':
			parseEscape = "&gt;"
		case '"':
			parseEscape = "&#34;"
		default:
			continue
		}
		parseBuilder.WriteString(parseValue[parseStart:parseIndex])
		parseBuilder.WriteString(parseEscape)
		parseStart = parseIndex + 1
	}
	parseBuilder.WriteString(parseValue[parseStart:])
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

// bindSerializedSubtreeRoot installs the parsed root handle and marks its
// descendants as already materialized but not yet bound into Go. The previous
// implementation eagerly zipped every descendant using GetFirstChild plus
// GetNextSibling, paying roughly two Go/JS crossings per node during mount.
// Reconciliation only needs those handles when a parent is subsequently
// visited, so bindSerializedChildLevel resolves them one level at a time.
func (parseRt *Runtime) bindSerializedSubtreeRoot(parseFiber *Fiber, parseDom DOMNode) {
	parseFiber.dom = parseDom
	parseFiber.serializedUnbound = false
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseChild.effectTag = effectTagNone
		parseChild.serializedUnbound = true
		markSerializedDescendantsUnbound(parseChild)
	}
}

func markSerializedDescendantsUnbound(parseFiber *Fiber) {
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseChild.effectTag = effectTagNone
		parseChild.serializedUnbound = true
		markSerializedDescendantsUnbound(parseChild)
	}
}

// bindSerializedChildLevel resolves one already-parsed direct child list while
// its committed sibling order is still authoritative. It intentionally does
// not descend; untouched subtrees retain the mount-time bridge savings.
func (parseRt *Runtime) bindSerializedChildLevel(parseParent *Fiber) {
	if parseRt == nil || parseParent == nil || parseParent.child == nil {
		return
	}
	parseNeedsBinding := false
	for parseChild := parseParent.child; parseChild != nil; parseChild = parseChild.sibling {
		if parseChild.serializedUnbound {
			parseNeedsBinding = true
			break
		}
	}
	if !parseNeedsBinding || IsDOMNodeNull(parseParent.dom) {
		return
	}
	parseDomChild := parseRt.domAdapter.GetFirstChild(parseParent.dom)
	for parseChild := parseParent.child; parseChild != nil && !IsDOMNodeNull(parseDomChild); parseChild = parseChild.sibling {
		if parseChild.serializedUnbound {
			parseChild.dom = parseDomChild
			parseChild.serializedUnbound = false
		}
		parseDomChild = parseRt.domAdapter.GetNextSibling(parseDomChild)
	}
}
