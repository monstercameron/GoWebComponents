package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

type binaryPropsPayload struct {
	Title string
	Count int
}

// TestBuildBinaryPropsValueRoundTripsSerializablePayload verifies props use the same binary value graph as source values.
func TestBuildBinaryPropsValueRoundTripsSerializablePayload(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinaryPropsValue(binaryPropsPayload{
		Title: "Orders",
		Count: 3,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryPropsValue returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseBinaryPropsValue(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryPropsValue returned error: %v", parseErr)
	}
	parseMap, parseOk := parseDecoded.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected decoded props map, got %#v", parseDecoded)
	}
	if parseMap["Title"] != "Orders" {
		parseT.Fatalf("expected Title Orders, got %#v", parseMap["Title"])
	}
	if parseMap["Count"] != float64(3) {
		parseT.Fatalf("expected Count 3, got %#v", parseMap["Count"])
	}
}

// TestBuildBinaryPropsValueRejectsNonSerializablePayload verifies props validation fails before binary encoding for unsupported values.
func TestBuildBinaryPropsValueRejectsNonSerializablePayload(parseT *testing.T) {
	if _, parseErr := runtime2.BuildBinaryPropsValue(map[string]any{
		"callback": func() {},
	}); parseErr == nil {
		parseT.Fatal("expected BuildBinaryPropsValue to reject non-serializable props")
	}
}
