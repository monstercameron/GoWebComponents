package runtime

import "testing"

func BenchmarkFiberAllocation(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = &Fiber{
			typeOf: "div",
			props:  map[string]any{"id": "node"},
			dirty:  true,
		}
	}
}

func BenchmarkElementLiteralCreation(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = &Element{
			Type:        "TEXT_ELEMENT",
			TextContent: "payload",
			Children:    emptyChildren,
		}
	}
}

func BenchmarkHooksPackedStateRead(parseB *testing.B) {
	parseHooks := &Hooks{
		states: []any{1, 1, 2, 2, 3, 3, 4, 4},
	}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseHooks.states[0]
		_ = parseHooks.states[2]
		_ = parseHooks.states[4]
		_ = parseHooks.states[6]
	}
}

func BenchmarkFetchStateCopy(parseB *testing.B) {
	parseState := FetchState{Data: "payload", Error: "", Loading: true}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseState
	}
}
