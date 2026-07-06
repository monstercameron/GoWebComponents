//go:build js && wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	gwcruntime "github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// Fetch performs an asynchronous HTTP fetch operation and returns a channel for the result.
// This uses the browser Fetch API from WASM; it currently returns the response body as text.
func Fetch(parseUrl string, parseOptions Options) <-chan Result {
	parseCh := make(chan Result, 1)

	go func() {
		// Browser fetch() throws SYNCHRONOUSLY (not via promise rejection) for
		// realistic inputs — a GET/HEAD with a body, an invalid URL, a
		// disallowed header name. A plain RecoverContainedPanic would log and
		// swallow that, leaving parseCh unwritten and the caller (result := <-Fetch)
		// blocked forever. This defer guarantees a terminal Result on any
		// synchronous throw during request setup. The async promise callbacks
		// below run on later event-loop turns (after this goroutine returns), so
		// parseSettled is only ever observed here for the synchronous path.
		parseSettled := false
		sendResult := func(parseResult Result) {
			if parseSettled {
				return
			}
			parseSettled = true
			parseCh <- parseResult
		}
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				gwcruntime.ContainPanic("fetch", gwcruntime.PanicPhaseAsync, "Do request", parseRecovered)
				sendResult(Result{Err: fmt.Errorf("fetch request failed: %v", parseRecovered)})
			}
		}()
		parseFetchFunction := js.Global().Get("fetch")
		if !parseFetchFunction.Truthy() {
			sendResult(Result{Err: errors.New("fetch API unavailable in this environment")})
			return
		}

		parseRequestOptions := js.Global().Get("Object").New()

		parseMethod := parseOptions.Method
		if parseMethod == "" {
			parseMethod = "GET"
		}
		parseRequestOptions.Set("method", parseMethod)

		parseBodyValue, isFormData, parseErr := bodyToJSValue(parseOptions.Body)
		if parseErr != nil {
			sendResult(Result{Err: parseErr})
			return
		}

		if parseOptions.Headers != nil {
			parseHeaders := js.Global().Get("Object").New()
			for parseK, parseV := range parseOptions.Headers {
				if isFormData && strings.EqualFold(parseK, "Content-Type") {
					continue
				}
				parseHeaders.Set(parseK, fmt.Sprint(parseV))
			}
			parseRequestOptions.Set("headers", parseHeaders)
		}

		if !parseBodyValue.IsUndefined() && !parseBodyValue.IsNull() {
			parseRequestOptions.Set("body", parseBodyValue)
		}

		parsePromise := parseFetchFunction.Invoke(parseUrl, parseRequestOptions)

		var parseBodyThen js.Func
		var parseBodyCatch js.Func
		var parseResolve js.Func
		var parseReject js.Func

		parseResolve = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Fetch callback")
			defer parseResolve.Release()
			defer parseReject.Release()

			parseResp := parseArgs[0]
			parseStatus := parseResp.Get("status").Int()
			parseStatusText := parseResp.Get("statusText").String()
			parseHeaders2 := responseHeadersToMap(parseResp.Get("headers"))
			parseTextPromise := parseResp.Call("text")

			parseBodyThen = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
				defer gwcruntime.RecoverContainedPanic("fetch", "Fetch callback")
				defer parseBodyThen.Release()
				defer parseBodyCatch.Release()
				parseResult := Result{
					Status:  parseStatus,
					Headers: parseHeaders2,
				}
				if len(parseArgs2) > 0 {
					parseResult.Data = parseArgs2[0].String()
					if parseStatus < 200 || parseStatus >= 300 {
						parseResult.Err = HTTPError{Status: parseStatus, StatusText: parseStatusText, Body: parseResult.Text(), Headers: parseHeaders2}
					}
					parseCh <- parseResult
					return nil
				}
				parseResult.Err = errors.New("empty response from fetch")
				parseCh <- parseResult
				return nil
			})

			parseBodyCatch = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
				defer gwcruntime.RecoverContainedPanic("fetch", "Fetch callback")
				defer parseBodyThen.Release()
				defer parseBodyCatch.Release()
				parseCh <- Result{
					Status:  parseStatus,
					Headers: parseHeaders2,
					Err:     fmt.Errorf("failed to read body: %v", parseArgs3),
				}
				return nil
			})

			parseTextPromise.Call("then", parseBodyThen)
			parseTextPromise.Call("catch", parseBodyCatch)
			return nil
		})

		parseReject = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Fetch callback")
			defer parseResolve.Release()
			defer parseReject.Release()
			parseCh <- Result{Err: fmt.Errorf("fetch failed: %v", parseArgs4)}
			return nil
		})

		parsePromise.Call("then", parseResolve)
		parsePromise.Call("catch", parseReject)
	}()

	return parseCh
}

// Upload performs an XHR-backed upload so callers can observe progress and cancel through context.
func Upload(parseCtx context.Context, parseUrl string, parseOptions Options) <-chan UploadUpdate {
	parseCh := make(chan UploadUpdate, 8)

	go func() {
		var parseProgressFn js.Func
		var parseLoadFn js.Func
		var parseErrorFn js.Func
		var parseAbortFn js.Func
		var isProgressRegistered bool
		var isLoadRegistered bool
		var isErrorRegistered bool
		var isAbortRegistered bool
		var parseOnce sync.Once
		parseDone := make(chan struct{})
		parseCleanup := func() {
			if isProgressRegistered {
				parseProgressFn.Release()
			}
			if isLoadRegistered {
				parseLoadFn.Release()
			}
			if isErrorRegistered {
				parseErrorFn.Release()
			}
			if isAbortRegistered {
				parseAbortFn.Release()
			}
		}
		// parseFinalize emits the single terminal update, closes parseCh, and
		// releases the listener funcs — exactly once. Hoisted above every
		// throwable js call (New/open/setRequestHeader/send all throw
		// synchronously on bad input) so the recover below can settle the
		// channel; without it a synchronous XHR throw left parseCh unclosed and
		// any range-based consumer hanging forever.
		parseFinalize := func(parseUpdate2 UploadUpdate) {
			parseOnce.Do(func() {
				close(parseDone)
				parseCh <- parseUpdate2
				close(parseCh)
				parseCleanup()
			})
		}
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				gwcruntime.ContainPanic("fetch", gwcruntime.PanicPhaseAsync, "Upload request", parseRecovered)
				parseFinalize(UploadUpdate{Done: true, Result: Result{Err: fmt.Errorf("upload request failed: %v", parseRecovered)}})
			}
		}()

		parseXhrCtor := js.Global().Get("XMLHttpRequest")
		if !parseXhrCtor.Truthy() {
			parseFinalize(UploadUpdate{Done: true, Result: Result{Err: errors.New("XMLHttpRequest unavailable in this environment")}})
			return
		}

		parseBodyValue, isFormData, parseErr := bodyToJSValue(parseOptions.Body)
		if parseErr != nil {
			parseFinalize(UploadUpdate{Done: true, Result: Result{Err: parseErr}})
			return
		}

		parseXhr := parseXhrCtor.New()
		parseMethod := parseOptions.Method
		if parseMethod == "" {
			parseMethod = "POST"
		}
		parseXhr.Call("open", parseMethod, parseUrl, true)
		if parseOptions.Headers != nil {
			for parseK, parseV := range parseOptions.Headers {
				if isFormData && strings.EqualFold(parseK, "Content-Type") {
					continue
				}
				parseXhr.Call("setRequestHeader", parseK, fmt.Sprint(parseV))
			}
		}

		parseProgressFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Upload callback")
			if len(parseArgs) == 0 {
				return nil
			}
			parseEvent := parseArgs[0]
			parseUpdate := UploadUpdate{
				Loaded:           int64(parseEvent.Get("loaded").Float()),
				Total:            int64(parseEvent.Get("total").Float()),
				LengthComputable: parseEvent.Get("lengthComputable").Truthy(),
			}
			select {
			case parseCh <- parseUpdate:
			default:
			}
			return nil
		})
		isProgressRegistered = true

		parseLoadFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Upload callback")
			parseStatus := parseXhr.Get("status").Int()
			parseStatusText := parseXhr.Get("statusText").String()
			parseResponseText := parseXhr.Get("responseText")
			parseHeaders := parseRawHeaders(parseXhr.Call("getAllResponseHeaders").String())
			parseResult := Result{
				Data:    parseResponseText.String(),
				Status:  parseStatus,
				Headers: parseHeaders,
			}
			if parseStatus < 200 || parseStatus >= 300 {
				parseResult.Err = HTTPError{Status: parseStatus, StatusText: parseStatusText, Body: parseResult.Text(), Headers: parseHeaders}
			}
			parseLoaded, parseTotal, parseComputable := progressFromEvent(parseArgs2)
			parseFinalize(UploadUpdate{
				Loaded:           parseLoaded,
				Total:            parseTotal,
				LengthComputable: parseComputable,
				Done:             true,
				Result:           parseResult,
			})
			return nil
		})
		isLoadRegistered = true

		parseErrorFn = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Upload callback")
			parseFinalize(UploadUpdate{Done: true, Result: Result{Err: errors.New("upload request failed")}})
			return nil
		})
		isErrorRegistered = true

		parseAbortFn = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			defer gwcruntime.RecoverContainedPanic("fetch", "Upload callback")
			parseErr2 := errors.New("upload aborted")
			if parseCtx != nil && parseCtx.Err() != nil {
				parseErr2 = parseCtx.Err()
			}
			parseFinalize(UploadUpdate{Done: true, Result: Result{Err: parseErr2}})
			return nil
		})
		isAbortRegistered = true

		if parseUpload := parseXhr.Get("upload"); parseUpload.Truthy() {
			parseUpload.Call("addEventListener", "progress", parseProgressFn)
		}
		parseXhr.Call("addEventListener", "load", parseLoadFn)
		parseXhr.Call("addEventListener", "error", parseErrorFn)
		parseXhr.Call("addEventListener", "abort", parseAbortFn)

		if parseCtx != nil {
			go func() {
				defer gwcruntime.RecoverContainedPanic("fetch", "Upload abort watcher")
				select {
				case <-parseCtx.Done():
					parseXhr.Call("abort")
				case <-parseDone:
				}
			}()
		}

		if !parseBodyValue.IsUndefined() && !parseBodyValue.IsNull() {
			parseXhr.Call("send", parseBodyValue)
		} else {
			parseXhr.Call("send")
		}
	}()

	return parseCh
}

// responseHeadersToMap is a core package helper.
func responseHeadersToMap(parseHeaders js.Value) map[string]string {
	if parseHeaders.IsUndefined() || parseHeaders.IsNull() {
		return nil
	}
	parseValues := map[string]string{}
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer gwcruntime.RecoverContainedPanic("fetch", "responseHeadersToMap callback")
		if len(parseArgs) < 2 {
			return nil
		}
		parseValues[parseArgs[1].String()] = parseArgs[0].String()
		return nil
	})
	defer parseCallback.Release()
	parseHeaders.Call("forEach", parseCallback)
	if len(parseValues) == 0 {
		return nil
	}
	return parseValues
}

// progressFromEvent is a core package helper.
func progressFromEvent(parseArgs []js.Value) (parseLoaded int64, parseTotal int64, isComputable bool) {
	if len(parseArgs) == 0 {
		return 0, 0, false
	}
	parseEvent := parseArgs[0]
	if parseEvent.IsUndefined() || parseEvent.IsNull() {
		return 0, 0, false
	}
	parseLoaded = int64(parseEvent.Get("loaded").Float())
	parseTotal = int64(parseEvent.Get("total").Float())
	isComputable = parseEvent.Get("lengthComputable").Truthy()
	return parseLoaded, parseTotal, isComputable
}

// bodyToJSValue is a core package helper.
func bodyToJSValue(parseBody interface{}) (js.Value, bool, error) {
	if parseBody == nil {
		return js.Undefined(), false, nil
	}
	switch parseTyped := parseBody.(type) {
	case string:
		return js.ValueOf(parseTyped), false, nil
	case MultipartBody:
		parseValue, parseErr := buildMultipartFormData(parseTyped)
		return parseValue, true, parseErr
	case *MultipartBody:
		if parseTyped == nil {
			return js.Undefined(), false, nil
		}
		parseValue2, parseErr2 := buildMultipartFormData(*parseTyped)
		return parseValue2, true, parseErr2
	default:
		parseEncoded, parseErr3 := json.Marshal(parseBody)
		if parseErr3 != nil {
			return js.Undefined(), false, fmt.Errorf("failed to encode body: %w", parseErr3)
		}
		return js.ValueOf(string(parseEncoded)), false, nil
	}
}

// buildMultipartFormData is a core package helper.
func buildMultipartFormData(parseBody MultipartBody) (js.Value, error) {
	parseFormDataCtor := js.Global().Get("FormData")
	if !parseFormDataCtor.Truthy() {
		return js.Undefined(), errors.New("FormData unavailable in this environment")
	}
	parseForm := parseFormDataCtor.New()
	for parseKey, parseValue := range parseBody.Fields {
		parseForm.Call("append", parseKey, parseValue)
	}
	for _, parseFile := range parseBody.Files {
		if strings.TrimSpace(parseFile.FieldName) == "" {
			continue
		}
		parseRaw := parseFile.File.JSValue()
		if parseRaw.IsUndefined() || parseRaw.IsNull() {
			continue
		}
		if strings.TrimSpace(parseFile.Filename) != "" {
			parseForm.Call("append", parseFile.FieldName, parseRaw, parseFile.Filename)
		} else {
			parseForm.Call("append", parseFile.FieldName, parseRaw)
		}
	}
	return parseForm, nil
}
