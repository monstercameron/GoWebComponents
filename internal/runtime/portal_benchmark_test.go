package runtime

import "testing"

func BenchmarkRenderPortalToSelector(b *testing.B) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	app := adapter.CreateElement("div")
	overlay := adapter.CreateElement("div")
	adapter.selectorResults["#overlay-root"] = overlay
	element := CreateElement("section", nil,
		CreateElement("p", map[string]interface{}{"id": "inline"}, "inline"),
		CreateElement(PortalNodeType, map[string]interface{}{"portalTargetSelector": "#overlay-root"},
			CreateElement("div", map[string]interface{}{"id": "portaled"}, "overlay"),
		),
	)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.Render(element, app)
		for len(scheduler.timeouts) > 0 {
			callbacks := append([]func(){}, scheduler.timeouts...)
			scheduler.timeouts = scheduler.timeouts[:0]
			for _, callback := range callbacks {
				callback()
			}
		}
	}
}