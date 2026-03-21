//go:build js && wasm
// +build js,wasm

package ui

import (
	"fmt"
	"syscall/js"
)

// ReadBootstrapScript reads an inline bootstrap script from the browser document.
func ReadBootstrapScript(scriptID string) (SSRBootstrap, error) {
	id := scriptID
	if id == "" {
		id = DefaultBootstrapScriptID
	}

	document := js.Global().Get("document")
	if !document.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapScript could not access document")
	}

	scriptNode := document.Call("getElementById", id)
	if !scriptNode.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapScript could not find script element with id %q", id)
	}

	text := scriptNode.Get("textContent").String()
	return UnmarshalSSRBootstrap([]byte(text))
}

// ReadBootstrapReferenceScript reads an inline bootstrap-reference script from the browser document.
func ReadBootstrapReferenceScript(scriptID string) (SSRBootstrapReference, error) {
	id := scriptID
	if id == "" {
		id = DefaultBootstrapReferenceScriptID
	}

	document := js.Global().Get("document")
	if !document.Truthy() {
		return SSRBootstrapReference{}, fmt.Errorf("ui.ReadBootstrapReferenceScript could not access document")
	}

	scriptNode := document.Call("getElementById", id)
	if !scriptNode.Truthy() {
		return SSRBootstrapReference{}, fmt.Errorf("ui.ReadBootstrapReferenceScript could not find script element with id %q", id)
	}

	text := scriptNode.Get("textContent").String()
	return UnmarshalSSRBootstrapReference([]byte(text))
}

// ReadBootstrapReference fetches and decodes an external bootstrap payload.
func ReadBootstrapReference(ref SSRBootstrapReference) (SSRBootstrap, error) {
	normalizedRef, err := normalizeSSRBootstrapReference(ref)
	if err != nil {
		return SSRBootstrap{}, err
	}
	ref = normalizedRef
	if ref.URL == "" {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapReference requires a non-empty URL")
	}

	fetchFunction := js.Global().Get("fetch")
	if !fetchFunction.Truthy() {
		return SSRBootstrap{}, fmt.Errorf("ui.ReadBootstrapReference could not access fetch")
	}

	type fetchResult struct {
		data []byte
		err  error
	}
	resultCh := make(chan fetchResult, 1)

	promiseCtor := js.Global().Get("Promise")
	uint8ArrayCtor := js.Global().Get("Uint8Array")

	var responseFn js.Func
	var dataFn js.Func
	var catchFn js.Func

	responseFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return promiseCtor.Call("reject", "missing fetch response")
		}
		response := args[0]
		ok := response.Get("ok")
		if ok.Truthy() && !ok.Bool() {
			return promiseCtor.Call("reject", fmt.Sprintf("bootstrap fetch failed with status %d", response.Get("status").Int()))
		}
		return response.Call("arrayBuffer")
	})

	dataFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			resultCh <- fetchResult{err: fmt.Errorf("bootstrap fetch returned no data")}
			return nil
		}
		buffer := args[0]
		bytesValue := uint8ArrayCtor.New(buffer)
		data := make([]byte, bytesValue.Get("length").Int())
		js.CopyBytesToGo(data, bytesValue)
		resultCh <- fetchResult{data: data}
		return nil
	})

	catchFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message := "bootstrap fetch failed"
		if len(args) > 0 {
			message = args[0].String()
		}
		resultCh <- fetchResult{err: fmt.Errorf("%s", message)}
		return nil
	})

	defer responseFn.Release()
	defer dataFn.Release()
	defer catchFn.Release()

	fetchFunction.Invoke(ref.URL).Call("then", responseFn).Call("then", dataFn).Call("catch", catchFn)
	result := <-resultCh
	if result.err != nil {
		return SSRBootstrap{}, result.err
	}

	switch ref.Format {
	case "", SSRBootstrapFormatJSON:
		return UnmarshalSSRBootstrap(result.data)
	case SSRBootstrapFormatCBOR:
		return UnmarshalSSRBootstrapBinary(result.data)
	default:
		return SSRBootstrap{}, fmt.Errorf("unsupported bootstrap reference format %q", ref.Format)
	}
}
