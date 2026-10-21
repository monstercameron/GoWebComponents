//go:build js && wasm

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
	for parseB.Loop() {
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
	for parseB.Loop() {
		_, parseDecodeErr := UnmarshalSnapshotJSON(parsePayload)
		if parseDecodeErr != nil {
			parseB.Fatal(parseDecodeErr)
		}
	}
}
