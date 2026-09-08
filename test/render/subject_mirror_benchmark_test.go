//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// These benchmarks mirror the Example 201 browser benchmark's core-list
// scenarios through the public html.Props construction path and the full
// native reconciler (mockdom adapter), so browser-side render/diff costs can
// be profiled with pprof. Keep the row shape in sync with
// examples/testing/render-benchmark (key + long class + one data attribute).

type mirrorCoreRow struct {
	GetID   int
	GetText string
}

const mirrorCoreRowClass = "benchmark-core-item rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-slate-100"

func buildMirrorCoreRows(parseCount int, parseSuffix string) []mirrorCoreRow {
	getRows := make([]mirrorCoreRow, parseCount)
	for parseIndex := range getRows {
		getRows[parseIndex] = mirrorCoreRow{GetID: parseIndex, GetText: "Item " + strconv.Itoa(parseIndex) + parseSuffix}
	}
	return getRows
}

func renderMirrorCoreList(parseRows []mirrorCoreRow, parseToken int) ui.Node {
	getItems := make([]ui.Node, 0, len(parseRows))
	for _, getRow := range parseRows {
		getItems = append(getItems, html.Div(
			html.Props{
				Key:      strconv.Itoa(getRow.GetID),
				Class:    mirrorCoreRowClass,
				DataAttr: html.DataAttribute{Name: "row-id", Value: strconv.Itoa(getRow.GetID)},
				Text:     getRow.GetText,
			},
		))
	}
	return html.Div(
		html.Props{
			ID:    "core-list-container",
			Class: "grid gap-2",
			Data:  map[string]string{"refresh-token": strconv.Itoa(parseToken)},
		},
		getItems...,
	)
}

// newMirrorBenchmarkHost mounts the mirror component and returns the setters
// that drive scenario updates plus the scheduler to flush scheduled work.
func newMirrorBenchmarkHost(parseB *testing.B) (func([]mirrorCoreRow), func(int), *mockdom.MockScheduler) {
	parseB.Helper()
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div")

	// ui.UseState is a stub in native builds; the runtime-layer hook drives
	// real scheduling through the mock scheduler.
	var storeSetRows func(any)
	var storeSetToken func(any)
	getComponent := func() ui.Node {
		getRows, parseSetRows := runtime.GoUseState(getRuntime, []mirrorCoreRow{})
		getToken, parseSetToken := runtime.GoUseState(getRuntime, 0)
		storeSetRows = parseSetRows
		storeSetToken = parseSetToken
		return renderMirrorCoreList(getRows(), getToken())
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		parseB.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()
	if storeSetRows == nil || storeSetToken == nil {
		parseB.Fatal("expected state setters captured during mount render")
	}
	getSetRows := func(parseRows []mirrorCoreRow) { storeSetRows(parseRows) }
	getSetToken := func(parseToken int) { storeSetToken(parseToken) }
	return getSetRows, getSetToken, getScheduler
}

// BenchmarkMirrorCoreRender40 mirrors the browser core-render scenario: mount
// 40 keyed rows from an empty list, then clear outside the timed window.
func BenchmarkMirrorCoreRender40(parseB *testing.B) {
	getSetRows, _, getScheduler := newMirrorBenchmarkHost(parseB)
	getRows := buildMirrorCoreRows(40, "")
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetRows(getRows)
		getScheduler.FlushAll()
		parseB.StopTimer()
		getSetRows(nil)
		getScheduler.FlushAll()
		parseB.StartTimer()
	}
}

// BenchmarkMirrorCoreUpdate40 mirrors the browser core-update scenario: swap
// every row's text on a mounted 40-row list.
func BenchmarkMirrorCoreUpdate40(parseB *testing.B) {
	getSetRows, _, getScheduler := newMirrorBenchmarkHost(parseB)
	getRowsA := buildMirrorCoreRows(40, "")
	getRowsB := buildMirrorCoreRows(40, " (Updated)")
	getSetRows(getRowsA)
	getScheduler.FlushAll()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if parseI%2 == 0 {
			getSetRows(getRowsB)
		} else {
			getSetRows(getRowsA)
		}
		getScheduler.FlushAll()
	}
}

// BenchmarkMirrorCoreAppend mirrors the browser core-append scenario: append
// 100 keyed rows to a mounted 200-row list, then trim back outside the timer.
func BenchmarkMirrorCoreAppend(parseB *testing.B) {
	getSetRows, _, getScheduler := newMirrorBenchmarkHost(parseB)
	getBase := buildMirrorCoreRows(200, "")
	getGrown := buildMirrorCoreRows(300, "")
	getSetRows(getBase)
	getScheduler.FlushAll()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetRows(getGrown)
		getScheduler.FlushAll()
		parseB.StopTimer()
		getSetRows(getBase)
		getScheduler.FlushAll()
		parseB.StartTimer()
	}
}

// BenchmarkMirrorCoreRefresh40 mirrors the browser core-refresh scenario: only
// the container's refresh token changes; all 40 rows should bail out.
func BenchmarkMirrorCoreRefresh40(parseB *testing.B) {
	getSetRows, getSetToken, getScheduler := newMirrorBenchmarkHost(parseB)
	getSetRows(buildMirrorCoreRows(40, ""))
	getScheduler.FlushAll()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		getSetToken(parseI + 1)
		getScheduler.FlushAll()
	}
}
