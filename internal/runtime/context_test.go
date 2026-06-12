package runtime

import "testing"

func contextText(parseValue string) *Element {
	return &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: parseValue,
		Props:       map[string]any{"nodeValue": parseValue},
		Children:    emptyChildren,
	}
}

func TestResolveContextValueUsesDefaultWhenMissing(parseT *testing.T) {
	parseDescriptor := NewContextDescriptor("fallback")

	parseValue := resolveContextValue(&Fiber{}, parseDescriptor)
	if parseValue != "fallback" {
		parseT.Fatalf("expected default value fallback, got %v", parseValue)
	}
}

func TestDeriveContextValuesCopiesAndOverrides(parseT *testing.T) {
	parseFirst := NewContextDescriptor("a")
	parseSecond := NewContextDescriptor("b")

	parseParent := map[int64]any{
		parseFirst.ID: "outer",
	}

	parseDerived := deriveContextValues(parseParent, parseSecond.ID, "inner")
	parseDerivedOverride := deriveContextValues(parseDerived, parseFirst.ID, "override")

	if parseParent[parseFirst.ID] != "outer" {
		parseT.Fatalf("expected parent value to remain outer, got %v", parseParent[parseFirst.ID])
	}
	if parseDerived[parseSecond.ID] != "inner" {
		parseT.Fatalf("expected derived second context value inner, got %v", parseDerived[parseSecond.ID])
	}
	if parseDerivedOverride[parseFirst.ID] != "override" {
		parseT.Fatalf("expected override value, got %v", parseDerivedOverride[parseFirst.ID])
	}
}

func TestContextProviderMakesValueAvailableToFunctionChild(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDescriptor := NewContextDescriptor("fallback")
	parseProvider := NewContextProviderType(parseDescriptor)
	parseReader := func(parseProps map[string]any) *Element {
		parseValue, _ := GoUseContextValue(parseDescriptor).(string)
		return contextText(parseValue)
	}

	parseRoot := &Fiber{
		typeOf: "ROOT",
		props: map[string]any{
			"children": []any{
				CreateElement(parseProvider, map[string]any{"value": "provided"}, CreateElement(parseReader, nil)),
			},
		},
		dirty: true,
	}

	parseRt.performUnitOfWork(parseRoot)
	parseProviderFiber := parseRoot.child
	if parseProviderFiber == nil {
		parseT.Fatal("expected provider fiber to be created")
	}

	parseRt.performUnitOfWork(parseProviderFiber)
	parseReaderFiber := parseProviderFiber.child
	if parseReaderFiber == nil {
		parseT.Fatal("expected reader fiber to be created")
	}

	parseRt.performUnitOfWork(parseReaderFiber)
	parseTextFiber := parseReaderFiber.child
	if parseTextFiber == nil {
		parseT.Fatal("expected text fiber to be created")
	}

	if parseTextFiber.textContent != "provided" {
		parseT.Fatalf("expected provider value to reach consumer, got %q", parseTextFiber.textContent)
	}
	if parseGot := parseReaderFiber.contextValues[parseDescriptor.ID]; parseGot != "provided" {
		parseT.Fatalf("expected reader fiber context value provided, got %v", parseGot)
	}
}

func TestNestedContextProvidersOverrideParentValue(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDescriptor := NewContextDescriptor("fallback")
	parseProvider := NewContextProviderType(parseDescriptor)
	parseReader := func(parseProps map[string]any) *Element {
		parseValue, _ := GoUseContextValue(parseDescriptor).(string)
		return contextText(parseValue)
	}

	parseRoot := &Fiber{
		typeOf: "ROOT",
		props: map[string]any{
			"children": []any{
				CreateElement(parseProvider, map[string]any{"value": "outer"},
					CreateElement(parseProvider, map[string]any{"value": "inner"},
						CreateElement(parseReader, nil),
					),
				),
			},
		},
		dirty: true,
	}

	parseRt.performUnitOfWork(parseRoot)
	parseOuterProvider := parseRoot.child
	parseRt.performUnitOfWork(parseOuterProvider)
	parseInnerProvider := parseOuterProvider.child
	parseRt.performUnitOfWork(parseInnerProvider)
	parseReaderFiber := parseInnerProvider.child
	parseRt.performUnitOfWork(parseReaderFiber)
	parseTextFiber := parseReaderFiber.child

	if parseTextFiber == nil {
		parseT.Fatal("expected nested text fiber to be created")
	}
	if parseTextFiber.textContent != "inner" {
		parseT.Fatalf("expected nested provider override to win, got %q", parseTextFiber.textContent)
	}
}

func TestContextProviderValueChangeForcesConsumerRerender(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDescriptor := NewContextDescriptor("fallback")
	parseProvider := NewContextProviderType(parseDescriptor)
	parseReader := func(parseProps map[string]any) *Element {
		parseValue, _ := GoUseContextValue(parseDescriptor).(string)
		return contextText(parseValue)
	}

	parseFirstRoot := &Fiber{
		typeOf: "ROOT",
		props: map[string]any{
			"children": []any{
				CreateElement(parseProvider, map[string]any{"value": "first"}, CreateElement(parseReader, nil)),
			},
		},
		dirty: true,
	}

	parseRt.performUnitOfWork(parseFirstRoot)
	parseFirstProvider := parseFirstRoot.child
	parseRt.performUnitOfWork(parseFirstProvider)
	parseFirstReader := parseFirstProvider.child
	parseRt.performUnitOfWork(parseFirstReader)
	if parseFirstReader.child == nil || parseFirstReader.child.textContent != "first" {
		parseT.Fatalf("expected initial consumer render to use first value, got %#v", parseFirstReader.child)
	}

	parseSecondRoot := &Fiber{
		typeOf:    "ROOT",
		props:     map[string]any{"children": []any{CreateElement(parseProvider, map[string]any{"value": "second"}, CreateElement(parseReader, nil))}},
		alternate: parseFirstRoot,
		dirty:     true,
	}

	parseRt.performUnitOfWork(parseSecondRoot)
	parseSecondProvider := parseSecondRoot.child
	parseRt.performUnitOfWork(parseSecondProvider)
	parseSecondReader := parseSecondProvider.child
	if parseSecondReader == nil {
		parseT.Fatal("expected second reader fiber")
	}
	parseRt.performUnitOfWork(parseSecondReader)

	if parseSecondReader.child == nil || parseSecondReader.child.textContent != "second" {
		parseT.Fatalf("expected consumer rerender to use updated context value, got %#v", parseSecondReader.child)
	}
}
