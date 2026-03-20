//go:build js && wasm
// +build js,wasm

package atlas

import (
	"context"
	"net/url"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

type atlasLocalState[T any] struct {
	value ui.State[T]
}

func useAtlasState[T any](initial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: ui.UseState(initial)}
}

func (s *atlasLocalState[T]) Get() T {
	return s.value.Get()
}

func (s *atlasLocalState[T]) Set(value T) {
	s.value.Set(value)
}

func useAtlasEffect(effect func() func(), deps ...interface{}) {
	ui.UseEffect(effect, deps...)
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

func useAtlasAtom[T any](id string, initial T) atlasAtom[T] {
	atom := state.UseAtom(id, initial)
	return atlasAtom[T]{get: atom.Get, set: atom.Set}
}

func (a atlasAtom[T]) Get() T {
	if a.get == nil {
		var zero T
		return zero
	}
	return a.get()
}

func (a atlasAtom[T]) Set(value T) {
	if a.set != nil {
		a.set(value)
	}
}

func useAtlasComputed[T any](compute func() T, deps ...interface{}) atlasComputed[T] {
	computed := state.UseComputed(compute, deps...)
	return atlasComputed[T]{get: computed.Get}
}

func (c atlasComputed[T]) Get() T {
	if c.get == nil {
		var zero T
		return zero
	}
	return c.get()
}

func useAtlasRevalidator() atlasRevalidator {
	revalidator := router.UseRevalidator()
	return atlasRevalidator{
		revalidate: revalidator.Revalidate,
		loading:    revalidator.Loading,
	}
}

func (r atlasRevalidator) Revalidate() {
	if r.revalidate != nil {
		r.revalidate()
	}
}

func (r atlasRevalidator) Loading() bool {
	if r.loading == nil {
		return false
	}
	return r.loading()
}

func startAtlasTransition(fn func()) {
	ui.StartTransition(fn)
}

func useAtlasTransition() atlasTransition {
	transition := ui.UseTransition()
	return atlasTransition{
		pending: transition.Pending,
		start:   transition.Start,
	}
}

func (t atlasTransition) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

func (t atlasTransition) Start(fn func()) {
	if t.start != nil {
		t.start(fn)
		return
	}
	startAtlasTransition(fn)
}

func useAtlasThrottled[T any](value T, interval time.Duration) atlasThrottled[T] {
	throttled := ui.UseThrottled(value, interval)
	return atlasThrottled[T]{
		get:     throttled.Get,
		pending: throttled.Pending,
	}
}

func (t atlasThrottled[T]) Get() T {
	if t.get == nil {
		var zero T
		return zero
	}
	return t.get()
}

func (t atlasThrottled[T]) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

func useAtlasViewportMetrics() atlasViewportMetrics {
	metrics := ui.UseState(readAtlasViewportMetrics())
	useAtlasEffect(func() func() {
		window := js.Global().Get("window")
		if !window.Truthy() || !window.Get("addEventListener").Truthy() {
			return nil
		}
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			metrics.Update(func(prev atlasViewportMetrics) atlasViewportMetrics {
				next := readAtlasViewportMetrics()
				next.SampleCount = prev.SampleCount + 1
				return next
			})
			return nil
		})
		metrics.Update(func(prev atlasViewportMetrics) atlasViewportMetrics {
			next := readAtlasViewportMetrics()
			if prev.SampleCount > 0 {
				next.SampleCount = prev.SampleCount
			}
			return next
		})
		window.Call("addEventListener", "scroll", handler)
		window.Call("addEventListener", "resize", handler)
		return func() {
			window.Call("removeEventListener", "scroll", handler)
			window.Call("removeEventListener", "resize", handler)
			handler.Release()
		}
	}, nil)
	return metrics.Get()
}

func readAtlasViewportMetrics() atlasViewportMetrics {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return atlasViewportMetrics{}
	}
	width := 0
	height := 0
	scrollY := 0
	if value := window.Get("innerWidth"); value.Truthy() {
		width = value.Int()
	}
	if value := window.Get("innerHeight"); value.Truthy() {
		height = value.Int()
	}
	if value := window.Get("scrollY"); value.Truthy() {
		scrollY = value.Int()
	}
	return atlasViewportMetrics{
		ScrollY:       scrollY,
		Width:         width,
		Height:        height,
		SampleCount:   1,
		MeasuredAtUTC: time.Now().UTC().Format(time.RFC3339),
	}
}

func useAtlasSearchParams() atlasSearchParams {
	params := router.UseSearchParams()
	return atlasSearchParams{
		values:     params.Values,
		replaceAll: params.ReplaceAll,
	}
}

func (s atlasSearchParams) Values() url.Values {
	if s.values == nil {
		return url.Values{}
	}
	return s.values()
}

func (s atlasSearchParams) ReplaceAll(values url.Values) {
	if s.replaceAll != nil {
		s.replaceAll(values)
	}
}

func useAtlasCachedResource[T any](key string, loader func(context.Context) (T, error)) atlasCachedResource[T] {
	resource := fetch.UseCachedResource(key, loader, fetch.CacheOptions{
		StaleAfter:   5 * time.Minute,
		MaxAge:       30 * time.Minute,
		DisposeAfter: 90 * time.Minute,
	})
	return atlasCachedResource[T]{
		get: func() atlasCachedResourceState[T] {
			state := resource.Get()
			return atlasCachedResourceState[T]{
				Value:   state.Value,
				Loading: state.Loading,
				Error:   state.Error,
				Ready:   state.Ready,
				Stale:   state.Stale,
			}
		},
		reload: resource.Reload,
		set:    resource.Set,
		update: resource.Update,
	}
}

func useAtlasResource[T any](loader func(context.Context) (T, error), deps ...interface{}) atlasResource[T] {
	resource := fetch.UseResource(loader, deps...)
	return atlasResource[T]{
		get: func() atlasResourceState[T] {
			state := resource.Get()
			return atlasResourceState[T]{
				Value:   state.Value,
				Loading: state.Loading,
				Error:   state.Error,
				Ready:   state.Ready,
			}
		},
		reload: resource.Reload,
	}
}

func atlasFetch(url string, options atlasFetchOptions) <-chan atlasImperativeFetchResult {
	resultCh := make(chan atlasImperativeFetchResult, 1)
	go func() {
		fetchCh := fetch.Fetch(url, fetch.Options{
			Method:  options.Method,
			Headers: options.Headers,
			Body:    options.Body,
		})
		result := <-fetchCh
		fetch.ReturnChannel(fetchCh)
		payload := atlasImperativeFetchResult{
			Data:    result.Text(),
			Status:  result.Status,
			Headers: result.Headers,
		}
		if result.Err != nil {
			payload.Error = result.Err.Error()
		}
		resultCh <- payload
	}()
	return resultCh
}

func useAtlasWorkerTask[Request any, Progress any, Result any](options interop.WorkerOptions, name string) atlasWorkerTask[Request, Progress, Result] {
	task := ui.UseWorkerTask[Request, Progress, Result](options, name)
	return atlasWorkerTask[Request, Progress, Result]{
		get: func() atlasWorkerTaskState[Progress, Result] {
			state := task.Get()
			return atlasWorkerTaskState[Progress, Result]{
				Value:         state.Value,
				Progress:      state.Progress,
				ProgressReady: state.ProgressReady,
				Running:       state.Running,
				Ready:         state.Ready,
				Cancelled:     state.Cancelled,
				Started:       state.Started,
				Error:         state.Error,
			}
		},
		start:  task.Start,
		cancel: task.Cancel,
	}
}

func useAtlasChannel[T any](ch <-chan T) atlasChannelValue[T] {
	channel := ui.UseChannel(ch)
	return atlasChannelValue[T]{
		get:    channel.Get,
		ok:     channel.Ok,
		closed: channel.Closed,
	}
}

func persistAtlasSnapshot(key string, atomIDs ...string) error {
	snapshot := state.ExportSnapshot().Select(atomIDs...)
	return state.SaveSnapshot(key, snapshot, state.LocalStorage)
}
