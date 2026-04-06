package runtime

import (
	"strings"
	"testing"
)

type runtimeTagStringer struct {
	value string
}

// String returns one configurable tag-like value for tag-name helper tests.
func (parseS runtimeTagStringer) String() string {
	return parseS.value
}

type runtimeHelperTestDOMAdapter struct {
	*resolveContainerTestDOMAdapter
	attributeValues map[DOMNode]map[string]string
	tagValues       map[DOMNode]interface{}
}

// buildRuntimeHelperTestDOMAdapter constructs one DOM adapter that exposes selector, target-resolution, attribute, and tag-name helper branches.
func buildRuntimeHelperTestDOMAdapter() *runtimeHelperTestDOMAdapter {
	return &runtimeHelperTestDOMAdapter{
		resolveContainerTestDOMAdapter: buildResolveContainerTestDOMAdapter(),
		attributeValues:                make(map[DOMNode]map[string]string),
		tagValues:                      make(map[DOMNode]interface{}),
	}
}

// storeRuntimeAttributeValue stores one test attribute value for one DOM node.
func (parseA *runtimeHelperTestDOMAdapter) storeRuntimeAttributeValue(parseNode DOMNode, parseName string, parseValue string) {
	if parseA.attributeValues[parseNode] == nil {
		parseA.attributeValues[parseNode] = make(map[string]string)
	}
	parseA.attributeValues[parseNode][parseName] = parseValue
}

// GetAttribute returns one stored test attribute value when present.
func (parseA *runtimeHelperTestDOMAdapter) GetAttribute(parseNode DOMNode, parseName string) string {
	if parseAttrs := parseA.attributeValues[parseNode]; parseAttrs != nil {
		return parseAttrs[parseName]
	}
	return ""
}

// GetProperty returns one configured tag-name override before falling back to the base test DOM adapter.
func (parseA *runtimeHelperTestDOMAdapter) GetProperty(parseNode DOMNode, parseName string) interface{} {
	if parseName == "tagName" {
		if parseValue, parseOk := parseA.tagValues[parseNode]; parseOk {
			return parseValue
		}
	}
	return parseA.resolveContainerTestDOMAdapter.GetProperty(parseNode, parseName)
}

// TestRuntimeHelperAccessorsCoverOptionalAdapterBranches covers optional adapter branches for attribute and tag-name helper accessors.
func TestRuntimeHelperAccessorsCoverOptionalAdapterBranches(parseT *testing.T) {
	var parseNilRuntime *Runtime
	if parseValue, parseOk := parseNilRuntime.GetAttributeValue(nil, "data-id"); parseOk || parseValue != "" {
		parseT.Fatalf("expected nil runtime attribute lookup to fail, got value=%q ok=%t", parseValue, parseOk)
	}
	if parseTag, parseOk := parseNilRuntime.GetTagName(nil); parseOk || parseTag != "" {
		parseT.Fatalf("expected nil runtime tag lookup to fail, got tag=%q ok=%t", parseTag, parseOk)
	}

	parseBaseAdapter := newTestDOMAdapter()
	parseBaseNode := parseBaseAdapter.CreateElement("section")
	parseBaseRuntime := NewRuntime(Config{DOMAdapter: parseBaseAdapter, Scheduler: newTestScheduler()})
	if parseValue, parseOk := parseBaseRuntime.GetAttributeValue(parseBaseNode, "data-id"); parseOk || parseValue != "" {
		parseT.Fatalf("expected runtime without attribute-reader adapter to fail lookup, got value=%q ok=%t", parseValue, parseOk)
	}
	if parseTag, parseOk := parseBaseRuntime.GetTagName(parseBaseNode); !parseOk || parseTag != "section" {
		parseT.Fatalf("expected base tag-name normalization to succeed, got tag=%q ok=%t", parseTag, parseOk)
	}

	parseAdapter := buildRuntimeHelperTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseNode := parseAdapter.CreateElement("article")
	parseAdapter.storeRuntimeAttributeValue(parseNode, "data-id", "hero")
	if parseValue, parseOk := parseRt.GetAttributeValue(parseNode, "data-id"); !parseOk || parseValue != "hero" {
		parseT.Fatalf("expected attribute lookup to succeed, got value=%q ok=%t", parseValue, parseOk)
	}

	parseStringerNode := parseAdapter.CreateElement("unused")
	parseAdapter.tagValues[parseStringerNode] = runtimeTagStringer{value: " ARTICLE "}
	if parseTag, parseOk := parseRt.GetTagName(parseStringerNode); !parseOk || parseTag != "article" {
		parseT.Fatalf("expected stringer tag-name normalization to succeed, got tag=%q ok=%t", parseTag, parseOk)
	}

	parseBlankTagNode := parseAdapter.CreateElement("unused")
	parseAdapter.tagValues[parseBlankTagNode] = "   "
	if parseTag, parseOk := parseRt.GetTagName(parseBlankTagNode); parseOk || parseTag != "" {
		parseT.Fatalf("expected blank tag-name value to fail, got tag=%q ok=%t", parseTag, parseOk)
	}

	parseNilTagNode := parseAdapter.CreateElement("unused")
	parseAdapter.tagValues[parseNilTagNode] = nil
	if parseTag, parseOk := parseRt.GetTagName(parseNilTagNode); parseOk || parseTag != "" {
		parseT.Fatalf("expected nil tag-name property to fail, got tag=%q ok=%t", parseTag, parseOk)
	}

	parseInvalidTagNode := parseAdapter.CreateElement("unused")
	parseAdapter.tagValues[parseInvalidTagNode] = 42
	if parseTag, parseOk := parseRt.GetTagName(parseInvalidTagNode); parseOk || parseTag != "" {
		parseT.Fatalf("expected invalid tag-name property type to fail, got tag=%q ok=%t", parseTag, parseOk)
	}
}

// TestRuntimeHelperFindNodesCoverSelectorAndTargetTraversal covers selector, target, guard, and traversal branches for attribute-based DOM lookup.
func TestRuntimeHelperFindNodesCoverSelectorAndTargetTraversal(parseT *testing.T) {
	var parseNilRuntime *Runtime
	if parseMatches := parseNilRuntime.FindNodesWithAttributeInSelector("#app", "data-id"); parseMatches != nil {
		parseT.Fatalf("expected nil runtime selector traversal to return nil, got %#v", parseMatches)
	}
	if parseMatches := parseNilRuntime.FindNodesWithAttributeInTarget("#app", "data-id"); parseMatches != nil {
		parseT.Fatalf("expected nil runtime target traversal to return nil, got %#v", parseMatches)
	}

	parseAdapter := buildRuntimeHelperTestDOMAdapter()
	parseRoot := parseAdapter.CreateElement("main")
	parseSection := parseAdapter.CreateElement("section")
	parseBlankLeaf := parseAdapter.CreateElement("aside")
	parseLeaf := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseRoot, parseSection)
	parseAdapter.AppendChild(parseSection, parseBlankLeaf)
	parseAdapter.AppendChild(parseSection, parseLeaf)
	parseRoot.(*testDOMNode).children = append(parseRoot.(*testDOMNode).children, nil)
	parseAdapter.storeRuntimeAttributeValue(parseRoot, "data-id", "root")
	parseAdapter.storeRuntimeAttributeValue(parseSection, "data-id", "section")
	parseAdapter.storeRuntimeAttributeValue(parseBlankLeaf, "data-id", "   ")
	parseAdapter.storeRuntimeAttributeValue(parseLeaf, "data-id", "leaf")
	parseAdapter.selectorResults["#app"] = parseRoot
	parseAdapter.resolveByKey["app-root"] = parseRoot

	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	if parseMatches := parseRt.findNodesWithAttribute(nil, "data-id"); parseMatches != nil {
		parseT.Fatalf("expected nil-root traversal guard to return nil, got %#v", parseMatches)
	}
	if parseMatches := parseRt.findNodesWithAttribute(parseRoot, ""); parseMatches != nil {
		parseT.Fatalf("expected empty-attribute traversal guard to return nil, got %#v", parseMatches)
	}

	parseSelectorMatches := parseRt.FindNodesWithAttributeInSelector("#app", "data-id")
	if len(parseSelectorMatches) != 3 {
		parseT.Fatalf("expected three selector matches, got %d", len(parseSelectorMatches))
	}
	if parseSelectorMatches[0] != parseRoot || parseSelectorMatches[1] != parseSection || parseSelectorMatches[2] != parseLeaf {
		parseT.Fatalf("expected depth-first selector match order, got %#v", parseSelectorMatches)
	}

	if parseMissingMatches := parseRt.FindNodesWithAttributeInSelector("#missing", "data-id"); parseMissingMatches != nil {
		parseT.Fatalf("expected missing selector traversal to return nil, got %#v", parseMissingMatches)
	}

	parseTargetMatches := parseRt.FindNodesWithAttributeInTarget("app-root", "data-id")
	if len(parseTargetMatches) != 3 {
		parseT.Fatalf("expected three target matches, got %d", len(parseTargetMatches))
	}
	if parseTargetMatches[0] != parseRoot || parseTargetMatches[1] != parseSection || parseTargetMatches[2] != parseLeaf {
		parseT.Fatalf("expected depth-first target match order, got %#v", parseTargetMatches)
	}
}

// TestRuntimeHelperDurationHydrationAndErrorBranches covers remaining helper guards for hydration strictness, duration formatting, and explicit target-resolution failures.
func TestRuntimeHelperDurationHydrationAndErrorBranches(parseT *testing.T) {
	var parseNilRuntime *Runtime
	parseNilRuntime.SetNextHydrationStrict(true)

	parseRt := &Runtime{}
	parseRt.SetNextHydrationStrict(true)
	if !parseRt.nextHydrationStrict {
		parseT.Fatal("expected strict hydration flag to enable")
	}
	parseRt.SetNextHydrationStrict(false)
	if parseRt.nextHydrationStrict {
		parseT.Fatal("expected strict hydration flag to disable")
	}

	if parseGot := formatRuntimeDurationNs(-1); parseGot != "0ms" {
		parseT.Fatalf("expected non-positive duration to normalize to 0ms, got %q", parseGot)
	}
	if parseGot2 := formatRuntimeDurationNs(2_500_000); parseGot2 != "2.50ms" {
		parseT.Fatalf("expected positive duration formatting to preserve millisecond precision, got %q", parseGot2)
	}

	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := buildRuntimeHelperTestDOMAdapter()
	parseRuntime := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "widget"}}

	if parseErr := parseRuntime.RenderInto("missing", parseElement); parseErr == nil || !strings.Contains(parseErr.Error(), "target node could not be resolved") {
		parseT.Fatalf("expected RenderInto unresolved-target error, got %v", parseErr)
	}
	if parseErr := parseRuntime.HydrateInto("missing", parseElement); parseErr == nil || !strings.Contains(parseErr.Error(), "target node could not be resolved") {
		parseT.Fatalf("expected HydrateInto unresolved-target error, got %v", parseErr)
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 2 {
		parseT.Fatalf("expected two explicit unresolved-target diagnostics, got %+v", parseDiagnostics)
	}
	if !strings.Contains(parseDiagnostics[0].Message, "RenderInto failed") || !strings.Contains(parseDiagnostics[1].Message, "HydrateInto failed") {
		parseT.Fatalf("expected unresolved-target diagnostics for both paths, got %+v", parseDiagnostics)
	}
}
