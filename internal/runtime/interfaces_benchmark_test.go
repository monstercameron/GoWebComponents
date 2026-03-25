package runtime

import "testing"

func BenchmarkDOMNodeInterfaceEquals(parseB *testing.B) {
	var parseNode DOMNode = &testDOMNode{tag: "div"}
	parseOther := parseNode
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseNode.Equals(parseOther)
	}
}

func BenchmarkDOMAdapterInterfaceCreateElement(parseB *testing.B) {
	var parseAdapter DOMAdapter = newTestDOMAdapter()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseAdapter.CreateElement("div")
	}
}

func BenchmarkSchedulerInterfaceSetTimeout(parseB *testing.B) {
	var parseScheduler Scheduler = newTestScheduler()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseScheduler.SetTimeout(func() {}, 0)
	}
}
