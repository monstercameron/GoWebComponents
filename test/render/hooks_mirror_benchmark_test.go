package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// These benchmarks mirror the Example 201 hooks-render scenario natively so
// the hook-slot machinery can be profiled with pprof: 40 leaf components,
// each registering 20x (UseState + UseEffect + UseMemo) = 60 hooks, 2400
// hook slots per render pass. Keep the shape in sync with
// renderBenchmarkManyHooks in examples/testing/render-benchmark.

const (
	mirrorHookComponents    = 40
	mirrorHooksPerComponent = 20
)

type mirrorHookCellProps struct {
	GetIndex        int
	GetRefreshToken int
}

// newMirrorHooksHost mounts the hook grid and returns the token setter that
// drives re-renders plus the scheduler used to flush scheduled work. With
// useTyped the cells construct through ui.Typed (static dispatch) instead of
// ui.CreateElement's reflect trampoline — the A/B for typed registration.
func mirrorHookStaticEffect() func() { return nil }
func mirrorHookStaticMemo(parseDep int) int { return parseDep * 2 }

func newMirrorHooksHost(parseB *testing.B, useTyped bool, useTypedHooks bool) (func(int), *mockdom.MockScheduler) {
	parseB.Helper()
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div")

	getCell := func(parseProps mirrorHookCellProps) ui.Node {
		for parseIndex := 0; parseIndex < mirrorHooksPerComponent; parseIndex++ {
			runtime.GoUseState(getRuntime, parseIndex+parseProps.GetIndex)
			if useTypedHooks {
				runtime.GoUseEffectOf(mirrorHookStaticEffect, parseIndex)
				runtime.GoUseMemoOf(mirrorHookStaticMemo, parseIndex)
			} else {
				runtime.GoUseEffect(func() func() { return nil })
				runtime.GoUseMemo(func() any { return parseIndex * 2 }, parseIndex)
			}
		}
		return html.Div(
			html.Props{
				Class: "benchmark-hook-node rounded-xl border border-white/10 bg-white/[0.05] px-3 py-2 text-sm text-slate-100",
				Data: map[string]string{
					"hook-index":    strconv.Itoa(parseProps.GetIndex),
					"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
				},
			},
			html.Text("Hooks "+strconv.Itoa(parseProps.GetIndex+1)),
		)
	}

	var storeTypedCell func(mirrorHookCellProps) ui.Node
	if useTyped {
		storeTypedCell = ui.Typed(getCell)
	}
	var storeSetToken func(any)
	getComponent := func() ui.Node {
		getToken, parseSetToken := runtime.GoUseState(getRuntime, 0)
		storeSetToken = parseSetToken
		getItems := make([]ui.Node, 0, mirrorHookComponents)
		for parseIndex := 0; parseIndex < mirrorHookComponents; parseIndex++ {
			getProps := mirrorHookCellProps{
				GetIndex:        parseIndex,
				GetRefreshToken: getToken(),
			}
			if useTyped {
				getItems = append(getItems, storeTypedCell(getProps))
			} else {
				getItems = append(getItems, ui.CreateElement(getCell, getProps))
			}
		}
		return html.Div(
			html.Props{
				ID:    "hooks-container",
				Class: "grid gap-3",
				Data:  map[string]string{"refresh-token": strconv.Itoa(getToken())},
			},
			getItems...,
		)
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		parseB.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()
	if storeSetToken == nil {
		parseB.Fatal("expected token setter captured during mount render")
	}
	return func(parseToken int) { storeSetToken(parseToken) }, getScheduler
}

// BenchmarkMirrorHooksRefresh mirrors the browser hooks-render scenario's
// steady state: every leaf re-renders (token prop changes), re-walking all
// 2400 hook slots.
func BenchmarkMirrorHooksRefresh(parseB *testing.B) {
	getSetToken, getScheduler := newMirrorHooksHost(parseB, false, false)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetToken(parseI + 1)
		getScheduler.FlushAll()
	}
}

// BenchmarkMirrorHooksRefreshTypedHooks combines ui.Typed dispatch with the
// zero-alloc GoUseMemoOf/GoUseEffectOf variants (static compute fns, single
// comparable dep) — the "well-written app" ceiling for the hooks scenario.
func BenchmarkMirrorHooksRefreshTypedHooks(parseB *testing.B) {
	getSetToken, getScheduler := newMirrorHooksHost(parseB, true, true)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetToken(parseI + 1)
		getScheduler.FlushAll()
	}
}

// BenchmarkMirrorHooksRefreshTyped is the same scenario with cells built via
// ui.Typed static dispatch instead of the reflect trampoline.
func BenchmarkMirrorHooksRefreshTyped(parseB *testing.B) {
	getSetToken, getScheduler := newMirrorHooksHost(parseB, true, false)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetToken(parseI + 1)
		getScheduler.FlushAll()
	}
}
