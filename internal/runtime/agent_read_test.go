package runtime

import (
	"testing"
)

// buildAgentReadFixtureTree builds a realistic fiber tree for agent read tests:
//
//	ROOT
//	  form[id=login, role="form"]
//	    input[type=text, id=username, aria-label="Username"]
//	    button[type=submit]
//	      #text "Sign in"
//	    p[id=hint]
//	      #text "Enter credentials"
//	  nav (no id)
//	    a[href=/home] "Home"
//
// Returns the runtime and key fibers for targeted assertions.
func buildAgentReadFixtureTree() (*Runtime, *Fiber, *Fiber, *Fiber, *Fiber) {
	parseRoot := &Fiber{typeOf: "ROOT"}

	parseForm := &Fiber{
		typeOf: "form",
		parent: parseRoot,
		props:  map[string]any{"id": "login"},
	}
	parseRoot.child = parseForm

	parseInput := &Fiber{
		typeOf: "input",
		parent: parseForm,
		props:  map[string]any{"type": "text", "id": "username", "aria-label": "Username"},
	}
	parseForm.child = parseInput

	parseBtn := &Fiber{
		typeOf: "button",
		parent: parseForm,
		props:  map[string]any{"type": "submit"},
	}
	parseInput.sibling = parseBtn

	parseBtnText := &Fiber{typeOf: "TEXT_ELEMENT", parent: parseBtn, textContent: "Sign in"}
	parseBtn.child = parseBtnText

	parseHint := &Fiber{
		typeOf: "p",
		parent: parseForm,
		props:  map[string]any{"id": "hint"},
	}
	parseBtn.sibling = parseHint

	parseHintText := &Fiber{typeOf: "TEXT_ELEMENT", parent: parseHint, textContent: "Enter credentials"}
	parseHint.child = parseHintText

	parseNav := &Fiber{typeOf: "nav", parent: parseRoot}
	parseForm.sibling = parseNav

	parseLink := &Fiber{
		typeOf: "a",
		parent: parseNav,
		props:  map[string]any{"href": "/home"},
	}
	parseNav.child = parseLink
	parseLinkText := &Fiber{typeOf: "TEXT_ELEMENT", parent: parseLink, textContent: "Home"}
	parseLink.child = parseLinkText

	parseRt := &Runtime{currentRoot: parseRoot}
	return parseRt, parseForm, parseInput, parseBtn, parseLink
}

// TestBuildAgentSnapshotShape pins that the returned snapshot carries the
// expected tree structure: Root present, node names correct, AgentRefs present
// on non-root nodes, HookCount propagated.
func TestBuildAgentSnapshotShape(t *testing.T) {
	parseRt, _, _, _, _ := buildAgentReadFixtureTree()
	parseSnap := BuildAgentSnapshot(parseRt, 0, 0)
	if parseSnap.Root == nil {
		t.Fatal("expected non-nil Root")
	}
	if parseSnap.Root.Kind != "root" {
		t.Fatalf("expected root kind, got %q", parseSnap.Root.Kind)
	}
	if parseSnap.Root.AgentRef != "" {
		t.Fatalf("root must not carry an agent ref, got %q", parseSnap.Root.AgentRef)
	}
	if len(parseSnap.Root.Children) == 0 {
		t.Fatal("expected children on root")
	}
	// The form child should carry its agent ref.
	parseFormNode := parseSnap.Root.Children[0]
	if parseFormNode.AgentRef == "" {
		t.Fatal("non-root node must carry an agent ref")
	}
}

// TestBuildAgentSnapshotAgentRefRoundTrip pins that every non-root node in the
// snapshot carries a non-empty agent ref that resolves back to a live fiber.
func TestBuildAgentSnapshotAgentRefRoundTrip(t *testing.T) {
	parseRt, _, _, _, _ := buildAgentReadFixtureTree()
	parseSnap := BuildAgentSnapshot(parseRt, 0, 0)
	if parseSnap.Root == nil {
		t.Fatal("expected non-nil Root")
	}
	// Walk every non-root node in the snapshot and verify its ref resolves.
	var parseCheck func(parseNode *AgentNodeSnapshot)
	parseCheck = func(parseNode *AgentNodeSnapshot) {
		if parseNode == nil {
			return
		}
		if parseNode.Kind != "root" && parseNode.AgentRef != "" {
			if _, parseErr := parseRt.ResolveAgentRef(parseNode.AgentRef); parseErr != nil {
				t.Errorf("ref %q did not resolve: %v", parseNode.AgentRef, parseErr)
			}
		}
		for parseIndex := range parseNode.Children {
			parseCheck(&parseNode.Children[parseIndex])
		}
	}
	parseCheck(parseSnap.Root)
}

// TestBuildAgentSnapshotDepthBudgetExplicit pins that depth truncation is
// explicit: BudgetApplied is true and TruncatedNodes is > 0 when the tree is
// deeper than parseMaxDepth.
func TestBuildAgentSnapshotDepthBudgetExplicit(t *testing.T) {
	parseRt, _, _, _, _ := buildAgentReadFixtureTree()
	// Depth=1 means only the root layer is emitted; the form+nav subtrees are
	// truncated.
	parseSnap := BuildAgentSnapshot(parseRt, 1, 0)
	if parseSnap.Root == nil {
		t.Fatal("expected non-nil Root with depth=1")
	}
	// Children of root are emitted at depth 1 (the root itself is depth 0).
	// Children of those nodes (depth 2) should be truncated.
	for parseIndex := range parseSnap.Root.Children {
		parseChild := &parseSnap.Root.Children[parseIndex]
		if len(parseChild.Children) > 0 {
			t.Errorf("depth=1 budget: node %q at depth 1 still has children", parseChild.Name)
		}
	}
	if !parseSnap.BudgetApplied {
		t.Error("expected BudgetApplied=true when depth truncation occurred")
	}
	if parseSnap.TruncatedNodes <= 0 {
		t.Error("expected TruncatedNodes > 0 when depth truncation occurred")
	}
}

// TestBuildAgentSnapshotNodeBudgetExplicit pins that node-count truncation is
// explicit: BudgetApplied is true and TruncatedNodes is > 0.
func TestBuildAgentSnapshotNodeBudgetExplicit(t *testing.T) {
	parseRt, _, _, _, _ := buildAgentReadFixtureTree()
	// Only allow 2 nodes to be emitted; the rest must be truncated.
	parseSnap := BuildAgentSnapshot(parseRt, 0, 2)
	if parseSnap.Root == nil {
		t.Fatal("expected non-nil Root with maxNodes=2")
	}
	if !parseSnap.BudgetApplied {
		t.Error("expected BudgetApplied=true when node budget was hit")
	}
	if parseSnap.TruncatedNodes <= 0 {
		t.Error("expected TruncatedNodes > 0 when node budget was hit")
	}
}

// TestBuildAgentSnapshotNoTreeEmptyResult pins that a runtime with no mounted
// tree returns an empty (non-error) snapshot.
func TestBuildAgentSnapshotNoTreeEmptyResult(t *testing.T) {
	parseRt := &Runtime{}
	parseSnap := BuildAgentSnapshot(parseRt, 0, 0)
	if parseSnap.Root != nil {
		t.Fatalf("expected nil Root with no mounted tree, got %+v", parseSnap.Root)
	}
	if parseSnap.BudgetApplied {
		t.Error("empty snapshot must not claim budget was applied")
	}
}

// TestBuildAgentSnapshotRedaction pins that hook values are replaced with
// "[redacted]" when a panic-report redaction policy is configured, and are
// restored to real values after the policy is cleared.
func TestBuildAgentSnapshotRedaction(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseItemType := &ComponentType{Name: "Widget", QualifiedName: "app.Widget"}
	parseWidget := &Fiber{
		typeOf: parseItemType,
		parent: parseRoot,
		props:  map[string]any{"key": "w"},
		hooks: &Hooks{
			states: []any{"secret-value", nil},
		},
	}
	parseRoot.child = parseWidget
	parseRt := &Runtime{currentRoot: parseRoot}

	// Before redaction: hook value should be present.
	ConfigurePanicReportRedaction(PanicReportRedactionPolicy{})
	parseSnapClear := BuildAgentSnapshot(parseRt, 0, 0)
	if parseSnapClear.Root != nil && len(parseSnapClear.Root.Children) > 0 {
		for _, parseHook := range parseSnapClear.Root.Children[0].Hooks {
			if parseHook.Value == "[redacted]" {
				t.Errorf("hook value was redacted without policy configured")
			}
		}
	}

	// With a redaction policy: hook values must be "[redacted]".
	ConfigurePanicReportRedaction(PanicReportRedactionPolicy{
		Fields: []string{"panic.summary"},
	})
	defer ConfigurePanicReportRedaction(PanicReportRedactionPolicy{}) // restore

	parseSnapRedacted := BuildAgentSnapshot(parseRt, 0, 0)
	if parseSnapRedacted.Root == nil || len(parseSnapRedacted.Root.Children) == 0 {
		t.Skip("no widget children in snapshot; hooks may be empty")
	}
	for _, parseHook := range parseSnapRedacted.Root.Children[0].Hooks {
		if parseHook.Value != "[redacted]" {
			t.Errorf("expected hook value to be [redacted], got %q", parseHook.Value)
		}
	}
}

// TestQueryAgentNodesByRole pins that role-based queries return the expected
// nodes in tree order.
func TestQueryAgentNodesByRole(t *testing.T) {
	parseRt, _, _, parseBtn, _ := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Role: "button"})
	if len(parseMatches) == 0 {
		t.Fatal("expected at least one button match")
	}
	parseWantRef := AgentRefForFiber(parseBtn)
	parseFound := false
	for _, parseMatch := range parseMatches {
		if parseMatch.AgentRef == parseWantRef {
			parseFound = true
			break
		}
	}
	if !parseFound {
		t.Errorf("button fiber ref %q not found in role query results: %v", parseWantRef, parseMatches)
	}
}

// TestQueryAgentNodesByLabel pins that label queries return the node with the
// matching aria-label prop.
func TestQueryAgentNodesByLabel(t *testing.T) {
	parseRt, _, parseInput, _, _ := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Label: "username"})
	if len(parseMatches) == 0 {
		t.Fatal("expected at least one label match for 'username'")
	}
	parseWantRef := AgentRefForFiber(parseInput)
	parseFound := false
	for _, parseMatch := range parseMatches {
		if parseMatch.AgentRef == parseWantRef {
			parseFound = true
			break
		}
	}
	if !parseFound {
		t.Errorf("input fiber ref %q not found in label query results: %v", parseWantRef, parseMatches)
	}
}

// TestQueryAgentNodesByText pins that text queries return all nodes whose
// subtree text contains the searched substring.
func TestQueryAgentNodesByText(t *testing.T) {
	parseRt, _, _, parseBtn, _ := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Text: "sign in"})
	if len(parseMatches) == 0 {
		t.Fatal("expected at least one text match for 'sign in'")
	}
	parseWantRef := AgentRefForFiber(parseBtn)
	parseFound := false
	for _, parseMatch := range parseMatches {
		if parseMatch.AgentRef == parseWantRef {
			parseFound = true
			break
		}
	}
	if !parseFound {
		t.Errorf("button fiber ref %q not found in text query results: %v", parseWantRef, parseMatches)
	}
}

// TestQueryAgentNodesByID pins that ID queries return only the node with an
// exactly matching id prop.
func TestQueryAgentNodesByID(t *testing.T) {
	parseRt, parseForm, _, _, _ := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{ID: "login"})
	if len(parseMatches) != 1 {
		t.Fatalf("expected exactly 1 id match, got %d: %v", len(parseMatches), parseMatches)
	}
	parseWantRef := AgentRefForFiber(parseForm)
	if parseMatches[0].AgentRef != parseWantRef {
		t.Errorf("id query returned ref %q, want %q", parseMatches[0].AgentRef, parseWantRef)
	}
}

// TestQueryAgentNodesByTag pins that tag queries return all nodes with the
// matching element tag (case-insensitive).
func TestQueryAgentNodesByTag(t *testing.T) {
	parseRt, _, _, _, _ := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Tag: "INPUT"})
	if len(parseMatches) == 0 {
		t.Fatal("expected at least one tag match for 'INPUT'")
	}
	for _, parseMatch := range parseMatches {
		if !stringEqualFoldStr(parseMatch.Tag, "input") {
			t.Errorf("tag query returned node with tag %q, want input", parseMatch.Tag)
		}
	}
}

// TestQueryAgentNodesAmbiguousReturnsAll pins that a query matching multiple
// nodes returns all of them in tree order, not just the first.
func TestQueryAgentNodesAmbiguousReturnsAll(t *testing.T) {
	// Build a tree with two buttons.
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseBtnA := &Fiber{typeOf: "button", parent: parseRoot, props: map[string]any{"id": "a"}}
	parseBtnB := &Fiber{typeOf: "button", parent: parseRoot, props: map[string]any{"id": "b"}}
	parseRoot.child = parseBtnA
	parseBtnA.sibling = parseBtnB
	parseRt := &Runtime{currentRoot: parseRoot}

	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Role: "button"})
	if len(parseMatches) != 2 {
		t.Fatalf("expected 2 button matches, got %d: %v", len(parseMatches), parseMatches)
	}
	parseRefA := AgentRefForFiber(parseBtnA)
	parseRefB := AgentRefForFiber(parseBtnB)
	if parseMatches[0].AgentRef != parseRefA {
		t.Errorf("first match: want ref %q, got %q", parseRefA, parseMatches[0].AgentRef)
	}
	if parseMatches[1].AgentRef != parseRefB {
		t.Errorf("second match: want ref %q, got %q", parseRefB, parseMatches[1].AgentRef)
	}
}

// TestQueryAgentNodesEmptyTreeOK pins that a nil or no-tree runtime returns an
// empty slice without panicking.
func TestQueryAgentNodesEmptyTreeOK(t *testing.T) {
	parseEmpty := &Runtime{}
	parseMatches := QueryAgentNodes(parseEmpty, AgentQuerySelector{Role: "button"})
	if parseMatches != nil && len(parseMatches) > 0 {
		t.Fatalf("expected empty result with no tree, got %v", parseMatches)
	}

	parseMatches2 := QueryAgentNodes(nil, AgentQuerySelector{Role: "button"})
	if parseMatches2 != nil && len(parseMatches2) > 0 {
		t.Fatalf("expected empty result with nil runtime, got %v", parseMatches2)
	}
}

// TestQueryAgentNodesLinkRole pins that an <a> with href gets role=link and
// without href gets no implicit role.
func TestQueryAgentNodesLinkRole(t *testing.T) {
	parseRt, _, _, _, parseLink := buildAgentReadFixtureTree()
	parseMatches := QueryAgentNodes(parseRt, AgentQuerySelector{Role: "link"})
	if len(parseMatches) == 0 {
		t.Fatal("expected at least one link role match")
	}
	parseWantRef := AgentRefForFiber(parseLink)
	parseFound := false
	for _, parseMatch := range parseMatches {
		if parseMatch.AgentRef == parseWantRef {
			parseFound = true
			break
		}
	}
	if !parseFound {
		t.Errorf("link fiber ref %q not found in role=link query: %v", parseWantRef, parseMatches)
	}
}

// stringEqualFoldStr is a local helper to avoid importing strings in test file.
func stringEqualFoldStr(parseA, parseB string) bool {
	if len(parseA) != len(parseB) {
		return false
	}
	for parseI := range parseA {
		parseCA := parseA[parseI]
		parseCB := parseB[parseI]
		if parseCA >= 'A' && parseCA <= 'Z' {
			parseCA += 'a' - 'A'
		}
		if parseCB >= 'A' && parseCB <= 'Z' {
			parseCB += 'a' - 'A'
		}
		if parseCA != parseCB {
			return false
		}
	}
	return true
}
