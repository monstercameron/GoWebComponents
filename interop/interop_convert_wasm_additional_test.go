//go:build js && wasm
// +build js,wasm

package interop

import (
	"encoding/json"
	"reflect"
	"syscall/js"
	"testing"
	"time"
)

// TestInteropConvertHelpersWasmCoverPrimitiveConversions verifies the low-level conversion helpers handle primitive wasm branches.
func TestInteropConvertHelpersWasmCoverPrimitiveConversions(parseT *testing.T) {
	if parseGot := durationMS(-time.Second); parseGot != 0 {
		parseT.Fatalf("expected negative duration to clamp to zero, got %d", parseGot)
	}
	if parseGot := durationMS(1500 * time.Millisecond); parseGot != 1500 {
		parseT.Fatalf("expected 1500ms duration conversion, got %d", parseGot)
	}

	parsePromise := js.Global().Get("Promise").Call("resolve", "ok")
	if !isPromise(parsePromise) {
		parseT.Fatal("expected Promise value to be detected")
	}
	if isPromise(js.ValueOf(7)) {
		parseT.Fatal("expected number to be excluded from promise detection")
	}

	parseArray := js.Global().Get("Uint8Array").New(3)
	parseArray.SetIndex(0, 3)
	parseArray.SetIndex(1, 7)
	parseArray.SetIndex(2, 11)
	parseRawBytes, parseOk := jsValueToBytes(parseArray.Get("buffer"))
	if !parseOk || len(parseRawBytes) != 3 || parseRawBytes[0] != 3 || parseRawBytes[2] != 11 {
		parseT.Fatalf("expected ArrayBuffer bytes, got ok=%t bytes=%v", parseOk, parseRawBytes)
	}
	parseViewBytes, parseOk := jsValueToBytes(parseArray)
	if !parseOk || len(parseViewBytes) != 3 || parseViewBytes[1] != 7 {
		parseT.Fatalf("expected typed-array bytes, got ok=%t bytes=%v", parseOk, parseViewBytes)
	}
	if parseBytes, parseOk := jsValueToBytes(js.Global().Get("Object").New()); parseOk || parseBytes != nil {
		parseT.Fatalf("expected plain object to skip byte decoding, got ok=%t bytes=%v", parseOk, parseBytes)
	}

	if parseGot := jsValueSummary(js.Undefined()); parseGot != "undefined" {
		parseT.Fatalf("expected undefined summary, got %q", parseGot)
	}
	if parseGot := jsValueSummary(js.Null()); parseGot != "null" {
		parseT.Fatalf("expected null summary, got %q", parseGot)
	}
	if parseGot := jsValueSummary(js.ValueOf(true)); parseGot != "true" {
		parseT.Fatalf("expected bool summary, got %q", parseGot)
	}
	if parseGot := jsValueSummary(js.ValueOf(9.5)); parseGot != "9.5" {
		parseT.Fatalf("expected number summary, got %q", parseGot)
	}
	parseSummaryObject := js.Global().Get("Object").New()
	parseSummaryObject.Set("kind", "demo")
	if parseGot := jsValueSummary(parseSummaryObject); parseGot != "{\"kind\":\"demo\"}" {
		parseT.Fatalf("expected JSON object summary, got %q", parseGot)
	}
}

// TestInteropConvertHelpersWasmCoverStructuredEncoding verifies structured payload helpers preserve tags, omitempty, and capability metadata.
func TestInteropConvertHelpersWasmCoverStructuredEncoding(parseT *testing.T) {
	type parseStructuredPayload struct {
		Visible string `json:"visible"`
		Omit    string `json:"omit,omitempty"`
		Skip    string `json:"-"`
		hidden  string
	}

	parsePayload := parseStructuredPayload{Visible: "shown"}
	parseStructured, parseErr := buildStructuredJSPayload("test", "payload", parsePayload, map[uintptr]struct{}{})
	if parseErr != nil {
		parseT.Fatalf("expected structured payload build to succeed, got %v", parseErr)
	}
	parseObject := parseStructured.(js.Value)
	if parseObject.Get("visible").String() != "shown" {
		parseT.Fatalf("expected visible field to survive conversion, got %q", parseObject.Get("visible").String())
	}
	if !parseObject.Get("omit").IsUndefined() || !parseObject.Get("Skip").IsUndefined() || !parseObject.Get("hidden").IsUndefined() {
		parseT.Fatalf("expected omitted and hidden fields to stay absent, got omit=%v skip=%v hidden=%v", parseObject.Get("omit"), parseObject.Get("Skip"), parseObject.Get("hidden"))
	}

	parseType := reflect.TypeOf(parseStructuredPayload{})
	parseVisibleField, parseVisibleOmitEmpty, parseVisibleSkip := buildStructuredJSFieldName(parseType.Field(0))
	if parseVisibleField != "visible" || parseVisibleOmitEmpty || parseVisibleSkip {
		parseT.Fatalf("unexpected visible field metadata: name=%q omitempty=%t skip=%t", parseVisibleField, parseVisibleOmitEmpty, parseVisibleSkip)
	}
	parseOmitField, parseOmitEmpty, parseOmitSkip := buildStructuredJSFieldName(parseType.Field(1))
	if parseOmitField != "omit" || !parseOmitEmpty || parseOmitSkip {
		parseT.Fatalf("unexpected omitempty field metadata: name=%q omitempty=%t skip=%t", parseOmitField, parseOmitEmpty, parseOmitSkip)
	}
	parseSkipField, parseSkipOmitEmpty, parseSkipSkip := buildStructuredJSFieldName(parseType.Field(2))
	if parseSkipField != "" || parseSkipOmitEmpty || !parseSkipSkip {
		parseT.Fatalf("unexpected skipped field metadata: name=%q omitempty=%t skip=%t", parseSkipField, parseSkipOmitEmpty, parseSkipSkip)
	}

	if !isStructuredEmptyValue(reflect.ValueOf("")) || !isStructuredEmptyValue(reflect.ValueOf(0)) || !isStructuredEmptyValue(reflect.ValueOf([]string{})) {
		parseT.Fatal("expected zero-value fields to be considered empty")
	}
	if isStructuredEmptyValue(reflect.ValueOf("value")) || isStructuredEmptyValue(reflect.ValueOf(3)) {
		parseT.Fatal("expected populated fields to remain non-empty")
	}

	parseTyped, parseErr := goValueToJS("test", "payload", map[string]any{"count": 2, "label": "ok"})
	if parseErr != nil {
		parseT.Fatalf("expected map conversion to succeed, got %v", parseErr)
	}
	if parseTyped.(js.Value).Get("count").Int() != 2 || parseTyped.(js.Value).Get("label").String() != "ok" {
		parseT.Fatalf("unexpected goValueToJS map payload: %#v", parseTyped)
	}
	if _, parseErr := goValueToJS("test", "payload", map[string]any{"handler": func() {}}); !IsCode(parseErr, CodeEncode) {
		parseT.Fatalf("expected unsupported function value to fail JSON encoding, got %v", parseErr)
	}

	parseCapabilities := clientCapabilitiesJS(ClientCapabilities{
		ProtocolVersion: "1.2",
		Transports:      []string{"broadcast", "storage"},
		Encodings:       []string{"json"},
		Topics:          []string{"presence"},
		MaxJSONBytes:    1024,
		MaxBinaryBytes:  4096,
	})
	if parseCapabilities.Get("protocolVersion").String() != "1.2" || parseCapabilities.Get("transports").Length() != 2 || parseCapabilities.Get("maxJsonBytes").Int() != 1024 || parseCapabilities.Get("maxBinaryBytes").Int() != 4096 {
		parseT.Fatalf("unexpected clientCapabilitiesJS payload: %#v", parseCapabilities)
	}

	parseMessageValue, parseErr := goValueToJSStructured("test", "client-message", ClientMessage{
		Kind:  ClientHello,
		Topic: "presence",
		Source: ClientIdentity{
			ID:      "client-1",
			App:     "atlas",
			Surface: "window",
		},
		Capabilities: &ClientCapabilities{
			ProtocolVersion: "1.2",
			Transports:      []string{"broadcast"},
			Encodings:       []string{"json"},
		},
		Payload: map[string]any{"count": 4},
	})
	if parseErr != nil {
		parseT.Fatalf("expected client message structured encoding to succeed, got %v", parseErr)
	}
	parseMessage := parseMessageValue.(js.Value)
	if parseMessage.Get("kind").String() != string(ClientHello) || parseMessage.Get("source").Get("id").String() != "client-1" || parseMessage.Get("capabilities").Get("protocolVersion").String() != "1.2" {
		parseT.Fatalf("unexpected structured client message payload: %#v", parseMessage)
	}
	if parseMessage.Get("payload").Get("count").Int() != 4 {
		parseT.Fatalf("expected structured payload field to survive client message encoding, got %#v", parseMessage.Get("payload"))
	}
}

// TestInteropConvertHelpersWasmRejectCyclesAndReportConsoleErrors verifies cyclic payload detection and console error forwarding branches.
func TestInteropConvertHelpersWasmRejectCyclesAndReportConsoleErrors(parseT *testing.T) {
	type parseNode struct {
		Next *parseNode `json:"next,omitempty"`
	}

	parseRoot := &parseNode{}
	parseRoot.Next = parseRoot
	if _, parseErr := buildStructuredJSPayload("test", "cycle", parseRoot, map[uintptr]struct{}{}); !IsCode(parseErr, CodeEncode) {
		parseT.Fatalf("expected cyclic pointer payload to fail, got %v", parseErr)
	}

	parseSlice := make([]any, 1)
	parseSlice[0] = parseSlice
	if _, parseErr := buildStructuredJSPayload("test", "slice-cycle", parseSlice, map[uintptr]struct{}{}); !IsCode(parseErr, CodeEncode) {
		parseT.Fatalf("expected cyclic slice payload to fail, got %v", parseErr)
	}

	parseConsole := js.Global().Get("Object").New()
	parseCaptured := ""
	parseErrorFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseCaptured = parseArgs[0].String()
		}
		return nil
	})
	defer parseErrorFn.Release()
	parseConsole.Set("error", parseErrorFn)
	parseRestoreConsole := setGlobalValue("console", parseConsole)
	defer parseRestoreConsole()
	consoleError("convert boom")
	if parseCaptured != "convert boom" {
		parseT.Fatalf("expected consoleError to forward the message, got %q", parseCaptured)
	}

	parseRestoreConsole = setGlobalValue("console", js.Null())
	defer parseRestoreConsole()
	consoleError("ignored")

	parseRoundTrip, parseErr := jsValueToGo("test", "payload", js.Global().Get("JSON").Call("parse", `{"name":"demo","items":[1,true]}`))
	if parseErr != nil {
		parseT.Fatalf("expected JSON-shaped value to round-trip to Go, got %v", parseErr)
	}
	parseJSON, parseErr := json.Marshal(parseRoundTrip)
	if parseErr != nil {
		parseT.Fatalf("expected round-trip Go value to marshal cleanly, got %v", parseErr)
	}
	if string(parseJSON) != `{"items":[1,true],"name":"demo"}` && string(parseJSON) != `{"name":"demo","items":[1,true]}` {
		parseT.Fatalf("unexpected round-trip JSON payload: %s", parseJSON)
	}
}
