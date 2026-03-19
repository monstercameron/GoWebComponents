package runtime

import "testing"

func buildHydrationListElement(count int) *Element {
	children := make([]interface{}, 0, count)
	for i := 0; i < count; i++ {
		children = append(children, CreateElement("li", map[string]interface{}{
			"id": "row-" + stringifyHydrationValue(i),
		}, "row-"+stringifyHydrationValue(i)))
	}
	return CreateElement("section", map[string]interface{}{"id": "dashboard"},
		CreateElement("h2", nil, "Hydration benchmark"),
		CreateElement("ul", map[string]interface{}{"id": "rows"}, children...),
	)
}

func buildHydrationListContainer(adapter *testDOMAdapter, count int) DOMNode {
	container := adapter.CreateElement("div")
	section := adapter.CreateElement("section")
	adapter.SetAttribute(section, "id", "dashboard")
	heading := adapter.CreateElement("h2")
	adapter.AppendChild(heading, adapter.CreateTextNode("Hydration benchmark"))
	adapter.AppendChild(section, heading)
	list := adapter.CreateElement("ul")
	adapter.SetAttribute(list, "id", "rows")
	for i := 0; i < count; i++ {
		item := adapter.CreateElement("li")
		label := "row-" + stringifyHydrationValue(i)
		adapter.SetAttribute(item, "id", label)
		adapter.AppendChild(item, adapter.CreateTextNode(label))
		adapter.AppendChild(list, item)
	}
	adapter.AppendChild(section, list)
	adapter.AppendChild(container, section)
	return container
}

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

func BenchmarkHydrateMediumTreeReuse(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		adapter := newTestDOMAdapter()
		scheduler := newTestScheduler()
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

		container := buildHydrationListContainer(adapter, 180)
		rt.Hydrate(buildHydrationListElement(180), container)
		if len(scheduler.timeouts) == 0 {
			b.Fatal("expected scheduled hydration work")
		}
		scheduler.timeouts[0]()
	}
}

func BenchmarkHydrateMediumTreeFirstUpdate(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		adapter := newTestDOMAdapter()
		scheduler := newTestScheduler()
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

		container := buildHydrationListContainer(adapter, 120)
		var setTitle func(interface{})
		component := func() *Element {
			title, set := GoUseState(rt, "Hydration benchmark")
			setTitle = set
			return CreateElement("section", map[string]interface{}{"id": "dashboard"},
				CreateElement("h2", nil, title()),
				CreateElement("ul", map[string]interface{}{"id": "rows"},
					func() []interface{} {
						children := make([]interface{}, 0, 120)
						for i := 0; i < 120; i++ {
							children = append(children, CreateElement("li", map[string]interface{}{
								"id": "row-" + stringifyHydrationValue(i),
							}, "row-"+stringifyHydrationValue(i)))
						}
						return children
					}()...,
				),
			)
		}

		rt.Hydrate(CreateElement(component, nil), container)
		if len(scheduler.timeouts) == 0 {
			b.Fatal("expected scheduled hydration work")
		}
		scheduler.timeouts[0]()
		if setTitle == nil {
			b.Fatal("expected hydrated component setter")
		}
		setTitle("Hydrated update")
		if len(scheduler.timeouts) == 0 {
			b.Fatal("expected scheduled post-hydration update")
		}
		scheduler.timeouts[0]()
	}
}
