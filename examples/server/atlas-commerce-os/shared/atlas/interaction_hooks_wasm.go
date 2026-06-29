//go:build js && wasm

package atlas

import (
	"context"
	"net/url"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/fetch"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type atlasLocalState[T any] struct {
	value ui.State[T]
}

func useAtlasState[T any](parseInitial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: ui.UseState(parseInitial)}
}

func (parseS *atlasLocalState[T]) Get() T {
	return parseS.value.Get()
}

func (parseS *atlasLocalState[T]) Set(parseValue T) {
	parseS.value.Set(parseValue)
}

func useAtlasEffect(parseEffect func() func(), parseDeps ...interface{}) {
	ui.UseEffect(parseEffect, parseDeps...)
}

type atlasComputed[T any] struct {
	get func() T
}

type atlasAtom[T any] struct {
	get func() T
	set func(T)
}

type atlasRevalidator struct {
	revalidate func()
	loading    func() bool
}

type atlasTransition struct {
	pending func() bool
	start   func(func())
}

type atlasThrottled[T any] struct {
	get     func() T
	pending func() bool
}

type atlasSearchParams struct {
	values     func() url.Values
	replaceAll func(url.Values)
}

type atlasViewportMetrics struct {
	ScrollY       int
	Width         int
	Height        int
	SampleCount   int
	MeasuredAtUTC string
}

func useAtlasAtom[T any](parseId string, parseInitial T) atlasAtom[T] {
	parseAtom := state.UseAtom(parseId, parseInitial)
	return atlasAtom[T]{get: parseAtom.Get, set: parseAtom.Set}
}

func (parseA atlasAtom[T]) Get() T {
	if parseA.get == nil {
		var parseZero T
		return parseZero
	}
	return parseA.get()
}

func (parseA atlasAtom[T]) Set(parseValue T) {
	if parseA.set != nil {
		parseA.set(parseValue)
	}
}

func useAtlasComputed[T any](parseCompute func() T, parseDeps ...interface{}) atlasComputed[T] {
	parseComputed := state.UseComputed(parseCompute, parseDeps...)
	return atlasComputed[T]{get: parseComputed.Get}
}

func (parseC atlasComputed[T]) Get() T {
	if parseC.get == nil {
		var parseZero T
		return parseZero
	}
	return parseC.get()
}

func useAtlasRevalidator() atlasRevalidator {
	parseRevalidator := router.UseRevalidator()
	return atlasRevalidator{
		revalidate: parseRevalidator.Revalidate,
		loading:    parseRevalidator.Loading,
	}
}

func (parseR atlasRevalidator) Revalidate() {
	if parseR.revalidate != nil {
		parseR.revalidate()
	}
}

func (parseR atlasRevalidator) Loading() bool {
	if parseR.loading == nil {
		return false
	}
	return parseR.loading()
}

func startAtlasTransition(parseFn func()) {
	ui.StartTransition(parseFn)
}

func useAtlasTransition() atlasTransition {
	parseTransition := ui.UseTransition()
	return atlasTransition{
		pending: parseTransition.Pending,
		start:   parseTransition.Start,
	}
}

func (parseT atlasTransition) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

func (parseT atlasTransition) Start(parseFn func()) {
	if parseT.start != nil {
		parseT.start(parseFn)
		return
	}
	startAtlasTransition(parseFn)
}

func useAtlasThrottled[T any](parseValue T, parseInterval time.Duration) atlasThrottled[T] {
	parseThrottled := ui.UseThrottled(parseValue, parseInterval)
	return atlasThrottled[T]{
		get:     parseThrottled.Get,
		pending: parseThrottled.Pending,
	}
}

func (parseT atlasThrottled[T]) Get() T {
	if parseT.get == nil {
		var parseZero T
		return parseZero
	}
	return parseT.get()
}

func (parseT atlasThrottled[T]) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

func useAtlasViewportMetrics() atlasViewportMetrics {
	parseMetrics := ui.UseState(readAtlasViewportMetrics())
	useAtlasEffect(func() func() {
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() || !parseWindow.Get("addEventListener").Truthy() {
			return nil
		}
		parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseMetrics.Update(func(parsePrev atlasViewportMetrics) atlasViewportMetrics {
				parseNext := readAtlasViewportMetrics()
				parseNext.SampleCount = parsePrev.SampleCount + 1
				return parseNext
			})
			return nil
		})
		parseMetrics.Update(func(parsePrev2 atlasViewportMetrics) atlasViewportMetrics {
			parseNext2 := readAtlasViewportMetrics()
			if parsePrev2.SampleCount > 0 {
				parseNext2.SampleCount = parsePrev2.SampleCount
			}
			return parseNext2
		})
		parseWindow.Call("addEventListener", "scroll", parseHandler)
		parseWindow.Call("addEventListener", "resize", parseHandler)
		return func() {
			parseWindow.Call("removeEventListener", "scroll", parseHandler)
			parseWindow.Call("removeEventListener", "resize", parseHandler)
			parseHandler.Release()
		}
	}, nil)
	return parseMetrics.Get()
}

func readAtlasViewportMetrics() atlasViewportMetrics {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return atlasViewportMetrics{}
	}
	parseWidth := 0
	parseHeight := 0
	parseScrollY := 0
	if parseValue := parseWindow.Get("innerWidth"); parseValue.Truthy() {
		parseWidth = parseValue.Int()
	}
	if parseValue2 := parseWindow.Get("innerHeight"); parseValue2.Truthy() {
		parseHeight = parseValue2.Int()
	}
	if parseValue3 := parseWindow.Get("scrollY"); parseValue3.Truthy() {
		parseScrollY = parseValue3.Int()
	}
	return atlasViewportMetrics{
		ScrollY:       parseScrollY,
		Width:         parseWidth,
		Height:        parseHeight,
		SampleCount:   1,
		MeasuredAtUTC: time.Now().UTC().Format(time.RFC3339),
	}
}

func useAtlasSearchParams() atlasSearchParams {
	parseParams := router.UseSearchParams()
	return atlasSearchParams{
		values:     parseParams.Values,
		replaceAll: parseParams.ReplaceAll,
	}
}

func (parseS atlasSearchParams) Values() url.Values {
	if parseS.values == nil {
		return url.Values{}
	}
	return parseS.values()
}

func (parseS atlasSearchParams) ReplaceAll(parseValues url.Values) {
	if parseS.replaceAll != nil {
		parseS.replaceAll(parseValues)
	}
}

func useAtlasCachedResource[T any](parseKey string, parseLoader func(context.Context) (T, error)) atlasCachedResource[T] {
	parseResource := fetch.UseCachedResource(parseKey, parseLoader, fetch.CacheOptions{
		StaleAfter:   5 * time.Minute,
		MaxAge:       30 * time.Minute,
		DisposeAfter: 90 * time.Minute,
	})
	return atlasCachedResource[T]{
		get: func() atlasCachedResourceState[T] {
			parseState := parseResource.Get()
			return atlasCachedResourceState[T]{
				Value:   parseState.Value,
				Loading: parseState.Loading,
				Error:   parseState.Error,
				Ready:   parseState.Ready,
				Stale:   parseState.Stale,
			}
		},
		reload: parseResource.Reload,
		set:    parseResource.Set,
		update: parseResource.Update,
	}
}

func useAtlasResource[T any](parseLoader func(context.Context) (T, error), parseDeps ...interface{}) atlasResource[T] {
	parseResource := fetch.UseResource(parseLoader, parseDeps...)
	return atlasResource[T]{
		get: func() atlasResourceState[T] {
			parseState := parseResource.Get()
			return atlasResourceState[T]{
				Value:   parseState.Value,
				Loading: parseState.Loading,
				Error:   parseState.Error,
				Ready:   parseState.Ready,
			}
		},
		reload: parseResource.Reload,
	}
}

func atlasFetch(parseUrl string, parseOptions atlasFetchOptions) <-chan atlasImperativeFetchResult {
	parseResultCh := make(chan atlasImperativeFetchResult, 1)
	go func() {
		parseFetchCh := fetch.Fetch(parseUrl, fetch.Options{
			Method:  parseOptions.Method,
			Headers: parseOptions.Headers,
			Body:    parseOptions.Body,
		})
		parseResult := <-parseFetchCh
		fetch.ReturnChannel(parseFetchCh)
		parsePayload := atlasImperativeFetchResult{
			Data:    parseResult.Text(),
			Status:  parseResult.Status,
			Headers: parseResult.Headers,
		}
		if parseResult.Err != nil {
			parsePayload.Error = parseResult.Err.Error()
		}
		parseResultCh <- parsePayload
	}()
	return parseResultCh
}

func useAtlasWorkerTask[Request any, Progress any, Result any](parseOptions interop.WorkerOptions, parseName string) atlasWorkerTask[Request, Progress, Result] {
	parseTask := ui.UseWorkerTask[Request, Progress, Result](parseOptions, parseName)
	return atlasWorkerTask[Request, Progress, Result]{
		get: func() atlasWorkerTaskState[Progress, Result] {
			parseState := parseTask.Get()
			return atlasWorkerTaskState[Progress, Result]{
				Value:         parseState.Value,
				Progress:      parseState.Progress,
				ProgressReady: parseState.ProgressReady,
				Running:       parseState.Running,
				Ready:         parseState.Ready,
				Cancelled:     parseState.Cancelled,
				Started:       parseState.Started,
				Error:         parseState.Error,
			}
		},
		start:  parseTask.Start,
		cancel: parseTask.Cancel,
	}
}

func useAtlasChannel[T any](parseCh <-chan T) atlasChannelValue[T] {
	parseChannel := ui.UseChannel(parseCh)
	return atlasChannelValue[T]{
		get:    parseChannel.Get,
		ok:     parseChannel.Ok,
		closed: parseChannel.Closed,
	}
}

func persistAtlasSnapshot(parseKey string, parseAtomIDs ...string) error {
	parseSnap, parseErr := state.GetSnapshot()
	if parseErr != nil {
		return parseErr
	}
	return state.SaveSnapshot(parseKey, parseSnap.Select(parseAtomIDs...), state.LocalStorage)
}
