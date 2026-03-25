package ui

import "testing"

func BenchmarkRenderToStringMicro(b *testing.B) {
	root := Fragment(
		Text("Benchmark Header"),
		Fragment(
			Text("alpha"),
			Text("beta"),
			Text("gamma"),
		),
	)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		markup, err := RenderToString(root)
		if err != nil {
			b.Fatal(err)
		}
		if markup == "" {
			b.Fatal("expected non-empty markup")
		}
	}
}

func BenchmarkMarshalUnmarshalSSRBootstrapMicro(b *testing.B) {
	payload := SSRBootstrap{
		Version:       CurrentSSRBootstrapVersion,
		CorrelationID: "bench-correlation",
		Route: SSRRouteBootstrap{
			Path: "/bench",
			Query: map[string][]string{
				"tab": {"perf"},
			},
			Params: map[string]string{
				"id": "42",
			},
		},
		Atoms: map[string]interface{}{
			"counter": 42,
			"theme":   "dark",
		},
		Data: map[string]interface{}{
			"featureFlags": map[string]interface{}{
				"beta": true,
			},
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		encoded, err := MarshalSSRBootstrap(payload)
		if err != nil {
			b.Fatal(err)
		}
		decoded, err := UnmarshalSSRBootstrap(encoded)
		if err != nil {
			b.Fatal(err)
		}
		if decoded.Version == 0 {
			b.Fatal("expected normalized bootstrap payload")
		}
	}
}
