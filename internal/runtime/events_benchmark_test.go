//go:build js && wasm

package runtime

import (
	"syscall/js"
	"testing"
)

func benchmarkGoEvent() GoEvent {
	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("value", "payload")
	parseTarget.Set("checked", true)

	parseEventValue := js.Global().Get("Object").New()
	parseEventValue.Set("target", parseTarget)
	parseEventValue.Set("key", "Enter")
	parseEventValue.Set("keyCode", 13)
	return NewGoEvent(parseEventValue)
}

func BenchmarkGoEventGetValue(parseB *testing.B) {
	parseEvent := benchmarkGoEvent()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseEvent.GetValue()
	}
}

func BenchmarkGoEventIsChecked(parseB *testing.B) {
	parseEvent := benchmarkGoEvent()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseEvent.IsChecked()
	}
}

func BenchmarkGoEventGetKey(parseB *testing.B) {
	parseEvent := benchmarkGoEvent()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseEvent.GetKey()
	}
}

func BenchmarkGoEventGetKeyCode(parseB *testing.B) {
	parseEvent := benchmarkGoEvent()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseEvent.GetKeyCode()
	}
}

func BenchmarkGoEventGetTarget(parseB *testing.B) {
	parseEvent := benchmarkGoEvent()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseEvent.GetTarget()
	}
}
