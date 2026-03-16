package runtime

import "testing"

func contextText(value string) *Element {
	return &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: value,
		Props:       map[string]interface{}{"nodeValue": value},
		Children:    emptyChildren,
	}
}

func TestResolveContextValueUsesDefaultWhenMissing(t *testing.T) {
	descriptor := NewContextDescriptor("fallback")

	value := resolveContextValue(&Fiber{}, descriptor)
	if value != "fallback" {
		t.Fatalf("expected default value fallback, got %v", value)
	}
}

func TestDeriveContextValuesCopiesAndOverrides(t *testing.T) {
	first := NewContextDescriptor("a")
	second := NewContextDescriptor("b")

	parent := map[int64]interface{}{
		first.ID: "outer",
	}

	derived := deriveContextValues(parent, second.ID, "inner")
	derivedOverride := deriveContextValues(derived, first.ID, "override")

	if parent[first.ID] != "outer" {
		t.Fatalf("expected parent value to remain outer, got %v", parent[first.ID])
	}
	if derived[second.ID] != "inner" {
		t.Fatalf("expected derived second context value inner, got %v", derived[second.ID])
	}
	if derivedOverride[first.ID] != "override" {
		t.Fatalf("expected override value, got %v", derivedOverride[first.ID])
	}
}

func TestContextProviderMakesValueAvailableToFunctionChild(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	descriptor := NewContextDescriptor("fallback")
	provider := NewContextProviderType(descriptor)
	reader := func(props map[string]interface{}) *Element {
		value, _ := GoUseContextValue(descriptor).(string)
		return contextText(value)
	}

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(provider, map[string]interface{}{"value": "provided"}, CreateElement(reader, nil)),
			},
		},
		dirty: true,
	}

	rt.performUnitOfWork(root)
	providerFiber := root.child
	if providerFiber == nil {
		t.Fatal("expected provider fiber to be created")
	}

	rt.performUnitOfWork(providerFiber)
	readerFiber := providerFiber.child
	if readerFiber == nil {
		t.Fatal("expected reader fiber to be created")
	}

	rt.performUnitOfWork(readerFiber)
	textFiber := readerFiber.child
	if textFiber == nil {
		t.Fatal("expected text fiber to be created")
	}

	if textFiber.textContent != "provided" {
		t.Fatalf("expected provider value to reach consumer, got %q", textFiber.textContent)
	}
	if got := readerFiber.contextValues[descriptor.ID]; got != "provided" {
		t.Fatalf("expected reader fiber context value provided, got %v", got)
	}
}

func TestNestedContextProvidersOverrideParentValue(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	descriptor := NewContextDescriptor("fallback")
	provider := NewContextProviderType(descriptor)
	reader := func(props map[string]interface{}) *Element {
		value, _ := GoUseContextValue(descriptor).(string)
		return contextText(value)
	}

	root := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(provider, map[string]interface{}{"value": "outer"},
					CreateElement(provider, map[string]interface{}{"value": "inner"},
						CreateElement(reader, nil),
					),
				),
			},
		},
		dirty: true,
	}

	rt.performUnitOfWork(root)
	outerProvider := root.child
	rt.performUnitOfWork(outerProvider)
	innerProvider := outerProvider.child
	rt.performUnitOfWork(innerProvider)
	readerFiber := innerProvider.child
	rt.performUnitOfWork(readerFiber)
	textFiber := readerFiber.child

	if textFiber == nil {
		t.Fatal("expected nested text fiber to be created")
	}
	if textFiber.textContent != "inner" {
		t.Fatalf("expected nested provider override to win, got %q", textFiber.textContent)
	}
}

func TestContextProviderValueChangeForcesConsumerRerender(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	descriptor := NewContextDescriptor("fallback")
	provider := NewContextProviderType(descriptor)
	reader := func(props map[string]interface{}) *Element {
		value, _ := GoUseContextValue(descriptor).(string)
		return contextText(value)
	}

	firstRoot := &Fiber{
		typeOf: "ROOT",
		props: map[string]interface{}{
			"children": []interface{}{
				CreateElement(provider, map[string]interface{}{"value": "first"}, CreateElement(reader, nil)),
			},
		},
		dirty: true,
	}

	rt.performUnitOfWork(firstRoot)
	firstProvider := firstRoot.child
	rt.performUnitOfWork(firstProvider)
	firstReader := firstProvider.child
	rt.performUnitOfWork(firstReader)
	if firstReader.child == nil || firstReader.child.textContent != "first" {
		t.Fatalf("expected initial consumer render to use first value, got %#v", firstReader.child)
	}

	secondRoot := &Fiber{
		typeOf:    "ROOT",
		props:     map[string]interface{}{"children": []interface{}{CreateElement(provider, map[string]interface{}{"value": "second"}, CreateElement(reader, nil))}},
		alternate: firstRoot,
		dirty:     true,
	}

	rt.performUnitOfWork(secondRoot)
	secondProvider := secondRoot.child
	rt.performUnitOfWork(secondProvider)
	secondReader := secondProvider.child
	if secondReader == nil {
		t.Fatal("expected second reader fiber")
	}
	rt.performUnitOfWork(secondReader)

	if secondReader.child == nil || secondReader.child.textContent != "second" {
		t.Fatalf("expected consumer rerender to use updated context value, got %#v", secondReader.child)
	}
}