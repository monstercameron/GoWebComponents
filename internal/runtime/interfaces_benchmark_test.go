package runtime

import "testing"

func BenchmarkDOMNodeInterfaceEquals(parseB *testing.B) {
	var parseNode DOMNode = &testDOMNode{tag: "div"}
	parseOther := parseNode
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseNode.Equals(parseOther)
	}
}

func BenchmarkDOMAdapterInterfaceCreateElement(parseB *testing.B) {
	var parseAdapter DOMAdapter = newTestDOMAdapter()
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseAdapter.CreateElement("div")
	}
}

func BenchmarkSchedulerInterfaceSetTimeout(parseB *testing.B) {
	var parseScheduler Scheduler = newTestScheduler()
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseScheduler.SetTimeout(func() {}, 0)
	}
}
