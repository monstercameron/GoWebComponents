package runtime

import (
	"testing"
)

// ============================================================================
// Fiber Structure and Relationship Tests - 25 tests
// ============================================================================

func TestFiber_ParentChildRelationship(t *testing.T) {
	parent := &Fiber{}
	child := &Fiber{}

	parent.child = child
	child.parent = parent

	if parent.child != child {
		t.Error("Expected parent.child to reference child")
	}
	if child.parent != parent {
		t.Error("Expected child.parent to reference parent")
	}
}

func TestFiber_SiblingChain(t *testing.T) {
	sibling1 := &Fiber{}
	sibling2 := &Fiber{}
	sibling3 := &Fiber{}

	sibling1.sibling = sibling2
	sibling2.sibling = sibling3

	if sibling1.sibling != sibling2 {
		t.Error("Expected sibling1.sibling to reference sibling2")
	}
	if sibling2.sibling != sibling3 {
		t.Error("Expected sibling2.sibling to reference sibling3")
	}
	if sibling3.sibling != nil {
		t.Error("Expected sibling3.sibling to be nil")
	}
}

func TestFiber_AlternateReference(t *testing.T) {
	fiber1 := &Fiber{}
	fiber2 := &Fiber{alternate: fiber1}

	if fiber2.alternate != fiber1 {
		t.Error("Expected fiber2.alternate to reference fiber1")
	}
}

func TestFiber_DOMReference(t *testing.T) {
	adapter := newTestDOMAdapter()
	dom := adapter.CreateElement("div")

	fiber := &Fiber{dom: dom}

	if fiber.dom == nil {
		t.Error("Expected fiber.dom to be set")
	}
}

func TestFiber_EffectTag_Placement(t *testing.T) {
	fiber := &Fiber{effectTag: "PLACEMENT"}

	if fiber.effectTag != "PLACEMENT" {
		t.Error("Expected effectTag to be PLACEMENT")
	}
}

func TestFiber_EffectTag_Update(t *testing.T) {
	fiber := &Fiber{effectTag: "UPDATE"}

	if fiber.effectTag != "UPDATE" {
		t.Error("Expected effectTag to be UPDATE")
	}
}

func TestFiber_EffectTag_Deletion(t *testing.T) {
	fiber := &Fiber{effectTag: "DELETION"}

	if fiber.effectTag != "DELETION" {
		t.Error("Expected effectTag to be DELETION")
	}
}

func TestFiber_PropsInitialization(t *testing.T) {
	props := map[string]interface{}{
		"id":        "test",
		"className": "container",
	}
	fiber := &Fiber{props: props}

	if fiber.props["id"] != "test" {
		t.Error("Expected id prop")
	}
	if fiber.props["className"] != "container" {
		t.Error("Expected className prop")
	}
}

func TestFiber_EmptyProps(t *testing.T) {
	fiber := &Fiber{props: make(map[string]interface{})}

	if fiber.props == nil {
		t.Error("Expected props to be initialized")
	}
	if len(fiber.props) != 0 {
		t.Error("Expected props to be empty")
	}
}

func TestFiber_DirtyFlag(t *testing.T) {
	fiber := &Fiber{dirty: true}

	if !fiber.dirty {
		t.Error("Expected fiber to be marked dirty")
	}
}

func TestFiber_CleanFlag(t *testing.T) {
	fiber := &Fiber{dirty: false}

	if fiber.dirty {
		t.Error("Expected fiber to not be marked dirty")
	}
}

func TestFiber_EffectsArray(t *testing.T) {
	effect1 := func() func() { return nil }
	effect2 := func() func() { return nil }

	fiber := &Fiber{
		effects: []Effect{{Fn: effect1}, {Fn: effect2}},
	}

	if len(fiber.effects) != 2 {
		t.Errorf("Expected 2 effects, got %d", len(fiber.effects))
	}
}

func TestFiber_EmptyEffects(t *testing.T) {
	fiber := &Fiber{effects: []Effect{}}

	if len(fiber.effects) != 0 {
		t.Error("Expected empty effects array")
	}
}

func TestFiber_HooksAttachment(t *testing.T) {
	hooks := &Hooks{
		states: make([]interface{}, 0),
		index:  0,
	}
	fiber := &Fiber{hooks: hooks}

	if fiber.hooks == nil {
		t.Error("Expected hooks to be attached")
	}
	if fiber.hooks.index != 0 {
		t.Error("Expected hooks index to be 0")
	}
}

func TestFiber_TypeOfString(t *testing.T) {
	fiber := &Fiber{typeOf: "div"}

	if fiber.typeOf != "div" {
		t.Error("Expected typeOf to be string 'div'")
	}
}

func TestFiber_TypeOfFunction(t *testing.T) {
	component := func(props map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}
	fiber := &Fiber{typeOf: component}

	if fiber.typeOf == nil {
		t.Error("Expected typeOf to be function")
	}
}

func TestFiber_TextFiber(t *testing.T) {
	fiber := &Fiber{typeOf: "TEXT_ELEMENT"}

	if fiber.typeOf != "TEXT_ELEMENT" {
		t.Error("Expected typeOf to be TEXT_ELEMENT")
	}
}

func TestFiber_NilDOM(t *testing.T) {
	fiber := &Fiber{dom: nil}

	if fiber.dom != nil {
		t.Error("Expected dom to be nil")
	}
}

func TestFiber_NilAlternateReference(t *testing.T) {
	fiber := &Fiber{alternate: nil}

	if fiber.alternate != nil {
		t.Error("Expected alternate to be nil")
	}
}

func TestFiber_ComplexTree(t *testing.T) {
	//     root
	//     /  \
	//  child1 child2
	//    |
	//  grandchild

	root := &Fiber{typeOf: "root"}
	child1 := &Fiber{typeOf: "child1", parent: root}
	child2 := &Fiber{typeOf: "child2", parent: root}
	grandchild := &Fiber{typeOf: "grandchild", parent: child1}

	root.child = child1
	child1.sibling = child2
	child1.child = grandchild

	// Verify structure
	if root.child != child1 {
		t.Error("root.child should be child1")
	}
	if child1.sibling != child2 {
		t.Error("child1.sibling should be child2")
	}
	if child1.child != grandchild {
		t.Error("child1.child should be grandchild")
	}
	if grandchild.parent != child1 {
		t.Error("grandchild.parent should be child1")
	}
}

func TestFiber_CircularAlternate(t *testing.T) {
	fiber1 := &Fiber{typeOf: "div"}
	fiber2 := &Fiber{typeOf: "div"}

	fiber1.alternate = fiber2
	fiber2.alternate = fiber1

	if fiber1.alternate != fiber2 {
		t.Error("Expected fiber1.alternate to be fiber2")
	}
	if fiber2.alternate != fiber1 {
		t.Error("Expected fiber2.alternate to be fiber1")
	}
}

func TestFiber_LinearSiblingChain(t *testing.T) {
	siblings := make([]*Fiber, 10)
	for i := 0; i < 10; i++ {
		siblings[i] = &Fiber{typeOf: "sibling"}
		if i > 0 {
			siblings[i-1].sibling = siblings[i]
		}
	}

	// Verify chain
	for i := 0; i < 9; i++ {
		if siblings[i].sibling != siblings[i+1] {
			t.Errorf("Sibling %d should link to sibling %d", i, i+1)
		}
	}
	if siblings[9].sibling != nil {
		t.Error("Last sibling should have nil sibling")
	}
}

func TestFiber_DeepNesting(t *testing.T) {
	depth := 50
	var root *Fiber
	var current *Fiber

	for i := 0; i < depth; i++ {
		fiber := &Fiber{typeOf: "div"}
		if i == 0 {
			root = fiber
			current = fiber
		} else {
			current.child = fiber
			fiber.parent = current
			current = fiber
		}
	}

	// Verify depth
	count := 0
	current = root
	for current != nil {
		count++
		current = current.child
	}

	if count != depth {
		t.Errorf("Expected depth %d, got %d", depth, count)
	}
}

func TestFiber_ManyChildren(t *testing.T) {
	parent := &Fiber{typeOf: "parent"}

	var firstChild *Fiber
	var prevSibling *Fiber

	for i := 0; i < 100; i++ {
		child := &Fiber{typeOf: "child", parent: parent}
		if i == 0 {
			firstChild = child
			parent.child = child
		} else {
			prevSibling.sibling = child
		}
		prevSibling = child
	}

	// Count children
	count := 0
	current := firstChild
	for current != nil {
		count++
		current = current.sibling
	}

	if count != 100 {
		t.Errorf("Expected 100 children, got %d", count)
	}
}

func TestFiber_NoEffectTag(t *testing.T) {
	fiber := &Fiber{}

	if fiber.effectTag != "" {
		t.Error("Expected effectTag to be empty")
	}
}
