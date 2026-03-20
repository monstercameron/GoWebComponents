//go:build !js || !wasm
// +build !js !wasm

package atlas

import (
	"context"
	"net/url"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
)

type atlasLocalState[T any] struct {
	value T
}

func useAtlasState[T any](initial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: initial}
}

func (s *atlasLocalState[T]) Get() T {
	return s.value
}

func (s *atlasLocalState[T]) Set(value T) {
	s.value = value
}

func useAtlasEffect(effect func() func(), deps ...interface{}) {
}

type atlasComputed[T any] struct {
	value T
}

type atlasAtom[T any] struct {
	value T
}

type atlasRevalidator struct{}

type atlasTransition struct {
	pending func() bool
	start   func(func())
}

type atlasThrottled[T any] struct {
	get     func() T
	pending func() bool
}

type atlasSearchParams struct{}

type atlasViewportMetrics struct {
	ScrollY       int
	Width         int
	Height        int
	SampleCount   int
	MeasuredAtUTC string
}

func useAtlasAtom[T any](id string, initial T) atlasAtom[T] {
	return atlasAtom[T]{value: initial}
}

func (a atlasAtom[T]) Get() T {
	return a.value
}

func (a atlasAtom[T]) Set(value T) {
}

func useAtlasComputed[T any](compute func() T, deps ...interface{}) atlasComputed[T] {
	return atlasComputed[T]{value: compute()}
}

func (c atlasComputed[T]) Get() T {
	return c.value
}

func useAtlasRevalidator() atlasRevalidator {
	return atlasRevalidator{}
}

func (r atlasRevalidator) Revalidate() {
}

func (r atlasRevalidator) Loading() bool {
	return false
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
	return atlasViewportMetrics{}
}

func useAtlasSearchParams() atlasSearchParams {
	return atlasSearchParams{}
}

func (s atlasSearchParams) Values() url.Values {
	return url.Values{}
}

func (s atlasSearchParams) ReplaceAll(values url.Values) {
}

func useAtlasCachedResource[T any](key string, loader func(context.Context) (T, error)) atlasCachedResource[T] {
	return atlasCachedResource[T]{}
}

func useAtlasResource[T any](loader func(context.Context) (T, error), deps ...interface{}) atlasResource[T] {
	return atlasResource[T]{}
}

func atlasFetch(url string, options atlasFetchOptions) <-chan atlasImperativeFetchResult {
	resultCh := make(chan atlasImperativeFetchResult, 1)
	resultCh <- atlasImperativeFetchResult{Error: "fetch unavailable in native atlas build"}
	return resultCh
}

func useAtlasWorkerTask[Request any, Progress any, Result any](options interop.WorkerOptions, name string) atlasWorkerTask[Request, Progress, Result] {
	return atlasWorkerTask[Request, Progress, Result]{}
}

func useAtlasChannel[T any](ch <-chan T) atlasChannelValue[T] {
	return atlasChannelValue[T]{}
}

func persistAtlasSnapshot(key string, atomIDs ...string) error {
	return nil
}
