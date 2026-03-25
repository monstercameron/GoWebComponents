//go:build js && wasm
// +build js,wasm

package state

import "testing"

func BenchmarkMarshalSnapshotJSONMicro(parseB *testing.B) {
	parseSnapshot := Snapshot{
		"count": 42,
		"user":  "bench",
		"flags": map[string]interface{}{
			"darkMode": true,
			"beta":     false,
		},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, parseErr := MarshalSnapshotJSON(parseSnapshot)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
	}
}

func BenchmarkUnmarshalSnapshotJSONMicro(parseB *testing.B) {
	parsePayload, parseErr := MarshalSnapshotJSON(Snapshot{
		"count": 42,
		"user":  "bench",
		"flags": map[string]interface{}{
			"darkMode": true,
			"beta":     false,
		},
	})
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, parseDecodeErr := UnmarshalSnapshotJSON(parsePayload)
		if parseDecodeErr != nil {
			parseB.Fatal(parseDecodeErr)
		}
	}
}
