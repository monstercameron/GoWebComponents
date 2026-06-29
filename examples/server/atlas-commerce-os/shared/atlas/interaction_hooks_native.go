//go:build !js || !wasm

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

func useAtlasState[T any](parseInitial T) *atlasLocalState[T] {
	return &atlasLocalState[T]{value: parseInitial}
}

func (parseS *atlasLocalState[T]) Get() T {
	return parseS.value
}

func (parseS *atlasLocalState[T]) Set(parseValue T) {
	parseS.value = parseValue
}

func useAtlasEffect(parseEffect func() func(), parseDeps ...any) {
	_ = parseEffect
	_ = parseDeps
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

func useAtlasAtom[T any](parseId string, parseInitial T) atlasAtom[T] {
	_ = parseId
	return atlasAtom[T]{value: parseInitial}
}

func (parseA atlasAtom[T]) Get() T {
	return parseA.value
}

func (parseA atlasAtom[T]) Set(parseValue T) {
	_ = parseValue
}

func useAtlasComputed[T any](parseCompute func() T, parseDeps ...any) atlasComputed[T] {
	_ = parseDeps
	return atlasComputed[T]{value: parseCompute()}
}

func (parseC atlasComputed[T]) Get() T {
	return parseC.value
}

func useAtlasRevalidator() atlasRevalidator {
	return atlasRevalidator{}
}

func (parseR atlasRevalidator) Revalidate() {
}

func (parseR atlasRevalidator) Loading() bool {
	return false
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
	return atlasViewportMetrics{}
}

func useAtlasSearchParams() atlasSearchParams {
	return atlasSearchParams{}
}

func (parseS atlasSearchParams) Values() url.Values {
	return url.Values{}
}

func (parseS atlasSearchParams) ReplaceAll(parseValues url.Values) {
	_ = parseValues
}

func useAtlasCachedResource[T any](parseKey string, parseLoader func(context.Context) (T, error)) atlasCachedResource[T] {
	_ = parseKey
	_ = parseLoader
	return atlasCachedResource[T]{}
}

func useAtlasResource[T any](parseLoader func(context.Context) (T, error), parseDeps ...any) atlasResource[T] {
	_ = parseLoader
	_ = parseDeps
	return atlasResource[T]{}
}

func atlasFetch(parseUrl string, parseOptions atlasFetchOptions) <-chan atlasImperativeFetchResult {
	_ = parseUrl
	_ = parseOptions
	parseResultCh := make(chan atlasImperativeFetchResult, 1)
	parseResultCh <- atlasImperativeFetchResult{Error: "fetch unavailable in native atlas build"}
	return parseResultCh
}

func useAtlasWorkerTask[Request any, Progress any, Result any](parseOptions interop.WorkerOptions, parseName string) atlasWorkerTask[Request, Progress, Result] {
	_ = parseOptions
	_ = parseName
	return atlasWorkerTask[Request, Progress, Result]{}
}

func useAtlasChannel[T any](parseCh <-chan T) atlasChannelValue[T] {
	_ = parseCh
	return atlasChannelValue[T]{}
}

func persistAtlasSnapshot(parseKey string, parseAtomIDs ...string) error {
	_ = parseKey
	_ = parseAtomIDs
	return nil
}
