package agentui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/agentui"
	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// sampleAgentSchema is the kind of JSON an agent emits server-side: a small dashboard view
// composed only of allow-listed, presentational components.
const sampleAgentSchema = `{
  "type": "stack",
  "children": [
    {"type": "heading", "props": {"level": "1"}, "text": "Quarterly Report"},
    {"type": "card", "children": [
      {"type": "text", "text": "All systems green"},
      {"type": "badge", "props": {"class": "ok"}, "text": "healthy"}
    ]}
  ]
}`

// TestValidateAcceptsAllowListedTree proves a fully allow-listed tree passes.
func TestValidateAcceptsAllowListedTree(parseT *testing.T) {
	parseNode, parseErr := agentui.Parse([]byte(sampleAgentSchema))
	if parseErr != nil {
		parseT.Fatalf("parse: %v", parseErr)
	}
	if parseErr := agentui.DefaultRegistry().Validate(parseNode); parseErr != nil {
		parseT.Fatalf("expected the allow-listed tree to validate, got %v", parseErr)
	}
}

// TestValidateRejectsUnknownType proves a component outside the allow-list is rejected with
// its type named — the core safety property.
func TestValidateRejectsUnknownType(parseT *testing.T) {
	parseErr := agentui.DefaultRegistry().Validate(agentui.Node{Type: "script"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), `"script"`) {
		parseT.Fatalf("expected an allow-list rejection naming script, got %v", parseErr)
	}
}

// TestValidateRejectsDisallowedProp proves a prop the component does not permit is
// rejected (an agent cannot smuggle arbitrary attributes).
func TestValidateRejectsDisallowedProp(parseT *testing.T) {
	parseNode := agentui.Node{Type: "text", Props: map[string]string{"onclick": "steal()"}}
	parseErr := agentui.DefaultRegistry().Validate(parseNode)
	if parseErr == nil || !strings.Contains(parseErr.Error(), "onclick") {
		parseT.Fatalf("expected a disallowed-prop rejection naming onclick, got %v", parseErr)
	}
}

// TestValidateRejectsNestedViolation proves validation is recursive and the error names the
// offending path.
func TestValidateRejectsNestedViolation(parseT *testing.T) {
	parseNode := agentui.Node{Type: "stack", Children: []agentui.Node{
		{Type: "card", Children: []agentui.Node{
			{Type: "iframe"},
		}},
	}}
	parseErr := agentui.DefaultRegistry().Validate(parseNode)
	if parseErr == nil || !strings.Contains(parseErr.Error(), `"iframe"`) {
		parseT.Fatalf("expected a nested rejection naming iframe, got %v", parseErr)
	}
}

// TestRenderJSONRejectsUnknownTypeBeforeRendering proves Render gates on validation: bad
// agent output never produces a node.
func TestRenderJSONRejectsUnknownTypeBeforeRendering(parseT *testing.T) {
	parseNode, parseErr := agentui.DefaultRegistry().RenderJSON([]byte(`{"type":"object","props":{"data":"evil"}}`))
	if parseErr == nil {
		parseT.Fatal("expected RenderJSON to reject a non-allow-listed type")
	}
	if parseNode != nil {
		parseT.Fatal("expected no node when validation fails")
	}
}

// TestValidateRejectsTooDeepTree proves the depth guard stops a hostile or runaway nesting
// from exhausting the stack.
func TestValidateRejectsTooDeepTree(parseT *testing.T) {
	// Build a chain of stacks deeper than the limit.
	parseDeep := agentui.Node{Type: "stack"}
	parseCursor := &parseDeep
	for range 40 {
		parseChild := agentui.Node{Type: "stack"}
		parseCursor.Children = []agentui.Node{parseChild}
		parseCursor = &parseCursor.Children[0]
	}
	parseErr := agentui.DefaultRegistry().ValidateWithLimits(parseDeep, agentui.Limits{MaxDepth: 8})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "max depth") {
		parseT.Fatalf("expected a max-depth rejection, got %v", parseErr)
	}
}

// TestValidateRejectsTooManyNodes proves the node-count guard bounds total tree size.
func TestValidateRejectsTooManyNodes(parseT *testing.T) {
	parseWide := agentui.Node{Type: "stack"}
	for range 20 {
		parseWide.Children = append(parseWide.Children, agentui.Node{Type: "text", Text: "x"})
	}
	parseErr := agentui.DefaultRegistry().ValidateWithLimits(parseWide, agentui.Limits{MaxNodes: 10})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "max node count") {
		parseT.Fatalf("expected a max-node-count rejection, got %v", parseErr)
	}
}

// TestRenderToDOMEndToEnd is the FC3 guarantee end-to-end: an agent's JSON schema is parsed,
// validated against the allow-list, rendered, and mounted — and the resulting DOM shows the
// agent's content. Rendered natively through the real runtime + a mock DOM.
func TestRenderToDOMEndToEnd(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")

	parseNode, parseErr := agentui.DefaultRegistry().RenderJSON([]byte(sampleAgentSchema))
	if parseErr != nil {
		parseT.Fatalf("RenderJSON: %v", parseErr)
	}
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}

	parseGot := collectDOMText(parseAdapter, parseRoot)
	for _, parseWant := range []string{"Quarterly Report", "All systems green", "healthy"} {
		if !strings.Contains(parseGot, parseWant) {
			parseT.Fatalf("rendered DOM missing %q; got %q", parseWant, parseGot)
		}
	}
}

// collectDOMText concatenates the text content of a rendered mock-DOM subtree.
func collectDOMText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectDOMText(parseAdapter, parseChild)
	}
	return parseText
}
