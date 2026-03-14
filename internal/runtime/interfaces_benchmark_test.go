package runtime

import "testing"

func BenchmarkDOMNodeInterfaceEquals(b *testing.B) {
	var node DOMNode = &testDOMNode{tag: "div"}
	other := node
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = node.Equals(other)
	}
}

func BenchmarkDOMAdapterInterfaceCreateElement(b *testing.B) {
	var adapter DOMAdapter = newTestDOMAdapter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = adapter.CreateElement("div")
	}
}

func BenchmarkSchedulerInterfaceSetTimeout(b *testing.B) {
	var scheduler Scheduler = newTestScheduler()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scheduler.SetTimeout(func() {}, 0)
	}
}
