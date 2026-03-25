package mockdom

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func BenchmarkMockDOMAdapterCreateElementAndAppendChild(parseB *testing.B) {
	parseB.ReportAllocs()
	parseAdapter := NewMockDOMAdapter()
	parseRoot := parseAdapter.CreateElement("div")
	for parseB.Loop() {
		parseChild := parseAdapter.CreateElement("span")
		parseAdapter.AppendChild(parseRoot, parseChild)
	}
}

func BenchmarkMockSchedulerFlushAll(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseScheduler := NewMockScheduler(false)
		for parseI := 0; parseI < 16; parseI++ {
			parseScheduler.RequestIdleCallback(func(parseDeadline runtime.Deadline) {
				_ = parseDeadline.TimeRemaining()
			})
			parseScheduler.SetTimeout(func() {}, 0)
		}
		parseScheduler.FlushAll()
	}
}
