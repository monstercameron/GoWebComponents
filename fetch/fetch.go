package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v6/deprecation"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Options represents configuration for HTTP fetch operations.
type Options struct {
	Method string
	// Headers are request headers. Values are typed any (not string) so callers can pass
	// non-string values that are stringified at send time; response headers in Result are
	// always plain strings.
	Headers map[string]any
	Body    any
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
	Data   any
	Status int
	// Headers are response headers, always plain strings (unlike Options.Headers, whose
	// values are typed any for request-side flexibility).
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
	if parseR.Data == nil {
		return ""
	}
	if parseText, parseOk := parseR.Data.(string); parseOk {
		return parseText
	}
	return fmt.Sprint(parseR.Data)
}

// DecodeJSON decodes a string-backed response body into target.
func (parseR Result) DecodeJSON(parseTarget any) error {
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

type resourceStateSink[T any] interface {
	Set(ResourceState[T])
	Update(func(ResourceState[T]) ResourceState[T])
}

type resourceCancelRef interface {
	Get() context.CancelFunc
	Set(context.CancelFunc)
}

// UseFetch is a hook that simplifies data fetching within a component.
// It uses the runtime fetch hook directly.
//
// Deprecated: prefer the typed UseResource[T] (a Go loader returning a typed value) or ui.UseQuery
// (cache + dedupe + stale-while-revalidate). UseFetch remains for a quick untyped URL GET.
func UseFetch(parseUrl string, parseOptions ...Options) Resource {
	parseArgs := make([]any, len(parseOptions))
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
func UseResource[T any](parseLoader func(context.Context) (T, error), parseDeps ...any) AsyncResource[T] {
	parseState := ui.UseState(ResourceState[T]{})
	parseReloadTick := ui.UseState(0)
	parseCancelRef := ui.UseRef((context.CancelFunc)(nil))
	// parseRequestSeqRef is rendered as an atomic so the loader goroutine can
	// safely read it on native builds without a data race with the render thread.
	parseRequestSeqRef := ui.UseRef((*atomic.Int32)(nil))
	if parseRequestSeqRef.Get() == nil {
		parseRequestSeqRef.Set(new(atomic.Int32))
	}
	parseRequestSeq := parseRequestSeqRef.Get()

	parseStartLoad := func() {
		startResourceLoad(parseLoader, parseState, parseCancelRef, parseRequestSeq)
	}

	parseEffectDeps := make([]any, 0, len(parseDeps)+1)
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

func startResourceLoad[T any](parseLoader func(context.Context) (T, error), parseState resourceStateSink[T], parseCancelRef resourceCancelRef, parseRequestSeq *atomic.Int32) {
	if parseCancel := parseCancelRef.Get(); parseCancel != nil {
		parseCancel()
	}

	parseSeq := parseRequestSeq.Add(1)
	parseCtx, parseCancel2 := context.WithCancel(context.Background())
	parseCancelRef.Set(parseCancel2)

	parseState.Update(func(parsePrev ResourceState[T]) ResourceState[T] {
		parsePrev.Loading = true
		parsePrev.Error = nil
		return parsePrev
	})

	go func() {
		defer runtime.RecoverContainedPanic("fetch", "UseResource loader")
		parseValue, parseErr := parseLoader(parseCtx)
		if parseCtx.Err() != nil || parseRequestSeq.Load() != parseSeq {
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

// ReturnChannel returns a fetch result channel to the pool for reuse.
// With the new implementation channels are one-shot, so this is a no-op kept for API compatibility.
//
// Deprecated: channels are now one-shot and do not need to be returned. This function is a no-op and will be removed in a future release.
func ReturnChannel(parseCh <-chan Result) {
	deprecation.Warn("fetch.ReturnChannel", "")
	_ = parseCh
}
