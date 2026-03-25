package interop

import (
	"errors"
	"testing"
)

type benchmarkDecodePayload struct {
	ID      string            `json:"id"`
	Enabled bool              `json:"enabled"`
	Meta    map[string]string `json:"meta"`
}

func BenchmarkDecodeMapToStruct(b *testing.B) {
	b.ReportAllocs()
	value := map[string]interface{}{
		"id":      "bench",
		"enabled": true,
		"meta": map[string]interface{}{
			"lane":  "interop",
			"stage": "decode",
		},
	}
	for b.Loop() {
		var payload benchmarkDecodePayload
		if err := Decode(value, &payload); err != nil {
			b.Fatalf("Decode: %v", err)
		}
	}
}

func BenchmarkInteropErrorString(b *testing.B) {
	b.ReportAllocs()
	err := &Error{
		Op:     "OpenPersistentStore",
		Target: "indexedDB",
		Code:   CodeInvalid,
		Err:    errors.New("missing store name"),
	}
	for b.Loop() {
		value := err.Error()
		if value == "" {
			b.Fatal("Error() returned empty string")
		}
	}
}
