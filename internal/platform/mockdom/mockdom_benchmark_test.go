package mockdom

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func BenchmarkMockDOMAdapterCreateElementAndAppendChild(b *testing.B) {
	b.ReportAllocs()
	adapter := NewMockDOMAdapter()
	root := adapter.CreateElement("div")
	for b.Loop() {
		child := adapter.CreateElement("span")
		adapter.AppendChild(root, child)
	}
}

func BenchmarkMockSchedulerFlushAll(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		scheduler := NewMockScheduler(false)
		for i := 0; i < 16; i++ {
			scheduler.RequestIdleCallback(func(deadline runtime.Deadline) {
				_ = deadline.TimeRemaining()
			})
			scheduler.SetTimeout(func() {}, 0)
		}
		scheduler.FlushAll()
	}
}
