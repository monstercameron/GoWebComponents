package html

import "testing"

func BenchmarkTagDivWithCommonProps(parseB *testing.B) {
	parseB.ReportAllocs()
	parseProps := Props{
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
	for parseB.Loop() {
		parseNode := Tag("div", parseProps, Text("hello"), Text("world"))
		if parseNode == nil {
			parseB.Fatal("Tag returned nil")
		}
	}
}

func BenchmarkCompactPropsClassOnly(parseB *testing.B) {
	parseB.ReportAllocs()
	parseProps := Props{Class: "row"}
	for parseB.Loop() {
		_, parseAttrs, parseOK := toRuntimeCompactProps(&parseProps, nil)
		if !parseOK || len(parseAttrs) != 1 {
			parseB.Fatal("class-only props missed compact lane")
		}
	}
}

func BenchmarkCompactPropsClassData(parseB *testing.B) {
	parseB.ReportAllocs()
	parseProps := Props{Class: "row", Data: map[string]string{"row-id": "42", "state": "ready"}}
	for parseB.Loop() {
		_, parseAttrs, parseOK := toRuntimeCompactProps(&parseProps, nil)
		if !parseOK || len(parseAttrs) != 3 {
			parseB.Fatal("class/data props missed compact lane")
		}
	}
}

func BenchmarkCustomElementWithAttributesAndProperties(parseB *testing.B) {
	parseB.ReportAllocs()
	parseProps := CustomElementProps{
		Props: Props{
			Class: "widget",
		},
		Attributes: map[string]string{
			"mode": "compact",
		},
		Presence: map[string]bool{
			"hydrated": true,
		},
		Properties: map[string]any{
			"value": 42,
		},
	}
	for parseB.Loop() {
		parseNode := CustomElement("x-widget", parseProps, Text("payload"))
		if parseNode == nil {
			parseB.Fatal("CustomElement returned nil")
		}
	}
}
