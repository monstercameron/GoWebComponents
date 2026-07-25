package mockdom

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
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
		for range 16 {
			parseScheduler.RequestIdleCallback(func(parseDeadline runtime.Deadline) {
				_ = parseDeadline.TimeRemaining()
			})
			parseScheduler.SetTimeout(func() {}, 0)
		}
		parseScheduler.FlushAll()
	}
}
