//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
)

func benchmarkGoEvent() GoEvent {
	target := js.Global().Get("Object").New()
	target.Set("value", "payload")
	target.Set("checked", true)

	eventValue := js.Global().Get("Object").New()
	eventValue.Set("target", target)
	eventValue.Set("key", "Enter")
	eventValue.Set("keyCode", 13)
	return NewGoEvent(eventValue)
}

func BenchmarkGoEventGetValue(b *testing.B) {
	event := benchmarkGoEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = event.GetValue()
	}
}

func BenchmarkGoEventIsChecked(b *testing.B) {
	event := benchmarkGoEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = event.IsChecked()
	}
}

func BenchmarkGoEventGetKey(b *testing.B) {
	event := benchmarkGoEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = event.GetKey()
	}
}

func BenchmarkGoEventGetKeyCode(b *testing.B) {
	event := benchmarkGoEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = event.GetKeyCode()
	}
}

func BenchmarkGoEventGetTarget(b *testing.B) {
	event := benchmarkGoEvent()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = event.GetTarget()
	}
}
