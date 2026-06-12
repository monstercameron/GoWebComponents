package runtime

import (
	"errors"
	"testing"
)

// buildAgentRefFixtureTree builds ROOT -> ul -> [Item(key:a), Item(key:b) -> button]
// and returns the runtime plus the fibers a test addresses.
func buildAgentRefFixtureTree() (*Runtime, *Fiber, *Fiber, *Fiber) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseList := &Fiber{typeOf: "ul", parent: parseRoot}
	parseRoot.child = parseList

	parseItemType := &ComponentType{Name: "Item", QualifiedName: "example.com/app.Item"}
	parseItemA := &Fiber{typeOf: parseItemType, parent: parseList, props: map[string]any{"key": "a"}}
	parseItemB := &Fiber{typeOf: parseItemType, parent: parseList, props: map[string]any{"key": "b"}}
	parseList.child = parseItemA
	parseItemA.sibling = parseItemB

	parseButton := &Fiber{typeOf: "button", parent: parseItemB}
	parseItemB.child = parseButton

	parseRt := &Runtime{currentRoot: parseRoot}
	return parseRt, parseItemA, parseItemB, parseButton
}

// TestResolveAgentRefRoundTripsEveryFiber pins that every non-root fiber's
// AgentRefForFiber resolves back to that exact fiber pointer.
func TestResolveAgentRefRoundTripsEveryFiber(t *testing.T) {
	parseRt, parseItemA, parseItemB, parseButton := buildAgentRefFixtureTree()
	for _, parseFiber := range []*Fiber{parseRt.currentRoot.child, parseItemA, parseItemB, parseButton} {
		parseRef := AgentRefForFiber(parseFiber)
		if parseRef == "" {
			t.Fatalf("expected non-empty ref for fiber %v", parseFiber.typeOf)
		}
		parseResolved, parseErr := parseRt.ResolveAgentRef(parseRef)
		if parseErr != nil {
			t.Fatalf("resolve %q: %v", parseRef, parseErr)
		}
		if parseResolved != parseFiber {
			t.Fatalf("ref %q resolved to a different fiber", parseRef)
		}
	}
}

// TestResolveAgentRefDisambiguatesKeyedSiblings pins that two keyed siblings
// of the same component type get distinct refs resolving to the right instance.
func TestResolveAgentRefDisambiguatesKeyedSiblings(t *testing.T) {
	parseRt, parseItemA, parseItemB, _ := buildAgentRefFixtureTree()
	parseRefA := AgentRefForFiber(parseItemA)
	parseRefB := AgentRefForFiber(parseItemB)
	if parseRefA == parseRefB {
		t.Fatalf("keyed siblings share ref %q", parseRefA)
	}
	parseResolvedB, parseErr := parseRt.ResolveAgentRef(parseRefB)
	if parseErr != nil {
		t.Fatalf("resolve %q: %v", parseRefB, parseErr)
	}
	if parseResolvedB != parseItemB {
		t.Fatalf("ref %q resolved to the wrong sibling", parseRefB)
	}
}

// TestResolveAgentRefSurvivesRerender pins that a ref taken before a render
// that recreates every fiber still resolves to the equivalent new fiber.
func TestResolveAgentRefSurvivesRerender(t *testing.T) {
	parseRt, _, parseItemB, _ := buildAgentRefFixtureTree()
	parseRef := AgentRefForFiber(parseItemB)

	// Simulate a re-render: a structurally identical tree of brand-new fibers.
	parseRt2, _, parseNewItemB, _ := buildAgentRefFixtureTree()
	parseRt.currentRoot = parseRt2.currentRoot

	parseResolved, parseErr := parseRt.ResolveAgentRef(parseRef)
	if parseErr != nil {
		t.Fatalf("resolve after rerender: %v", parseErr)
	}
	if parseResolved == parseItemB {
		t.Fatalf("resolved the OLD fiber pointer; tree replacement not exercised")
	}
	if parseResolved != parseNewItemB {
		t.Fatalf("ref %q resolved to the wrong fiber after rerender", parseRef)
	}
}

// TestResolveAgentRefStaleAfterUnmount pins the stale-ref contract when the
// addressed component is no longer mounted.
func TestResolveAgentRefStaleAfterUnmount(t *testing.T) {
	parseRt, parseItemA, parseItemB, _ := buildAgentRefFixtureTree()
	parseRef := AgentRefForFiber(parseItemB)
	parseItemA.sibling = nil // unmount item b

	_, parseErr := parseRt.ResolveAgentRef(parseRef)
	if !errors.Is(parseErr, ErrAgentRefStale) {
		t.Fatalf("expected ErrAgentRefStale, got %v", parseErr)
	}
}

// TestResolveAgentRefIsReadOnly pins that resolving a ref (and walking past
// other fibers to reach it) never marks any fiber dirty or needing update -
// the todo's read-only-resolution guarantee, protected against regression.
func TestResolveAgentRefIsReadOnly(t *testing.T) {
	parseRt, parseItemA, parseItemB, parseButton := buildAgentRefFixtureTree()
	parseAll := []*Fiber{parseRt.currentRoot, parseRt.currentRoot.child, parseItemA, parseItemB, parseButton}

	if _, parseErr := parseRt.ResolveAgentRef(AgentRefForFiber(parseButton)); parseErr != nil {
		t.Fatalf("resolve: %v", parseErr)
	}
	// A stale lookup walks the tree too; it must also leave no marks.
	_, _ = parseRt.ResolveAgentRef("ul@0/does-not-exist@9")

	for _, parseFiber := range parseAll {
		if parseFiber.dirty {
			t.Fatalf("fiber %v became dirty during ref resolution", parseFiber.typeOf)
		}
		if parseFiber.needsUpdate {
			t.Fatalf("fiber %v was marked needsUpdate during ref resolution", parseFiber.typeOf)
		}
		if parseFiber.subtreeDirty {
			t.Fatalf("fiber %v subtreeDirty was set during ref resolution", parseFiber.typeOf)
		}
	}
}

// TestResolveAgentRefRejectsInvalidAndUnmountedTrees pins the invalid-ref and
// no-tree error paths.
func TestResolveAgentRefRejectsInvalidAndUnmountedTrees(t *testing.T) {
	parseRt, _, _, _ := buildAgentRefFixtureTree()
	if _, parseErr := parseRt.ResolveAgentRef("   "); !errors.Is(parseErr, ErrAgentRefInvalid) {
		t.Fatalf("expected ErrAgentRefInvalid for blank ref, got %v", parseErr)
	}
	parseEmpty := &Runtime{}
	if _, parseErr := parseEmpty.ResolveAgentRef("ul@0"); !errors.Is(parseErr, ErrAgentRefStale) {
		t.Fatalf("expected ErrAgentRefStale with no mounted tree, got %v", parseErr)
	}
}

// TestInspectAnnotatesAgentRefs pins that inspection snapshots carry each
// node's resolvable agent ref (and an empty ref on the root).
func TestInspectAnnotatesAgentRefs(t *testing.T) {
	parseRt, _, parseItemB, _ := buildAgentRefFixtureTree()
	parseSnapshot := parseRt.Inspect()
	if parseSnapshot.Root == nil {
		t.Fatalf("expected a root snapshot")
	}
	if parseSnapshot.Root.AgentRef != "" {
		t.Fatalf("root must not be addressable, got ref %q", parseSnapshot.Root.AgentRef)
	}

	// Walk the snapshot for item b's node and require its ref to match the
	// live fiber's ref and to resolve back to the live fiber.
	parseWantRef := AgentRefForFiber(parseItemB)
	parseFound := false
	parseStack := []*FiberSnapshot{parseSnapshot.Root}
	for len(parseStack) > 0 {
		parseNode := parseStack[len(parseStack)-1]
		parseStack = parseStack[:len(parseStack)-1]
		if parseNode.AgentRef == parseWantRef {
			parseFound = true
			break
		}
		for parseIndex := range parseNode.Children {
			parseStack = append(parseStack, &parseNode.Children[parseIndex])
		}
	}
	if !parseFound {
		t.Fatalf("snapshot does not carry agent ref %q", parseWantRef)
	}
	parseResolved, parseErr := parseRt.ResolveAgentRef(parseWantRef)
	if parseErr != nil || parseResolved != parseItemB {
		t.Fatalf("snapshot ref %q did not resolve to the live fiber: %v", parseWantRef, parseErr)
	}
}
