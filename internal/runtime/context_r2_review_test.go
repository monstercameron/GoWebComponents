package runtime

import "testing"

// pumpContextReviewWork drains the scheduler until no timeout is queued.
func pumpContextReviewWork(parseScheduler *testScheduler) {
	for parseGuard := 0; parseGuard < 50; parseGuard++ {
		parseWork := parseScheduler.getPendingTimeouts()
		if len(parseWork) == 0 {
			return
		}
		for _, parseFn := range parseWork {
			parseFn()
		}
	}
}

// TestContextValueChangeReachesDeepMemoizedConsumer pins that a provider value
// change re-renders a consumer that sits behind a host element and an element
// subtree whose identity never changes (the memoized-children shape).
func TestContextValueChangeReachesDeepMemoizedConsumer(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, FrameBudgetMs: -1})

	parseDescriptor := NewContextDescriptor("fallback")
	parseProvider := NewContextProviderType(parseDescriptor)
	parseConsumer := func() *Element {
		parseValue, _ := GoUseContextValue(parseDescriptor).(string)
		return CreateElement("b", nil, parseValue)
	}
	// Built once: every parent render hands the provider the SAME child element.
	parseStatic := CreateElement("section", nil, CreateElement("div", nil, CreateElement(parseConsumer, nil)))

	var parseSet func(any)
	parseApp := func() *Element {
		parseValue, parseSetValue := GoUseState(parseRt, "first")
		parseSet = parseSetValue
		return CreateElement(parseProvider, map[string]any{"value": parseValue()}, parseStatic)
	}

	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	pumpContextReviewWork(parseScheduler)
	parseSet("second")
	pumpContextReviewWork(parseScheduler)
	parseSet("third")
	pumpContextReviewWork(parseScheduler)

	parseSection := parseAdapter.GetChildren(parseContainer)[0]
	parseDiv := parseAdapter.GetChildren(parseSection)[0]
	parseB := parseAdapter.GetChildren(parseDiv)[0]
	parseText := parseAdapter.GetChildren(parseB)[0].(*testDOMNode).text
	if parseText != "third" {
		parseT.Fatalf("deep consumer kept a stale context value: got %q, want %q", parseText, "third")
	}
}
