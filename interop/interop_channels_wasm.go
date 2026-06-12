//go:build js && wasm

package interop

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

func OpenCrossTabChannel(parseOptions CrossTabChannelOptions) (CrossTabChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return CrossTabChannel{}, wrapError("OpenCrossTabChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseSource := fmt.Sprintf("%s-%d", parseName, time.Now().UnixNano())
	if parseCtor := js.Global().Get("BroadcastChannel"); parseCtor.Type() == js.TypeFunction {
		return newBroadcastCrossTabChannel(parseName, parseSource, parseCtor.New(parseName)), nil
	}
	return newStorageCrossTabChannel(parseName, parseSource, resolveCrossTabStorageKey(parseName, parseOptions.StorageKey))
}

// OpenSecondaryWindowChannel opens a postMessage channel to a window opened via window.open.
func OpenSecondaryWindowChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseRawWindow, parseErr := globalProperty("Window", "window")
	if parseErr != nil {
		return WindowChannel{}, parseErr
	}
	parseOpenFn := parseRawWindow.Get("open")
	if parseOpenFn.Type() != js.TypeFunction {
		return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", parseName)
	}
	parseRawURL := strings.TrimSpace(parseOptions.URL)
	if parseRawURL == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseName, CodeInvalid, errors.New("window URL is empty"))
	}
	parseRaw := parseOpenFn.Invoke(parseRawURL, parseName, strings.TrimSpace(parseOptions.Features))
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseName, CodeUnavailable, errors.New("window.open returned no handle"))
	}
	return newWindowChannel(parseName, resolveWindowTargetOrigin(strings.TrimSpace(parseOptions.TargetOrigin)), parseRaw, true), nil
}

// OpenWindowOpenerChannel opens a postMessage channel to the window.opener.
func OpenWindowOpenerChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return WindowChannel{}, wrapError("OpenWindowOpenerChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseRawWindow, parseErr := globalProperty("Window", "window")
	if parseErr != nil {
		return WindowChannel{}, parseErr
	}
	parseOpener := parseRawWindow.Get("opener")
	if parseOpener.IsUndefined() || parseOpener.IsNull() {
		return WindowChannel{}, unavailable("OpenWindowOpenerChannel", parseName)
	}
	return newWindowChannel(parseName, resolveWindowTargetOrigin(strings.TrimSpace(parseOptions.TargetOrigin)), parseOpener, false), nil
}

const defaultWorkerReadyTimeout = 5 * time.Second

func newBroadcastCrossTabChannel(parseName string, parseSource string, parseRaw js.Value) CrossTabChannel {
	var (
		parseMu       sync.Mutex
		parseSequence int64
		isParseActive = true
	)
	parseNextEnvelope := func(parsePayload any) CrossTabEnvelope {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseSequence++
		return CrossTabEnvelope{
			Name:     parseName,
			Payload:  parsePayload,
			Source:   parseSource,
			Sequence: parseSequence,
			SentAt:   time.Now().UTC(),
		}
	}
	parseCurrent := func(parseOp string) (js.Value, error) {
		parseMu.Lock()
		defer parseMu.Unlock()
		if !isParseActive || parseRaw.IsUndefined() || parseRaw.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseName, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return parseRaw, nil
	}
	return CrossTabChannel{
		name:      func() string { return parseName },
		transport: func() string { return "broadcast-channel" },
		publish: func(parsePayload2 any) error {
			parseTarget, parseErr := parseCurrent("CrossTabChannel.Publish")
			if parseErr != nil {
				return parseErr
			}
			parseValue, parseErr := goValueToJS("CrossTabChannel.Publish", parseName, parseNextEnvelope(parsePayload2))
			if parseErr != nil {
				return parseErr
			}
			parseTarget.Call("postMessage", parseValue)
			return nil
		},
		publishClientBinary: func(parseMessage ClientMessage) error {
			parseTarget2, parseErr2 := parseCurrent("PublishClientBinaryCrossTab")
			if parseErr2 != nil {
				return parseErr2
			}
			parseEnvelope, parseErr2 := crossTabEnvelopeJS(parseName, parseSource, parseNextEnvelope(parseMessage))
			if parseErr2 != nil {
				return parseErr2
			}
			parseTarget2.Call("postMessage", parseEnvelope)
			return nil
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			parseTarget3, parseErr3 := parseCurrent("CrossTabChannel.Subscribe")
			if parseErr3 != nil {
				return Subscription{}, parseErr3
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				defer RecoverContainedPanic("newBroadcastCrossTabChannel callback")
				if len(parseArgs) == 0 {
					handler(CrossTabEnvelope{Name: parseName}, wrapError("CrossTabChannel.Subscribe", parseName, CodeDecode, errors.New("broadcast message event is missing")))
					return nil
				}
				parseData := parseArgs[0].Get("data")
				parseValue2, parseDecodeErr := jsValueToGo("CrossTabChannel.Subscribe", parseName, parseData)
				if parseDecodeErr != nil {
					handler(CrossTabEnvelope{Name: parseName}, parseDecodeErr)
					return nil
				}
				handler(crossTabEnvelopeFromGo(parseValue2, parseName), nil)
				return nil
			})
			parseTarget3.Call("addEventListener", "message", parseListener)
			return Subscription{cancel: func() {
				parseTarget3.Call("removeEventListener", "message", parseListener)
				parseListener.Release()
			}}, nil
		},
		close: func() error {
			parseTarget4, parseErr4 := parseCurrent("CrossTabChannel.Close")
			if parseErr4 != nil {
				return parseErr4
			}
			parseTarget4.Call("close")
			parseMu.Lock()
			isParseActive = false
			parseRaw = js.Undefined()
			parseMu.Unlock()
			return nil
		},
	}
}

func newStorageCrossTabChannel(parseName string, parseSource string, parseStorageKey string) (CrossTabChannel, error) {
	parseStorage, parseErr := GetLocalStorage()
	if parseErr != nil {
		return CrossTabChannel{}, parseErr
	}
	parseWindow, parseErr := globalProperty("EventTarget", "window")
	if parseErr != nil {
		return CrossTabChannel{}, parseErr
	}
	var (
		parseMu       sync.Mutex
		parseSequence int64
		isParseActive = true
	)
	parseNextEnvelope := func(parsePayload any) CrossTabEnvelope {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseSequence++
		return CrossTabEnvelope{
			Name:     parseName,
			Payload:  parsePayload,
			Source:   parseSource,
			Sequence: parseSequence,
			SentAt:   time.Now().UTC(),
		}
	}
	parseEnsureActive := func(parseOp string) error {
		parseMu.Lock()
		defer parseMu.Unlock()
		if !isParseActive {
			return wrapError(parseOp, parseName, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return nil
	}
	return CrossTabChannel{
		name:      func() string { return parseName },
		transport: func() string { return "storage-event" },
		publish: func(parsePayload2 any) error {
			if parseErr2 := parseEnsureActive("CrossTabChannel.Publish"); parseErr2 != nil {
				return parseErr2
			}
			parseData, parseErr3 := json.Marshal(parseNextEnvelope(parsePayload2))
			if parseErr3 != nil {
				return wrapError("CrossTabChannel.Publish", parseName, CodeEncode, parseErr3)
			}
			if parseErr4 := parseStorage.SetItem(parseStorageKey, string(parseData)); parseErr4 != nil {
				return parseErr4
			}
			return parseStorage.RemoveItem(parseStorageKey)
		},
		publishClientBinary: func(parseMessage2 ClientMessage) error {
			return wrapError("PublishClientBinaryCrossTab", parseName, CodeInvalid, errors.New("binary client payloads require broadcast-channel transport"))
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			if parseErr5 := parseEnsureActive("CrossTabChannel.Subscribe"); parseErr5 != nil {
				return Subscription{}, parseErr5
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				defer RecoverContainedPanic("newStorageCrossTabChannel callback")
				if len(parseArgs) == 0 {
					return nil
				}
				parseEvent := parseArgs[0]
				if parseEvent.IsUndefined() || parseEvent.IsNull() {
					return nil
				}
				if parseEvent.Get("key").String() != parseStorageKey {
					return nil
				}
				parseNewValue := parseEvent.Get("newValue")
				if parseNewValue.IsUndefined() || parseNewValue.IsNull() || strings.TrimSpace(parseNewValue.String()) == "" {
					return nil
				}
				var parseMessage CrossTabEnvelope
				if parseErr6 := json.Unmarshal([]byte(parseNewValue.String()), &parseMessage); parseErr6 != nil {
					handler(CrossTabEnvelope{Name: parseName}, wrapError("CrossTabChannel.Subscribe", parseName, CodeDecode, parseErr6))
					return nil
				}
				if strings.TrimSpace(parseMessage.Name) == "" {
					parseMessage.Name = parseName
				}
				handler(parseMessage, nil)
				return nil
			})
			parseWindow.Call("addEventListener", "storage", parseListener)
			return Subscription{cancel: func() {
				parseWindow.Call("removeEventListener", "storage", parseListener)
				parseListener.Release()
			}}, nil
		},
		close: func() error {
			if parseErr7 := parseEnsureActive("CrossTabChannel.Close"); parseErr7 != nil {
				return parseErr7
			}
			parseMu.Lock()
			isParseActive = false
			parseMu.Unlock()
			return nil
		},
	}, nil
}

func newWindowChannel(parseName string, parseTargetOrigin string, parsePeer js.Value, isAllowClose bool) WindowChannel {
	parseSource := fmt.Sprintf("%s-%d", parseName, time.Now().UnixNano())
	parseRawWindow := js.Global().Get("window")
	return WindowChannel{
		name:         func() string { return parseName },
		targetOrigin: func() string { return parseTargetOrigin },
		publish: func(parsePayload any) error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Publish", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed := parsePeer.Get("closed"); !parseClosed.IsUndefined() && !parseClosed.IsNull() && parseClosed.Bool() {
				return wrapError("WindowChannel.Publish", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			parseValue, parseErr := goValueToJS("WindowChannel.Publish", parseName, WindowEnvelope{
				Name:    parseName,
				Payload: parsePayload,
				Source:  parseSource,
				SentAt:  time.Now().UTC(),
			})
			if parseErr != nil {
				return parseErr
			}
			parsePeer.Call("postMessage", parseValue, parseTargetOrigin)
			return nil
		},
		publishClientBinary: func(parseMessage ClientMessage) error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("PublishClientBinaryWindow", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed2 := parsePeer.Get("closed"); !parseClosed2.IsUndefined() && !parseClosed2.IsNull() && parseClosed2.Bool() {
				return wrapError("PublishClientBinaryWindow", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			parseEnvelope, parseErr2 := windowEnvelopeJS(parseName, parseSource, parseMessage)
			if parseErr2 != nil {
				return parseErr2
			}
			parsePeer.Call("postMessage", parseEnvelope, parseTargetOrigin)
			return nil
		},
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("WindowChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				defer RecoverContainedPanic("newWindowChannel callback")
				if len(parseArgs) == 0 {
					handler(WindowEnvelope{Name: parseName}, wrapError("WindowChannel.Subscribe", parseName, CodeDecode, errors.New("message event is missing")))
					return nil
				}
				parseEvent := parseArgs[0]
				if parseEvent.IsUndefined() || parseEvent.IsNull() {
					return nil
				}
				parseEventSource := parseEvent.Get("source")
				if !parseEventSource.IsUndefined() && !parseEventSource.IsNull() && !parseEventSource.Equal(parsePeer) {
					return nil
				}
				if parseTargetOrigin != "*" {
					parseOrigin := strings.TrimSpace(parseEvent.Get("origin").String())
					if parseOrigin != "" && parseOrigin != parseTargetOrigin {
						handler(WindowEnvelope{Name: parseName}, wrapError("WindowChannel.Subscribe", parseName, CodeUnauthorized, errors.New("message origin does not match target origin")))
						return nil
					}
				}
				parseValue2, parseDecodeErr := jsValueToGo("WindowChannel.Subscribe", parseName, parseEvent.Get("data"))
				if parseDecodeErr != nil {
					handler(WindowEnvelope{Name: parseName}, parseDecodeErr)
					return nil
				}
				handler(windowEnvelopeFromGo(parseValue2, parseName), nil)
				return nil
			})
			parseRawWindow.Call("addEventListener", "message", parseListener)
			return Subscription{cancel: func() {
				parseRawWindow.Call("removeEventListener", "message", parseListener)
				parseListener.Release()
			}}, nil
		},
		focus: func() error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Focus", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed := parsePeer.Get("closed"); !parseClosed.IsUndefined() && !parseClosed.IsNull() && parseClosed.Bool() {
				return wrapError("WindowChannel.Focus", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			if parseFn := parsePeer.Get("focus"); parseFn.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Focus", parseName)
			}
			parsePeer.Call("focus")
			return nil
		},
		close: func() error {
			if !isAllowClose {
				return unavailable("WindowChannel.Close", parseName)
			}
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Close", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed := parsePeer.Get("closed"); !parseClosed.IsUndefined() && !parseClosed.IsNull() && parseClosed.Bool() {
				return wrapError("WindowChannel.Close", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			if parseFn2 := parsePeer.Get("close"); parseFn2.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Close", parseName)
			}
			parsePeer.Call("close")
			return nil
		},
		closed: func() bool {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return true
			}
			parseClosed3 := parsePeer.Get("closed")
			if parseClosed3.IsUndefined() || parseClosed3.IsNull() {
				return false
			}
			return parseClosed3.Bool()
		},
	}
}

func resolveCrossTabStorageKey(parseName string, parseOverride string) string {
	parseTrimmed := strings.TrimSpace(parseOverride)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	return "__gwc_cross_tab__:" + parseName
}

func resolveWindowTargetOrigin(parseRawTargetOrigin string) string {
	parseTrimmed := strings.TrimSpace(parseRawTargetOrigin)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return "*"
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.IsUndefined() || parseLocation.IsNull() {
		return "*"
	}
	parseOrigin := strings.TrimSpace(parseLocation.Get("origin").String())
	if parseOrigin == "" {
		return "*"
	}
	return parseOrigin
}

func crossTabEnvelopeFromGo(parseValue any, parseFallbackName string) CrossTabEnvelope {
	parseMessage := CrossTabEnvelope{
		Name:    parseFallbackName,
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseSource := workerStringField(parseData, "source"); parseSource != "" {
		parseMessage.Source = parseSource
	}
	if parseSequence, parseOk3 := crossTabInt64Field(parseData["sequence"]); parseOk3 {
		parseMessage.Sequence = parseSequence
	}
	if parseSentAt, parseOk4 := crossTabTimeField(parseData["sentAt"]); parseOk4 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage
}

func windowEnvelopeFromGo(parseValue any, parseFallbackName string) WindowEnvelope {
	parseMessage := WindowEnvelope{
		Name:    parseFallbackName,
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseSource := workerStringField(parseData, "source"); parseSource != "" {
		parseMessage.Source = parseSource
	}
	if parseSentAt, parseOk3 := crossTabTimeField(parseData["sentAt"]); parseOk3 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage
}

func crossTabInt64Field(parseValue any) (int64, bool) {
	switch parseTyped := parseValue.(type) {
	case float64:
		return int64(parseTyped), true
	case float32:
		return int64(parseTyped), true
	case int:
		return int64(parseTyped), true
	case int64:
		return parseTyped, true
	case json.Number:
		parseParsed, parseErr := parseTyped.Int64()
		return parseParsed, parseErr == nil
	default:
		return 0, false
	}
}

func crossTabTimeField(parseValue any) (time.Time, bool) {
	parseText, parseOk := parseValue.(string)
	if !parseOk || strings.TrimSpace(parseText) == "" {
		return time.Time{}, false
	}
	parseParsed, parseErr := time.Parse(time.RFC3339Nano, parseText)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseParsed, true
}
