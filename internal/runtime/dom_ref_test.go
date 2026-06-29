package runtime

import "testing"

// captureSink is a test DOMRefSink that records the published node and counts
// attach/detach transitions.
type captureSink struct {
	node     DOMNode
	attaches int
	detaches int
}

func (parseS *captureSink) SetDOMNode(parseNode DOMNode) {
	parseS.node = parseNode
	if IsDOMNodeNull(parseNode) {
		parseS.detaches++
	} else {
		parseS.attaches++
	}
}

// renderHostWith builds a host div optionally containing an input that carries
// the given ref sink (and key, for remount tests).
func renderHostWith(parseSink DOMRefSink, parseInclude bool, parseKey string) *Element {
	parseKids := []any{}
	if parseInclude {
		parseProps := map[string]any{"id": "target", DOMRefKey: parseSink}
		if parseKey != "" {
			parseProps["key"] = parseKey
		}
		parseKids = append(parseKids, CreateElement("input", parseProps))
	}
	return CreateElement("div", map[string]any{"id": "host"}, parseKids...)
}

func TestDOMRefPublishesOnMountAndClearsOnUnmount(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseSink := &captureSink{}

	// Mount: ref must point at the live <input>.
	parseRt.Render(renderHostWith(parseSink, true, ""), parseContainer)
	if IsDOMNodeNull(parseSink.node) {
		parseT.Fatal("ref not published on mount")
	}
	parseNode, parseOk := parseSink.node.(*testDOMNode)
	if !parseOk || parseNode.tag != "input" {
		parseT.Fatalf("ref published wrong node: %#v", parseSink.node)
	}
	if parseSink.attaches != 1 || parseSink.detaches != 0 {
		parseT.Fatalf("expected one attach, zero detach; got attaches=%d detaches=%d", parseSink.attaches, parseSink.detaches)
	}

	// Unmount: ref must be cleared to nil.
	parseRt.Render(renderHostWith(parseSink, false, ""), parseContainer)
	if !IsDOMNodeNull(parseSink.node) {
		parseT.Fatalf("ref not cleared on unmount: %#v", parseSink.node)
	}
	if parseSink.detaches != 1 {
		parseT.Fatalf("expected one detach, got %d", parseSink.detaches)
	}
}

func TestDOMRefKeyIsNeverAppliedToTheDOM(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseSink := &captureSink{}

	parseRt.Render(renderHostWith(parseSink, true, ""), parseContainer)

	parseNode, _ := parseSink.node.(*testDOMNode)
	if parseNode == nil {
		parseT.Fatal("ref not published")
	}
	if _, parseExists := parseNode.attributes[DOMRefKey]; parseExists {
		parseT.Errorf("ref key leaked as an attribute: %v", parseNode.attributes)
	}
	if _, parseExists := parseNode.properties[DOMRefKey]; parseExists {
		parseT.Errorf("ref key leaked as a property: %v", parseNode.properties)
	}
	if getPropMeta(DOMRefKey).kind != propKindSkip {
		parseT.Error("DOMRefKey should be registered propKindSkip")
	}
}

func TestDOMRefReassignsAcrossRemount(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseSink := &captureSink{}

	parseRt.Render(renderHostWith(parseSink, true, "a"), parseContainer)
	parseFirst := parseSink.node
	if IsDOMNodeNull(parseFirst) {
		parseT.Fatal("no initial mount")
	}

	// Unmount then mount a fresh keyed element reusing the same ref. Because
	// commitRoot processes deletions before placements, the ref ends pointing at
	// the NEW node, never stuck nil.
	parseRt.Render(renderHostWith(parseSink, false, ""), parseContainer)
	if !IsDOMNodeNull(parseSink.node) {
		parseT.Fatal("ref should detach between mounts")
	}
	parseRt.Render(renderHostWith(parseSink, true, "b"), parseContainer)
	if IsDOMNodeNull(parseSink.node) {
		parseT.Fatal("ref should re-attach on remount")
	}
	if parseSink.attaches != 2 || parseSink.detaches != 1 {
		parseT.Fatalf("expected attaches=2 detaches=1, got attaches=%d detaches=%d", parseSink.attaches, parseSink.detaches)
	}
}

func TestDOMRefNilSinkAndMissingKeyAreSafe(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})

	// publishDOMRef must tolerate: nil fiber, nil props, missing key, and a
	// typed-nil sink stored under the key.
	parseRt.publishDOMRef(nil, nil)
	parseRt.publishDOMRef(&Fiber{}, nil)
	parseRt.publishDOMRef(&Fiber{props: map[string]any{"id": "x"}}, parseAdapter.CreateElement("div"))
	var parseTypedNil DOMRefSink
	parseRt.publishDOMRef(&Fiber{props: map[string]any{DOMRefKey: parseTypedNil}}, parseAdapter.CreateElement("div"))
	parseRt.releaseDOMRefsSubtree(nil)
	// Reaching here without a panic is the assertion.
}
