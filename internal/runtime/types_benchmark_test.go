package runtime

import "testing"

func BenchmarkFiberAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = &Fiber{
			typeOf: "div",
			props:  map[string]interface{}{"id": "node"},
			dirty:  true,
		}
	}
}

func BenchmarkElementLiteralCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = &Element{
			Type:        "TEXT_ELEMENT",
			TextContent: "payload",
			Children:    emptyChildren,
		}
	}
}

func BenchmarkHooksPackedStateRead(b *testing.B) {
	hooks := &Hooks{
		states: []interface{}{1, 1, 2, 2, 3, 3, 4, 4},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hooks.states[0]
		_ = hooks.states[2]
		_ = hooks.states[4]
		_ = hooks.states[6]
	}
}

func BenchmarkFetchStateCopy(b *testing.B) {
	state := FetchState{Data: "payload", Error: "", Loading: true}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = state
	}
}
