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

func BenchmarkDecodeMapToStruct(parseB *testing.B) {
	parseB.ReportAllocs()
	parseValue := map[string]any{
		"id":      "bench",
		"enabled": true,
		"meta": map[string]any{
			"lane":  "interop",
			"stage": "decode",
		},
	}
	for parseB.Loop() {
		var parsePayload benchmarkDecodePayload
		if parseErr := Decode(parseValue, &parsePayload); parseErr != nil {
			parseB.Fatalf("Decode: %v", parseErr)
		}
	}
}

func BenchmarkInteropErrorString(parseB *testing.B) {
	parseB.ReportAllocs()
	parseErr := &Error{
		Op:     "OpenPersistentStore",
		Target: "indexedDB",
		Code:   CodeInvalid,
		Err:    errors.New("missing store name"),
	}
	for parseB.Loop() {
		parseValue := parseErr.Error()
		if parseValue == "" {
			parseB.Fatal("Error() returned empty string")
		}
	}
}
