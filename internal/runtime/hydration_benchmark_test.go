package runtime

import "testing"

func BenchmarkHydrateSimpleReuse(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		adapter := newTestDOMAdapter()
		scheduler := newTestScheduler()
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

		container := adapter.CreateElement("div")
		serverNode := adapter.CreateElement("section")
		adapter.SetAttribute(serverNode, "id", "hero")
		serverText := adapter.CreateTextNode("Hello")
		adapter.AppendChild(serverNode, serverText)
		adapter.AppendChild(container, serverNode)

		rt.Hydrate(CreateElement("section", map[string]interface{}{"id": "hero"}, "Hello"), container)
		if len(scheduler.timeouts) == 0 {
			b.Fatal("expected scheduled hydration work")
		}
		scheduler.timeouts[0]()
	}
}

func BenchmarkHydrateTagMismatchFallback(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		adapter := newTestDOMAdapter()
		scheduler := newTestScheduler()
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

		container := adapter.CreateElement("div")
		adapter.AppendChild(container, adapter.CreateElement("span"))

		rt.Hydrate(CreateElement("div", map[string]interface{}{"id": "client"}), container)
		if len(scheduler.timeouts) == 0 {
			b.Fatal("expected scheduled hydration work")
		}
		scheduler.timeouts[0]()
	}
}
