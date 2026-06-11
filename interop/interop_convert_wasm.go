//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

func globalProperty(parseOp string, parseName string) (js.Value, error) {
	parseValue := js.Global().Get(parseName)
	if (parseValue.IsUndefined() || parseValue.IsNull()) && parseName != "window" {
		if parseWindow := js.Global().Get("window"); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
			parseValue = parseWindow.Get(parseName)
		}
	}
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return js.Undefined(), unavailable(parseOp, parseName)
	}
	return parseValue, nil
}

func globalPath(parseOp string, parseHead string, parseTail string) (js.Value, error) {
	parseRoot, parseErr := globalProperty(parseOp, parseHead)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue := parseRoot.Get(parseTail)
	if (parseValue.IsUndefined() || parseValue.IsNull()) && parseHead != "window" {
		if parseWindow := js.Global().Get("window"); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
			parseNextRoot := parseWindow.Get(parseHead)
			if !parseNextRoot.IsUndefined() && !parseNextRoot.IsNull() {
				parseValue = parseNextRoot.Get(parseTail)
			}
		}
	}
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return js.Undefined(), unavailable(parseOp, parseHead+"."+parseTail)
	}
	return parseValue, nil
}

func durationMS(parseValue time.Duration) int {
	if parseValue <= 0 {
		return 0
	}
	return int(parseValue / time.Millisecond)
}

func awaitValue(parseCtx context.Context, parseOp string, parseTarget string, parseValue js.Value) (js.Value, error) {
	if !isPromise(parseValue) {
		return parseValue, nil
	}
	parseResolvedCh := make(chan js.Value, 1)
	parseRejectedCh := make(chan js.Value, 1)
	var (
		parseResolve js.Func
		parseReject  js.Func
		parseOnce    sync.Once
	)
	parseCleanup := func() {
		parseResolve.Release()
		parseReject.Release()
	}
	parseResolve = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("awaitValue callback")
		parseOnce.Do(func() {
			if len(parseArgs) > 0 {
				parseResolvedCh <- parseArgs[0]
			} else {
				parseResolvedCh <- js.Undefined()
			}
			parseCleanup()
		})
		return nil
	})
	parseReject = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("awaitValue callback")
		parseOnce.Do(func() {
			if len(parseArgs2) > 0 {
				parseRejectedCh <- parseArgs2[0]
			} else {
				parseRejectedCh <- js.ValueOf("promise rejected")
			}
			parseCleanup()
		})
		return nil
	})
	parseValue.Call("then", parseResolve, parseReject)

	select {
	case parseResolved := <-parseResolvedCh:
		return parseResolved, nil
	case parseRejected := <-parseRejectedCh:
		return js.Undefined(), wrapError(parseOp, parseTarget, CodePromiseRejected, errors.New(jsValueSummary(parseRejected)))
	case <-parseCtx.Done():
		return js.Undefined(), wrapError(parseOp, parseTarget, CodePromiseRejected, parseCtx.Err())
	}
}

func isPromise(parseValue js.Value) bool {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return false
	}
	if parseValue.Type() != js.TypeObject && parseValue.Type() != js.TypeFunction {
		return false
	}
	parseThen := parseValue.Get("then")
	return parseThen.Type() == js.TypeFunction
}

func goValueToJS(parseOp string, parseTarget string, parseValue any) (interface{}, error) {
	switch parseTyped := parseValue.(type) {
	case nil:
		return js.Null(), nil
	case []byte:
		parseArray := js.Global().Get("Uint8Array").New(len(parseTyped))
		js.CopyBytesToJS(parseArray, parseTyped)
		return parseArray, nil
	case SharedBuffer:
		if parseRaw, parseOk := getSharedBufferRaw(parseTyped); parseOk && !parseRaw.IsUndefined() && !parseRaw.IsNull() {
			return parseRaw, nil
		}
		return js.Undefined(), unavailable(parseOp, parseTarget)
	case Value:
		if parseRaw, parseOk := parseTyped.rawValue(); parseOk {
			return parseRaw, nil
		}
		return js.Undefined(), unavailable(parseOp, parseTarget)
	case js.Value:
		return parseTyped, nil
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return js.ValueOf(parseTyped), nil
	default:
		parseData, parseErr := json.Marshal(parseValue)
		if parseErr != nil {
			return nil, wrapError(parseOp, parseTarget, CodeEncode, parseErr)
		}
		return js.Global().Get("JSON").Call("parse", string(parseData)), nil
	}
}

func goValuesToJS(parseOp string, parseTarget string, parseValues ...any) ([]interface{}, error) {
	if len(parseValues) == 0 {
		return nil, nil
	}
	parseConverted := make([]interface{}, len(parseValues))
	for parseIndex, parseValue := range parseValues {
		parseJsValue, parseErr := goValueToJS(parseOp, parseTarget, parseValue)
		if parseErr != nil {
			return nil, parseErr
		}
		parseConverted[parseIndex] = parseJsValue
	}
	return parseConverted, nil
}

func jsValueToGo(parseOp string, parseTarget string, parseValue js.Value) (any, error) {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return nil, nil
	}
	if parseSharedBuffer, parseOk := getSharedBufferFromJS(parseTarget, parseValue); parseOk {
		return parseSharedBuffer, nil
	}
	if parseBytes, parseOk := jsValueToBytes(parseValue); parseOk {
		return parseBytes, nil
	}
	switch parseValue.Type() {
	case js.TypeBoolean:
		return parseValue.Bool(), nil
	case js.TypeString:
		return parseValue.String(), nil
	case js.TypeNumber:
		return parseValue.Float(), nil
	case js.TypeObject:
		if parseValue.InstanceOf(js.Global().Get("Array")) {
			parseItems := make([]any, parseValue.Length())
			for parseIndex := 0; parseIndex < parseValue.Length(); parseIndex++ {
				parseItem, parseErr := jsValueToGo(parseOp, parseTarget, parseValue.Index(parseIndex))
				if parseErr != nil {
					return nil, parseErr
				}
				parseItems[parseIndex] = parseItem
			}
			return parseItems, nil
		}
		parseKeys := js.Global().Get("Object").Call("keys", parseValue)
		parseDecoded := make(map[string]any, parseKeys.Length())
		for parseIndex2 := 0; parseIndex2 < parseKeys.Length(); parseIndex2++ {
			parseKey := parseKeys.Index(parseIndex2).String()
			parseItem2, parseErr2 := jsValueToGo(parseOp, parseTarget, parseValue.Get(parseKey))
			if parseErr2 != nil {
				return nil, parseErr2
			}
			parseDecoded[parseKey] = parseItem2
		}
		return parseDecoded, nil
	case js.TypeFunction:
		return nil, wrapError(parseOp, parseTarget, CodeDecode, errors.New("function values are not serializable"))
	default:
		return parseValue.String(), nil
	}
}

func jsValueToBytes(parseValue js.Value) ([]byte, bool) {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return nil, false
	}
	parseArrayBuffer := js.Global().Get("ArrayBuffer")
	if parseArrayBuffer.Type() == js.TypeFunction && parseValue.InstanceOf(parseArrayBuffer) {
		parseView := js.Global().Get("Uint8Array").New(parseValue)
		parseBytes := make([]byte, parseView.Length())
		js.CopyBytesToGo(parseBytes, parseView)
		return parseBytes, true
	}
	if parseArrayBuffer.Type() != js.TypeFunction {
		return nil, false
	}
	isView := parseArrayBuffer.Get("isView")
	if isView.Type() != js.TypeFunction || !isView.Invoke(parseValue).Bool() {
		return nil, false
	}
	parseView2 := js.Global().Get("Uint8Array").New(parseValue.Get("buffer"), parseValue.Get("byteOffset"), parseValue.Get("byteLength"))
	parseBytes2 := make([]byte, parseView2.Length())
	js.CopyBytesToGo(parseBytes2, parseView2)
	return parseBytes2, true
}

// getSharedBufferFromJS wraps a SharedArrayBuffer JS value as SharedBuffer.
func getSharedBufferFromJS(parseTarget string, parseValue js.Value) (SharedBuffer, bool) {
	parseCtor := getGlobalValue("SharedArrayBuffer")
	if parseCtor.Type() != js.TypeFunction || !parseValue.InstanceOf(parseCtor) {
		return SharedBuffer{}, false
	}
	return buildSharedBuffer(parseTarget, parseValue), true
}

func crossTabEnvelopeJS(parseName string, parseSource string, parseEnvelope CrossTabEnvelope) (js.Value, error) {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("name", parseName)
	parseValue.Set("source", parseSource)
	parseValue.Set("sequence", parseEnvelope.Sequence)
	parseValue.Set("sentAt", parseEnvelope.SentAt.Format(time.RFC3339Nano))
	parsePayload, parseErr := goValueToJSStructured("CrossTabChannel.Publish", parseName, parseEnvelope.Payload)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue.Set("payload", parsePayload)
	return parseValue, nil
}

func windowEnvelopeJS(parseName string, parseSource string, parsePayload any) (js.Value, error) {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("name", parseName)
	parseValue.Set("source", parseSource)
	parseValue.Set("sentAt", time.Now().UTC().Format(time.RFC3339Nano))
	parseConverted, parseErr := goValueToJSStructured("WindowChannel.Publish", parseName, parsePayload)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue.Set("payload", parseConverted)
	return parseValue, nil
}

func postStructuredMessageJS(parseOp string, parseTarget string, parseRaw js.Value, parsePayload any, parsePorts ...MessagePort) (parseErr error) {
	defer recoverInteropException(parseOp, parseTarget, &parseErr)
	parseConverted, parseErr := goValueToJSStructured(parseOp, parseTarget, parsePayload)
	if parseErr != nil {
		return parseErr
	}
	if len(parsePorts) == 0 {
		parseRaw.Call("postMessage", parseConverted)
		return nil
	}
	parseTransfers, parseErr := messagePortTransferListJS(parseOp, parseTarget, parsePorts)
	if parseErr != nil {
		return parseErr
	}
	parseRaw.Call("postMessage", parseConverted, parseTransfers)
	return nil
}

func messagePortTransferListJS(parseOp string, parseTarget string, parsePorts []MessagePort) (js.Value, error) {
	parseTransfers := js.Global().Get("Array").New(len(parsePorts))
	for parseIndex, parsePort := range parsePorts {
		parseRawPort, parseOk := parsePort.raw.(js.Value)
		if !parseOk || parseRawPort.IsUndefined() || parseRawPort.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid, fmt.Errorf("message port %d is unavailable", parseIndex))
		}
		parseTransfers.SetIndex(parseIndex, parseRawPort)
	}
	return parseTransfers, nil
}

func goValueToJSStructured(parseOp string, parseTarget string, parseValue any) (interface{}, error) {
	switch parseTyped := parseValue.(type) {
	case WorkerMessage:
		parseMessage := js.Global().Get("Object").New()
		if strings.TrimSpace(parseTyped.ID) != "" {
			parseMessage.Set("id", parseTyped.ID)
		}
		if strings.TrimSpace(parseTyped.Phase) != "" {
			parseMessage.Set("phase", parseTyped.Phase)
		}
		if strings.TrimSpace(parseTyped.Name) != "" {
			parseMessage.Set("name", parseTyped.Name)
		}
		if strings.TrimSpace(parseTyped.Error) != "" {
			parseMessage.Set("error", parseTyped.Error)
		}
		if parseTyped.Payload != nil {
			parsePayload, parseErr := buildStructuredJSPayload(parseOp, parseTarget, parseTyped.Payload, map[uintptr]struct{}{})
			if parseErr != nil {
				return nil, parseErr
			}
			parseMessage.Set("payload", parsePayload)
		}
		return parseMessage, nil
	case ClientMessage:
		parseMessage := js.Global().Get("Object").New()
		if strings.TrimSpace(parseTyped.ID) != "" {
			parseMessage.Set("id", parseTyped.ID)
		}
		parseMessage.Set("kind", string(parseTyped.Kind))
		parseMessage.Set("topic", parseTyped.Topic)
		parseMessage.Set("source", clientIdentityJS(parseTyped.Source))
		if parseTyped.Capabilities != nil {
			parseMessage.Set("capabilities", clientCapabilitiesJS(*parseTyped.Capabilities))
		}
		if strings.TrimSpace(parseTyped.Target) != "" {
			parseMessage.Set("target", parseTyped.Target)
		}
		if parseTyped.Encoding != "" {
			parseMessage.Set("encoding", string(parseTyped.Encoding))
		}
		if strings.TrimSpace(parseTyped.ContentType) != "" {
			parseMessage.Set("contentType", parseTyped.ContentType)
		}
		if strings.TrimSpace(parseTyped.Revision) != "" {
			parseMessage.Set("revision", parseTyped.Revision)
		}
		if strings.TrimSpace(parseTyped.Error) != "" {
			parseMessage.Set("error", parseTyped.Error)
		}
		if !parseTyped.SentAt.IsZero() {
			parseMessage.Set("sentAt", parseTyped.SentAt.Format(time.RFC3339Nano))
		}
		if parseTyped.Payload != nil {
			parsePayload, parseErr := buildStructuredJSPayload(parseOp, parseTarget, parseTyped.Payload, map[uintptr]struct{}{})
			if parseErr != nil {
				return nil, parseErr
			}
			parseMessage.Set("payload", parsePayload)
		}
		return parseMessage, nil
	default:
		return buildStructuredJSPayload(parseOp, parseTarget, parseValue, map[uintptr]struct{}{})
	}
}

// buildStructuredJSPayload recursively converts JSON-shaped Go values into JS
// objects while preserving `SharedBuffer` leaves for structured-clone paths.
func buildStructuredJSPayload(parseOp string, parseTarget string, parseValue any, parseSeen map[uintptr]struct{}) (interface{}, error) {
	if parseValue == nil {
		return js.Null(), nil
	}
	return buildStructuredJSReflect(parseOp, parseTarget, reflect.ValueOf(parseValue), parseSeen)
}

// buildStructuredJSReflect walks one reflected Go value into its JS structured
// clone equivalent while preserving supported interop wrapper leaves.
func buildStructuredJSReflect(parseOp string, parseTarget string, parseValue reflect.Value, parseSeen map[uintptr]struct{}) (interface{}, error) {
	if !parseValue.IsValid() {
		return js.Null(), nil
	}
	if parseValue.CanInterface() {
		switch parseValue.Interface().(type) {
		case nil, []byte, SharedBuffer, Value, js.Value, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		case json.Marshaler, encoding.TextMarshaler:
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
	}

	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return js.Null(), nil
		}
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		return buildStructuredJSReflect(parseOp, parseTarget, parseValue.Elem(), parseSeen)
	case reflect.Map:
		if parseValue.IsNil() {
			return js.Null(), nil
		}
		if parseValue.Type().Key().Kind() != reflect.String {
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		parseObject := js.Global().Get("Object").New()
		for _, parseKey := range parseValue.MapKeys() {
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseTarget, parseValue.MapIndex(parseKey), parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseObject.Set(parseKey.String(), parseConverted)
		}
		return parseObject, nil
	case reflect.Slice, reflect.Array:
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		parseArray := js.Global().Get("Array").New(parseValue.Len())
		for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseTarget, parseValue.Index(parseIndex), parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseArray.SetIndex(parseIndex, parseConverted)
		}
		return parseArray, nil
	case reflect.Struct:
		parseObject := js.Global().Get("Object").New()
		parseType := parseValue.Type()
		for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
			parseField := parseType.Field(parseIndex)
			parseFieldName, isParseOmitEmpty, isParseSkip := buildStructuredJSFieldName(parseField)
			if isParseSkip {
				continue
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if isParseOmitEmpty && isStructuredEmptyValue(parseFieldValue) {
				continue
			}
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseFieldName, parseFieldValue, parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseObject.Set(parseFieldName, parseConverted)
		}
		return parseObject, nil
	default:
		if parseValue.CanInterface() {
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
		return nil, wrapError(parseOp, parseTarget, CodeEncode, errors.New("value is not serializable"))
	}
}

// buildStructuredJSVisit tracks pointer-like values so recursive structured
// conversion fails cleanly on cycles instead of recursing forever.
func buildStructuredJSVisit(parseOp string, parseTarget string, parseValue reflect.Value, parseSeen map[uintptr]struct{}) (uintptr, error) {
	if parseSeen == nil {
		return 0, nil
	}
	switch parseValue.Kind() {
	case reflect.Map:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	case reflect.Pointer:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	case reflect.Slice:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if parsePointer == 0 {
			return 0, nil
		}
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	default:
		return 0, nil
	}
}

// buildStructuredJSFieldName resolves the JSON field name and omitempty
// behavior for one exported struct field.
func buildStructuredJSFieldName(parseField reflect.StructField) (string, bool, bool) {
	if parseField.PkgPath != "" {
		return "", false, true
	}
	parseTag := strings.TrimSpace(parseField.Tag.Get("json"))
	if parseTag == "-" {
		return "", false, true
	}
	parseName := parseField.Name
	isParseOmitEmpty := false
	if parseTag != "" {
		parseParts := strings.Split(parseTag, ",")
		if strings.TrimSpace(parseParts[0]) != "" {
			parseName = strings.TrimSpace(parseParts[0])
		}
		for _, parsePart := range parseParts[1:] {
			if strings.TrimSpace(parsePart) == "omitempty" {
				isParseOmitEmpty = true
			}
		}
	}
	return parseName, isParseOmitEmpty, false
}

// isStructuredEmptyValue reports whether one reflected field should be skipped
// for an `omitempty` structured payload field.
func isStructuredEmptyValue(parseValue reflect.Value) bool {
	if !parseValue.IsValid() {
		return true
	}
	switch parseValue.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return parseValue.Len() == 0
	case reflect.Bool:
		return !parseValue.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return parseValue.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return parseValue.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return parseValue.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return parseValue.IsNil()
	default:
		return parseValue.IsZero()
	}
}

func clientIdentityJS(parseIdentity ClientIdentity) js.Value {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("id", parseIdentity.ID)
	parseValue.Set("app", parseIdentity.App)
	parseValue.Set("surface", parseIdentity.Surface)
	if strings.TrimSpace(parseIdentity.Role) != "" {
		parseValue.Set("role", parseIdentity.Role)
	}
	if strings.TrimSpace(parseIdentity.Version) != "" {
		parseValue.Set("version", parseIdentity.Version)
	}
	return parseValue
}

func clientCapabilitiesJS(parseCapabilities ClientCapabilities) js.Value {
	parseValue := js.Global().Get("Object").New()
	if strings.TrimSpace(parseCapabilities.ProtocolVersion) != "" {
		parseValue.Set("protocolVersion", parseCapabilities.ProtocolVersion)
	}
	if len(parseCapabilities.Transports) > 0 {
		parseValue.Set("transports", stringArrayValue(parseCapabilities.Transports))
	}
	if len(parseCapabilities.Encodings) > 0 {
		parseValue.Set("encodings", stringArrayValue(parseCapabilities.Encodings))
	}
	if len(parseCapabilities.Topics) > 0 {
		parseValue.Set("topics", stringArrayValue(parseCapabilities.Topics))
	}
	if parseCapabilities.MaxJSONBytes > 0 {
		parseValue.Set("maxJsonBytes", parseCapabilities.MaxJSONBytes)
	}
	if parseCapabilities.MaxBinaryBytes > 0 {
		parseValue.Set("maxBinaryBytes", parseCapabilities.MaxBinaryBytes)
	}
	return parseValue
}

func jsValueSummary(parseValue js.Value) string {
	if parseValue.IsUndefined() {
		return "undefined"
	}
	if parseValue.IsNull() {
		return "null"
	}
	switch parseValue.Type() {
	case js.TypeString:
		return parseValue.String()
	case js.TypeBoolean:
		if parseValue.Bool() {
			return "true"
		}
		return "false"
	case js.TypeNumber:
		return fmt.Sprint(parseValue.Float())
	default:
		parseStringified := js.Global().Get("JSON").Call("stringify", parseValue)
		if parseStringified.IsUndefined() || parseStringified.IsNull() {
			return parseValue.String()
		}
		return parseStringified.String()
	}
}

// looksLikeJSTypeDescriptor returns true if a string looks like a Go syscall/js
// type descriptor (e.g. "<object>", "<undefined>", "<null>", "<function>") rather
// than a genuine value. Go 1.26+ returns these when js.Value.String() is called
// on a non-string JS value.
func looksLikeJSTypeDescriptor(parseS string) bool {
	parseTrimmed := strings.TrimSpace(parseS)
	switch parseTrimmed {
	case "<object>", "<undefined>", "<null>", "<function>", "<symbol>", "<number>", "<boolean>":
		return true
	}
	return false
}

// consoleError logs an error message to the browser console (console.error).
func consoleError(parseMsg string) {
	parseC := js.Global().Get("console")
	if parseC.IsUndefined() || parseC.IsNull() {
		return
	}
	parseC.Call("error", parseMsg)
}
