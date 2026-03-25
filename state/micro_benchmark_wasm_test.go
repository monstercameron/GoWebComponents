//go:build js && wasm
// +build js,wasm

package state

import "testing"

func BenchmarkMarshalSnapshotJSONMicro(b *testing.B) {
	snapshot := Snapshot{
		"count": 42,
		"user":  "bench",
		"flags": map[string]interface{}{
			"darkMode": true,
			"beta":     false,
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := MarshalSnapshotJSON(snapshot)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalSnapshotJSONMicro(b *testing.B) {
	payload, err := MarshalSnapshotJSON(Snapshot{
		"count": 42,
		"user":  "bench",
		"flags": map[string]interface{}{
			"darkMode": true,
			"beta":     false,
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, decodeErr := UnmarshalSnapshotJSON(payload)
		if decodeErr != nil {
			b.Fatal(decodeErr)
		}
	}
}
