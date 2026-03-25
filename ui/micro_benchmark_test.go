package ui

import "testing"

func BenchmarkRenderToStringMicro(parseB *testing.B) {
	parseRoot := Fragment(
		Text("Benchmark Header"),
		Fragment(
			Text("alpha"),
			Text("beta"),
			Text("gamma"),
		),
	)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseMarkup, parseErr := RenderToString(parseRoot)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if parseMarkup == "" {
			parseB.Fatal("expected non-empty markup")
		}
	}
}

func BenchmarkMarshalUnmarshalSSRBootstrapMicro(parseB *testing.B) {
	parsePayload := SSRBootstrap{
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

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseEncoded, parseErr := MarshalSSRBootstrap(parsePayload)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		parseDecoded, parseErr := UnmarshalSSRBootstrap(parseEncoded)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if parseDecoded.Version == 0 {
			parseB.Fatal("expected normalized bootstrap payload")
		}
	}
}
