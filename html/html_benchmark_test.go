package html

import "testing"

func BenchmarkTagDivWithCommonProps(b *testing.B) {
	b.ReportAllocs()
	props := Props{
		ID:    "bench-node",
		Class: "rounded-xl border border-slate-300 bg-white px-4 py-2",
		Data: map[string]string{
			"lane":  "perf",
			"stage": "bench",
		},
		Aria: map[string]string{
			"label": "bench node",
		},
	}
	for b.Loop() {
		node := Tag("div", props, Text("hello"), Text("world"))
		if node == nil {
			b.Fatal("Tag returned nil")
		}
	}
}

func BenchmarkCustomElementWithAttributesAndProperties(b *testing.B) {
	b.ReportAllocs()
	props := CustomElementProps{
		Props: Props{
			Class: "widget",
		},
		Attributes: map[string]string{
			"mode": "compact",
		},
		Presence: map[string]bool{
			"hydrated": true,
		},
		Properties: map[string]interface{}{
			"value": 42,
		},
	}
	for b.Loop() {
		node := CustomElement("x-widget", props, Text("payload"))
		if node == nil {
			b.Fatal("CustomElement returned nil")
		}
	}
}
