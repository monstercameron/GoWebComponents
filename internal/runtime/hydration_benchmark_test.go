package runtime

import "testing"

func buildHydrationListElement(parseCount int) *Element {
	parseChildren := make([]interface{}, 0, parseCount)
	for parseI := 0; parseI < parseCount; parseI++ {
		parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{
			"id": "row-" + stringifyHydrationValue(parseI),
		}, "row-"+stringifyHydrationValue(parseI)))
	}
	return CreateElement("section", map[string]interface{}{"id": "dashboard"},
		CreateElement("h2", nil, "Hydration benchmark"),
		CreateElement("ul", map[string]interface{}{"id": "rows"}, parseChildren...),
	)
}

func buildHydrationListContainer(parseAdapter *testDOMAdapter, parseCount int) DOMNode {
	parseContainer := parseAdapter.CreateElement("div")
	parseSection := parseAdapter.CreateElement("section")
	parseAdapter.SetAttribute(parseSection, "id", "dashboard")
	parseHeading := parseAdapter.CreateElement("h2")
	parseAdapter.AppendChild(parseHeading, parseAdapter.CreateTextNode("Hydration benchmark"))
	parseAdapter.AppendChild(parseSection, parseHeading)
	parseList := parseAdapter.CreateElement("ul")
	parseAdapter.SetAttribute(parseList, "id", "rows")
	for parseI := 0; parseI < parseCount; parseI++ {
		parseItem := parseAdapter.CreateElement("li")
		parseLabel := "row-" + stringifyHydrationValue(parseI)
		parseAdapter.SetAttribute(parseItem, "id", parseLabel)
		parseAdapter.AppendChild(parseItem, parseAdapter.CreateTextNode(parseLabel))
		parseAdapter.AppendChild(parseList, parseItem)
	}
	parseAdapter.AppendChild(parseSection, parseList)
	parseAdapter.AppendChild(parseContainer, parseSection)
	return parseContainer
}

func BenchmarkHydrateSimpleReuse(parseB *testing.B) {
	parseB.ReportAllocs()

	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

		parseContainer := parseAdapter.CreateElement("div")
		parseServerNode := parseAdapter.CreateElement("section")
		parseAdapter.SetAttribute(parseServerNode, "id", "hero")
		parseServerText := parseAdapter.CreateTextNode("Hello")
		parseAdapter.AppendChild(parseServerNode, parseServerText)
		parseAdapter.AppendChild(parseContainer, parseServerNode)

		parseRt.Hydrate(CreateElement("section", map[string]interface{}{"id": "hero"}, "Hello"), parseContainer)
		if len(parseScheduler.timeouts) == 0 {
			parseB.Fatal("expected scheduled hydration work")
		}
		parseScheduler.timeouts[0]()
	}
}

func BenchmarkHydrateTagMismatchFallback(parseB *testing.B) {
	parseB.ReportAllocs()

	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

		parseContainer := parseAdapter.CreateElement("div")
		parseAdapter.AppendChild(parseContainer, parseAdapter.CreateElement("span"))

		parseRt.Hydrate(CreateElement("div", map[string]interface{}{"id": "client"}), parseContainer)
		if len(parseScheduler.timeouts) == 0 {
			parseB.Fatal("expected scheduled hydration work")
		}
		parseScheduler.timeouts[0]()
	}
}

func BenchmarkHydrateMediumTreeReuse(parseB *testing.B) {
	parseB.ReportAllocs()

	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

		parseContainer := buildHydrationListContainer(parseAdapter, 180)
		parseRt.Hydrate(buildHydrationListElement(180), parseContainer)
		if len(parseScheduler.timeouts) == 0 {
			parseB.Fatal("expected scheduled hydration work")
		}
		parseScheduler.timeouts[0]()
	}
}

func BenchmarkHydrateMediumTreeFirstUpdate(parseB *testing.B) {
	parseB.ReportAllocs()

	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

		parseContainer := buildHydrationListContainer(parseAdapter, 120)
		var setTitle func(interface{})
		parseComponent := func() *Element {
			parseTitle, set := GoUseState(parseRt, "Hydration benchmark")
			setTitle = set
			return CreateElement("section", map[string]interface{}{"id": "dashboard"},
				CreateElement("h2", nil, parseTitle()),
				CreateElement("ul", map[string]interface{}{"id": "rows"},
					func() []interface{} {
						parseChildren := make([]interface{}, 0, 120)
						for parseI2 := 0; parseI2 < 120; parseI2++ {
							parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{
								"id": "row-" + stringifyHydrationValue(parseI2),
							}, "row-"+stringifyHydrationValue(parseI2)))
						}
						return parseChildren
					}()...,
				),
			)
		}

		parseRt.Hydrate(CreateElement(parseComponent, nil), parseContainer)
		if len(parseScheduler.timeouts) == 0 {
			parseB.Fatal("expected scheduled hydration work")
		}
		parseScheduler.timeouts[0]()
		if setTitle == nil {
			parseB.Fatal("expected hydrated component setter")
		}
		setTitle("Hydrated update")
		if len(parseScheduler.timeouts) == 0 {
			parseB.Fatal("expected scheduled post-hydration update")
		}
		parseScheduler.timeouts[0]()
	}
}
