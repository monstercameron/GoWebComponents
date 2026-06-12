package runtime

import "testing"

func BenchmarkRenderPortalToSelector(parseB *testing.B) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseApp := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#overlay-root"] = parseOverlay
	parseElement := CreateElement("section", nil,
		CreateElement("p", map[string]any{"id": "inline"}, "inline"),
		CreateElement(PortalNodeType, map[string]any{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]any{"id": "portaled"}, "overlay"),
		),
	)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.Render(parseElement, parseApp)
		for len(parseScheduler.timeouts) > 0 {
			parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
			parseScheduler.timeouts = parseScheduler.timeouts[:0]
			for _, parseCallback := range parseCallbacks {
				parseCallback()
			}
		}
	}
}
