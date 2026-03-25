//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Options represents configuration for HTTP fetch operations.
type Options struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

// MultipartFile describes one browser file to append to a multipart form-data body.
type MultipartFile struct {
	FieldName string
	File      ui.File
	Filename  string
}

// MultipartBody describes a multipart form-data payload with text fields and browser files.
type MultipartBody struct {
	Fields map[string]string
	Files  []MultipartFile
}

// State represents the state of a fetch operation managed by UseFetch.
type State = runtime.FetchState

// Result represents the result of a manual Fetch operation.
type Result struct {
	Data    interface{}
	Status  int
	Headers map[string]string
	Err     error
}

// UploadUpdate reports upload progress and the final upload result.
type UploadUpdate struct {
	Loaded           int64
	Total            int64
	LengthComputable bool
	Done             bool
	Result           Result
}

// HTTPError reports a non-success HTTP status while preserving the response metadata on Result.
type HTTPError struct {
	Status     int
	StatusText string
	Body       string
	Headers    map[string]string
}

// Error is a core package helper.
func (parseE HTTPError) Error() string {
	if parseE.Status <= 0 {
		return "request failed"
	}
	if parseStatusText := strings.TrimSpace(parseE.StatusText); parseStatusText != "" {
		return fmt.Sprintf("request failed with status %d %s", parseE.Status, parseStatusText)
	}
	return fmt.Sprintf("request failed with status %d", parseE.Status)
}

// Text returns the response body when it is string-backed.
func (parseR Result) Text() string {
	if parseText, parseOk := parseR.Data.(string); parseOk {
		return parseText
	}
	return fmt.Sprint(parseR.Data)
}

// DecodeJSON decodes a string-backed response body into target.
func (parseR Result) DecodeJSON(parseTarget interface{}) error {
	if parseTarget == nil {
		return errors.New("decode target is nil")
	}
	parseBody := strings.TrimSpace(parseR.Text())
	if parseBody == "" {
		return errors.New("response body is empty")
	}
	if parseErr := json.Unmarshal([]byte(parseBody), parseTarget); parseErr != nil {
		return fmt.Errorf("decode response json: %w", parseErr)
	}
	return nil
}

// Resource exposes the current low-level fetch state and a refetch helper.
type Resource struct {
	get     func() State
	refetch func()
}

// ResourceState describes the state of a typed async resource.
type ResourceState[T any] struct {
	Value   T
	Loading bool
	Error   error
	Ready   bool
}

// AsyncResource exposes the current typed resource state and lifecycle controls.
type AsyncResource[T any] struct {
	get    func() ResourceState[T]
	reload func()
	cancel func()
}

// UseFetch is a hook that simplifies data fetching within a component.
// It uses the runtime fetch hook directly.
func UseFetch(parseUrl string, parseOptions ...Options) Resource {
	parseArgs := make([]interface{}, len(parseOptions))
	for parseI, parseOpt := range parseOptions {
		parseArgs[parseI] = parseOpt
	}
	get, parseRefetch := runtime.GoUseFetch(parseUrl, parseArgs...)
	return Resource{get: get, refetch: parseRefetch}
}

// Get returns the current low-level fetch state.
func (parseR Resource) Get() State {
	return parseR.get()
}

// Refetch restarts the underlying fetch request.
func (parseR Resource) Refetch() {
	parseR.refetch()
}

// UseResource provides a typed async resource hook driven by a Go loader.
//
// The loader runs on mount and whenever deps or the reload token change. It
// receives a context that is cancelled when the component unmounts, the
// dependency list changes, or Cancel is called on the returned handle.
func UseResource[T any](parseLoader func(context.Context) (T, error), parseDeps ...interface{}) AsyncResource[T] {
	parseState := ui.UseState(ResourceState[T]{})
	parseReloadTick := ui.UseState(0)
	parseCancelRef := ui.UseRef((context.CancelFunc)(nil))
	parseRequestSeq := ui.UseRef(0)

	parseStartLoad := func() {
		if parseCancel := parseCancelRef.Get(); parseCancel != nil {
			parseCancel()
		}

		parseRequestSeq.Set(parseRequestSeq.Get() + 1)
		parseSeq := parseRequestSeq.Get()
		parseCtx, parseCancel2 := context.WithCancel(context.Background())
		parseCancelRef.Set(parseCancel2)

		parseState.Update(func(parsePrev ResourceState[T]) ResourceState[T] {
			parsePrev.Loading = true
			parsePrev.Error = nil
			return parsePrev
		})

		go func() {
			parseValue, parseErr := parseLoader(parseCtx)
			if parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
				return
			}

			parseState.Set(ResourceState[T]{
				Value:   parseValue,
				Loading: false,
				Error:   parseErr,
				Ready:   parseErr == nil,
			})
		}()
	}

	parseEffectDeps := make([]interface{}, 0, len(parseDeps)+1)
	parseEffectDeps = append(parseEffectDeps, parseReloadTick.Get())
	parseEffectDeps = append(parseEffectDeps, parseDeps...)

	ui.UseEffect(func() func() {
		parseStartLoad()
		return func() {
			if parseCancel3 := parseCancelRef.Get(); parseCancel3 != nil {
				parseCancel3()
				parseCancelRef.Set(nil)
			}
		}
	}, parseEffectDeps...)

	return AsyncResource[T]{
		get: func() ResourceState[T] { return parseState.Get() },
		reload: func() {
			parseReloadTick.Update(func(parsePrev2 int) int { return parsePrev2 + 1 })
		},
		cancel: func() {
			if parseCancel4 := parseCancelRef.Get(); parseCancel4 != nil {
				parseCancel4()
				parseCancelRef.Set(nil)
			}
			parseState.Update(func(parsePrev3 ResourceState[T]) ResourceState[T] {
				parsePrev3.Loading = false
				return parsePrev3
			})
		},
	}
}

// Get returns the current typed resource state.
func (parseR AsyncResource[T]) Get() ResourceState[T] {
	if parseR.get == nil {
		var parseZero ResourceState[T]
		return parseZero
	}

	return parseR.get()
}

// Reload starts a new resource load.
func (parseR AsyncResource[T]) Reload() {
	if parseR.reload != nil {
		parseR.reload()
	}
}

// Cancel cancels the active resource load, if any.
func (parseR AsyncResource[T]) Cancel() {
	if parseR.cancel != nil {
		parseR.cancel()
	}
}

// Fetch performs an asynchronous HTTP fetch operation and returns a channel for the result.
// This uses the browser Fetch API from WASM; it currently returns the response body as text.
func Fetch(parseUrl string, parseOptions Options) <-chan Result {
	parseCh := make(chan Result, 1)

	go func() {
		parseFetchFunction := js.Global().Get("fetch")
		if !parseFetchFunction.Truthy() {
			parseCh <- Result{Err: errors.New("fetch API unavailable in this environment")}
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
			parseCh <- Result{Err: parseErr}
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
			defer parseResolve.Release()
			defer parseReject.Release()

			parseResp := parseArgs[0]
			parseStatus := parseResp.Get("status").Int()
			parseStatusText := parseResp.Get("statusText").String()
			parseHeaders2 := responseHeadersToMap(parseResp.Get("headers"))
			parseTextPromise := parseResp.Call("text")

			parseBodyThen = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
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
		parseXhrCtor := js.Global().Get("XMLHttpRequest")
		if !parseXhrCtor.Truthy() {
			parseCh <- UploadUpdate{Done: true, Result: Result{Err: errors.New("XMLHttpRequest unavailable in this environment")}}
			close(parseCh)
			return
		}

		parseBodyValue, isFormData, parseErr := bodyToJSValue(parseOptions.Body)
		if parseErr != nil {
			parseCh <- UploadUpdate{Done: true, Result: Result{Err: parseErr}}
			close(parseCh)
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
		parseFinalize := func(parseUpdate2 UploadUpdate) {
			parseOnce.Do(func() {
				close(parseDone)
				parseCh <- parseUpdate2
				close(parseCh)
				parseCleanup()
			})
		}

		parseProgressFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
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
			parseFinalize(UploadUpdate{Done: true, Result: Result{Err: errors.New("upload request failed")}})
			return nil
		})
		isErrorRegistered = true

		parseAbortFn = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
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

// parseRawHeaders is a core package helper.
func parseRawHeaders(parseRaw string) map[string]string {
	parseLines := strings.Split(parseRaw, "\n")
	parseHeaders := make(map[string]string, len(parseLines))
	for _, parseLine := range parseLines {
		parseLine = strings.TrimSpace(parseLine)
		if parseLine == "" {
			continue
		}
		parseParts := strings.SplitN(parseLine, ":", 2)
		if len(parseParts) != 2 {
			continue
		}
		parseHeaders[strings.TrimSpace(parseParts[0])] = strings.TrimSpace(parseParts[1])
	}
	if len(parseHeaders) == 0 {
		return nil
	}
	return parseHeaders
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

// ReturnChannel returns a fetch result channel to the pool for reuse.
// With the new implementation channels are one-shot, so this is a no-op kept for API compatibility.
func ReturnChannel(parseCh <-chan Result) {
	_ = parseCh
}
