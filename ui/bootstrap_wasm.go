//go:build js && wasm

package ui

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"syscall/js"
)

// ReadBootstrapScript reads an inline bootstrap script from the browser document.
func ReadBootstrapScript(parseScriptID string) (SSRBootstrap, error) {
	parseId := parseScriptID
	if parseId == "" {
		parseId = DefaultBootstrapScriptID
	}

	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapScript could not access document")
	}

	parseScriptNode := parseDocument.Call("getElementById", parseId)
	if !parseScriptNode.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapScript could not find script element with id %q", parseId)
	}

	parseText := parseScriptNode.Get("textContent").String()
	return UnmarshalSSRBootstrap([]byte(parseText))
}

// ReadBootstrapReferenceScript reads an inline bootstrap-reference script from the browser document.
func ReadBootstrapReferenceScript(parseScriptID string) (SSRBootstrapReference, error) {
	parseId := parseScriptID
	if parseId == "" {
		parseId = DefaultBootstrapReferenceScriptID
	}

	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return SSRBootstrapReference{}, fmt.Errorf("ui.ReadBootstrapReferenceScript could not access document")
	}

	parseScriptNode := parseDocument.Call("getElementById", parseId)
	if !parseScriptNode.Truthy() {
		return SSRBootstrapReference{}, fmt.Errorf("ui.ReadBootstrapReferenceScript could not find script element with id %q", parseId)
	}

	parseText := parseScriptNode.Get("textContent").String()
	return UnmarshalSSRBootstrapReference([]byte(parseText))
}

// ReadBootstrapReference fetches and decodes an external bootstrap payload.
func ReadBootstrapReference(parseRef SSRBootstrapReference) (SSRBootstrap, error) {
	parseNormalizedRef, parseErr := normalizeSSRBootstrapReference(parseRef)
	if parseErr != nil {
		return SSRBootstrap{}, parseErr
	}
	parseRef = parseNormalizedRef
	if parseRef.URL == "" {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapReference requires a non-empty URL")
	}

	parseFetchFunction := js.Global().Get("fetch")
	if !parseFetchFunction.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapReference could not access fetch")
	}

	type fetchResult struct {
		data []byte
		err  error
	}
	parseResultCh := make(chan fetchResult, 1)

	parsePromiseCtor := js.Global().Get("Promise")
	parseUint8ArrayCtor := js.Global().Get("Uint8Array")

	var parseResponseFn js.Func
	var parseDataFn js.Func
	var parseCatchFn js.Func

	parseResponseFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("ui", "ReadBootstrapReference callback")
		if len(parseArgs) == 0 {
			return parsePromiseCtor.Call("reject", "missing fetch response")
		}
		parseResponse := parseArgs[0]
		parseOk := parseResponse.Get("ok")
		// Finding #60: for a JS false value Truthy() returns false, so the
		// original `Truthy() && !Bool()` branch never fired for 4xx/5xx.
		// Use a type-based check instead so false is correctly detected.
		if parseOk.Type() == js.TypeBoolean && !parseOk.Bool() {
			return parsePromiseCtor.Call("reject", fmt.Sprintf("bootstrap fetch failed with status %d", parseResponse.Get("status").Int()))
		}
		return parseResponse.Call("arrayBuffer")
	})

	parseDataFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("ui", "ReadBootstrapReference callback")
		if len(parseArgs2) == 0 {
			parseResultCh <- fetchResult{err: fmt.Errorf("bootstrap fetch returned no data")}
			return nil
		}
		parseBuffer := parseArgs2[0]
		parseBytesValue := parseUint8ArrayCtor.New(parseBuffer)
		parseData := make([]byte, parseBytesValue.Get("length").Int())
		js.CopyBytesToGo(parseData, parseBytesValue)
		parseResultCh <- fetchResult{data: parseData}
		return nil
	})

	parseCatchFn = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("ui", "ReadBootstrapReference callback")
		parseMessage := "bootstrap fetch failed"
		if len(parseArgs3) > 0 {
			parseMessage = parseArgs3[0].String()
		}
		parseResultCh <- fetchResult{err: fmt.Errorf("%s", parseMessage)}
		return nil
	})

	defer parseResponseFn.Release()
	defer parseDataFn.Release()
	defer parseCatchFn.Release()

	parseFetchFunction.Invoke(parseRef.URL).Call("then", parseResponseFn).Call("then", parseDataFn).Call("catch", parseCatchFn)
	parseResult := <-parseResultCh
	if parseResult.err != nil {
		return SSRBootstrap{}, parseResult.err
	}

	switch parseRef.Format {
	case "", SSRBootstrapFormatJSON:
		return UnmarshalSSRBootstrap(parseResult.data)
	case SSRBootstrapFormatCBOR:
		return UnmarshalSSRBootstrapBinary(parseResult.data)
	default:
		return SSRBootstrap{}, fmt.Errorf("unsupported bootstrap reference format %q", parseRef.Format)
	}
}
