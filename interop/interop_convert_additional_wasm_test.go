//go:build js && wasm
// +build js,wasm

package interop

import (
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

// TestStructuredPayloadHelpersWasmConvertNestedValues verifies structured payload conversion, omitempty handling, and cycle detection.
func TestStructuredPayloadHelpersWasmConvertNestedValues(parseT *testing.T) {
	type parseNestedPayload struct {
		Count int `json:"count"`
	}
	type parseStructuredPayload struct {
		Visible string                 `json:"visible"`
		Skip    string                 `json:"skip,omitempty"`
		Nested  *parseNestedPayload    `json:"nested,omitempty"`
		Labels  []string               `json:"labels"`
		Meta    map[string]interface{} `json:"meta"`
		Hidden  string                 `json:"-"`
	}

	parseConverted, parseErr := goValueToJSStructured("Structured", "payload", parseStructuredPayload{
		Visible: "ready",
		Nested:  &parseNestedPayload{Count: 3},
		Labels:  []string{"alpha", "beta"},
		Meta:    map[string]interface{}{"active": true},
		Hidden:  "secret",
	})
	if parseErr != nil {
		parseT.Fatalf("expected structured payload conversion, got %v", parseErr)
	}
	parseObject, parseOk := parseConverted.(js.Value)
	if !parseOk {
		parseT.Fatalf("expected js.Value structured payload, got %#v", parseConverted)
	}
	if parseObject.Get("visible").String() != "ready" {
		parseT.Fatalf("expected visible field, got %q", parseObject.Get("visible").String())
	}
	if !parseObject.Get("skip").IsUndefined() {
		parseT.Fatalf("expected omitempty field to be skipped, got %v", parseObject.Get("skip"))
	}
	if parseObject.Get("nested").Get("count").Int() != 3 {
		parseT.Fatalf("expected nested field conversion, got %d", parseObject.Get("nested").Get("count").Int())
	}
	if parseObject.Get("labels").Length() != 2 || parseObject.Get("labels").Index(1).String() != "beta" {
		parseT.Fatalf("expected slice conversion, got %v", parseObject.Get("labels"))
	}
	if !parseObject.Get("meta").Get("active").Bool() {
		parseT.Fatalf("expected map conversion, got %v", parseObject.Get("meta"))
	}
	if !parseObject.Get("Hidden").IsUndefined() {
		parseT.Fatalf("expected skipped field to be absent, got %v", parseObject.Get("Hidden"))
	}

	type parseCycleNode struct {
		Next *parseCycleNode `json:"next,omitempty"`
	}
	parseNode := &parseCycleNode{}
	parseNode.Next = parseNode
	if _, parseErr2 := buildStructuredJSPayload("Structured", "cycle", parseNode, map[uintptr]struct{}{}); !IsCode(parseErr2, CodeEncode) {
		parseT.Fatalf("expected cycle detection error, got %v", parseErr2)
	}
}

// TestInteropConvertHelpersWasmCoverLeafConversions verifies byte, capability, and summary helpers used by structured clone conversion.
func TestInteropConvertHelpersWasmCoverLeafConversions(parseT *testing.T) {
	if parseGot := durationMS(2500 * time.Microsecond); parseGot != 2 {
		parseT.Fatalf("durationMS(2500µs) = %d, want 2", parseGot)
	}
	if parseGot2 := durationMS(-time.Second); parseGot2 != 0 {
		parseT.Fatalf("durationMS(-1s) = %d, want 0", parseGot2)
	}

	parseBinary, parseErr := goValueToJS("Convert", "bytes", []byte{1, 2, 3, 4})
	if parseErr != nil {
		parseT.Fatalf("expected byte conversion, got %v", parseErr)
	}
	parseBinaryValue, parseOk := parseBinary.(js.Value)
	if !parseOk {
		parseT.Fatalf("expected js.Value byte conversion, got %#v", parseBinary)
	}
	parseBytes, parseBytesOk := jsValueToBytes(parseBinaryValue)
	if !parseBytesOk || !reflect.DeepEqual(parseBytes, []byte{1, 2, 3, 4}) {
		parseT.Fatalf("expected Uint8Array roundtrip, got %v (ok=%v)", parseBytes, parseBytesOk)
	}
	parseArrayBuffer := js.Global().Get("ArrayBuffer").New(2)
	parseView := js.Global().Get("Uint8Array").New(parseArrayBuffer)
	parseView.SetIndex(0, 7)
	parseView.SetIndex(1, 9)
	parseBufferBytes, parseBufferOk := jsValueToBytes(parseArrayBuffer)
	if !parseBufferOk || !reflect.DeepEqual(parseBufferBytes, []byte{7, 9}) {
		parseT.Fatalf("expected ArrayBuffer roundtrip, got %v (ok=%v)", parseBufferBytes, parseBufferOk)
	}

	parseCapabilities := clientCapabilitiesJS(ClientCapabilities{
		ProtocolVersion: "1.2.0",
		Transports:      []string{"window", "cross_tab"},
		Encodings:       []string{"json", "binary"},
		Topics:          []string{"selection"},
		MaxJSONBytes:    128,
		MaxBinaryBytes:  256,
	})
	if parseCapabilities.Get("protocolVersion").String() != "1.2.0" || parseCapabilities.Get("maxJsonBytes").Int() != 128 || parseCapabilities.Get("maxBinaryBytes").Int() != 256 {
		parseT.Fatalf("unexpected capabilities payload: %v", parseCapabilities)
	}
	if parseCapabilities.Get("transports").Length() != 2 || parseCapabilities.Get("encodings").Length() != 2 || parseCapabilities.Get("topics").Index(0).String() != "selection" {
		parseT.Fatalf("expected capability arrays, got %v", parseCapabilities)
	}

	if parseGot3 := jsValueSummary(js.Undefined()); parseGot3 != "undefined" {
		parseT.Fatalf("jsValueSummary(undefined) = %q, want undefined", parseGot3)
	}
	if parseGot4 := jsValueSummary(js.Null()); parseGot4 != "null" {
		parseT.Fatalf("jsValueSummary(null) = %q, want null", parseGot4)
	}
	if parseGot5 := jsValueSummary(js.ValueOf(true)); parseGot5 != "true" {
		parseT.Fatalf("jsValueSummary(true) = %q, want true", parseGot5)
	}
	if parseGot6 := jsValueSummary(js.ValueOf(12.5)); parseGot6 != "12.5" {
		parseT.Fatalf("jsValueSummary(12.5) = %q, want 12.5", parseGot6)
	}
	parseObject := js.Global().Get("Object").New()
	parseObject.Set("mode", "ready")
	if parseGot7 := jsValueSummary(parseObject); parseGot7 != "{\"mode\":\"ready\"}" {
		parseT.Fatalf("jsValueSummary(object) = %q, want JSON string", parseGot7)
	}
}
