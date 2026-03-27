//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	benchmarkshared "github.com/monstercameron/GoWebComponents/examples/201-render-benchmark/shared"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	benchmarkModeRuntime         = "runtime1"
	benchmarkModeRuntime3        = "runtime2"
	benchmarkModeRuntime3Workers = "runtime2-workers4"

	benchmarkCoreListSize       = 40
	benchmarkContentCardCount   = 12
	benchmarkDeepTreeDepth      = 60
	benchmarkHookComponentCount = 40
	benchmarkHooksPerComponent  = 20
	benchmarkRuntime3ShardCount = 4

	benchmarkRuntime3CoreRendererID    = "examples.render-benchmark.runtime3.core"
	benchmarkRuntime3ContentRendererID = "examples.render-benchmark.runtime3.content"
	benchmarkRuntime3DeepRendererID    = "examples.render-benchmark.runtime3.deep"
	benchmarkRuntime3HookRendererID    = "examples.render-benchmark.runtime3.hook"

	benchmarkRuntime3DeepRegionID = "examples.render-benchmark.runtime3.deep.primary"
)

type renderBenchmarkAppProps struct {
	GetMode string
}

type renderBenchmarkDeepTreeProps struct {
	GetDepth        int
	GetRefreshToken int
}

type renderBenchmarkContentCardProps struct {
	GetItem         benchmarkshared.BenchmarkContentCardData
	GetRefreshToken int
}

type renderBenchmarkRuntime3CoreProps struct {
	GetItems        []benchmarkshared.BenchmarkPreparedCoreItem
	GetWorker       string
	GetWorkDigest   uint64
	GetRefreshToken int
}

type renderBenchmarkRuntime3ContentProps struct {
	GetItems        []benchmarkshared.BenchmarkPreparedContentCard
	GetWorker       string
	GetWorkDigest   uint64
	GetRefreshToken int
}

type renderBenchmarkRuntime3DeepProps struct {
	GetDepth        int
	GetRefreshToken int
}

type renderBenchmarkRuntime3HookProps struct {
	GetLabel        string
	GetRefreshToken int
}

type renderBenchmarkHookCellProps struct {
	GetIndex             int
	GetRefreshToken      int
	GetMode              string
	GetSchedulerShardIDs []string
}

// buildBenchmarkMode resolves the requested benchmark framework mode from the current query string.
func buildBenchmarkMode() string {
	getLocation, parseErr := interop.GetWindowLocation()
	if parseErr != nil {
		return benchmarkModeRuntime
	}
	getQueryValues, parseQueryErr := url.ParseQuery(strings.TrimPrefix(getLocation.Search(), "?"))
	if parseQueryErr != nil {
		return benchmarkModeRuntime
	}
	switch strings.ToLower(strings.TrimSpace(getQueryValues.Get("framework"))) {
	case "", "runtime", benchmarkModeRuntime:
		return benchmarkModeRuntime
	case benchmarkModeRuntime3Workers, "runtime2-4workers", "runtime2workers4":
		return benchmarkModeRuntime3Workers
	case "runtime3", benchmarkModeRuntime3:
		return benchmarkModeRuntime3
	default:
		return benchmarkModeRuntime
	}
}

// buildBenchmarkCoreItems builds the stable list payload used by the render and update scenarios.
func buildBenchmarkCoreItems() []string {
	getItems := make([]string, benchmarkCoreListSize)
	for parseIndex := 0; parseIndex < benchmarkCoreListSize; parseIndex++ {
		getItems[parseIndex] = "Item " + strconv.Itoa(parseIndex)
	}
	return getItems
}

// buildBenchmarkContentItems builds the stable content-card payload used by the render and update scenarios.
func buildBenchmarkContentItems() []benchmarkshared.BenchmarkContentCardData {
	getItems := make([]benchmarkshared.BenchmarkContentCardData, benchmarkContentCardCount)
	for parseIndex := 0; parseIndex < benchmarkContentCardCount; parseIndex++ {
		getItems[parseIndex] = benchmarkshared.BenchmarkContentCardData{
			GetID:      parseIndex,
			GetTitle:   "Article " + strconv.Itoa(parseIndex),
			GetSummary: "This benchmark card exercises browser-side render and update work with nested content blocks.",
			GetStatus:  "draft",
			GetMeta:    "Section " + strconv.Itoa((parseIndex%3)+1),
			GetTags: []string{
				"perf",
				"browser",
				"card-" + strconv.Itoa(parseIndex%4),
			},
		}
	}
	return getItems
}

// buildBenchmarkUpdatedCoreItems applies the update payload used by the core-list update scenario.
func buildBenchmarkUpdatedCoreItems(parseItems []string) []string {
	getItems := make([]string, len(parseItems))
	for parseIndex, getItem := range parseItems {
		getItems[parseIndex] = getItem + " (Updated)"
	}
	return getItems
}

// buildBenchmarkUpdatedContentItems applies the update payload used by the content-card update scenario.
func buildBenchmarkUpdatedContentItems(parseItems []benchmarkshared.BenchmarkContentCardData) []benchmarkshared.BenchmarkContentCardData {
	getItems := make([]benchmarkshared.BenchmarkContentCardData, len(parseItems))
	for parseIndex, getItem := range parseItems {
		getTags := append([]string(nil), getItem.GetTags...)
		getItems[parseIndex] = benchmarkshared.BenchmarkContentCardData{
			GetID:      getItem.GetID,
			GetTitle:   getItem.GetTitle + " (Updated)",
			GetSummary: getItem.GetSummary + " Updated with fresh content.",
			GetStatus:  "live",
			GetMeta:    getItem.GetMeta + " / refreshed",
			GetTags:    getTags,
		}
	}
	return getItems
}

// registerBenchmarkRuntime3Renderers registers the public parallel-region renderers used by the experimental runtime3 benchmark mode.
func registerBenchmarkRuntime3Renderers() {
	if parseErr := ui.RegisterParallelRegion(benchmarkRuntime3CoreRendererID, renderBenchmarkRuntime3CoreRegion); parseErr != nil {
		panic(parseErr)
	}
	if parseErr := ui.RegisterParallelRegion(benchmarkRuntime3ContentRendererID, renderBenchmarkRuntime3ContentRegion); parseErr != nil {
		panic(parseErr)
	}
	if parseErr := ui.RegisterParallelRegion(benchmarkRuntime3DeepRendererID, renderBenchmarkRuntime3DeepRegion); parseErr != nil {
		panic(parseErr)
	}
	if parseErr := ui.RegisterParallelRegion(benchmarkRuntime3HookRendererID, renderBenchmarkRuntime3HookRegion); parseErr != nil {
		panic(parseErr)
	}
}

// buildBenchmarkRuntime3ModeKey normalizes one runtime2 benchmark mode into a stable region-ID suffix.
func buildBenchmarkRuntime3ModeKey(parseMode string) string {
	return strings.ReplaceAll(parseMode, "-", ".")
}

// buildBenchmarkRuntime3SchedulerShardIDs resolves the scheduler shard list for one runtime2 benchmark mode.
func buildBenchmarkRuntime3SchedulerShardIDs(parseMode string) []string {
	if parseMode == benchmarkModeRuntime3Workers {
		return []string{
			"runtime2-shard-a",
			"runtime2-shard-b",
			"runtime2-shard-c",
			"runtime2-shard-d",
		}
	}
	return []string{"runtime2-shard-a"}
}

// buildBenchmarkRuntime3ChunkBounds partitions one list length into stable region-sized chunk bounds.
func buildBenchmarkRuntime3ChunkBounds(parseItemCount int) [][2]int {
	if parseItemCount <= 0 {
		return nil
	}
	getChunkSize := (parseItemCount + benchmarkRuntime3ShardCount - 1) / benchmarkRuntime3ShardCount
	getBounds := make([][2]int, 0, benchmarkRuntime3ShardCount)
	for parseChunkIndex := 0; parseChunkIndex < benchmarkRuntime3ShardCount; parseChunkIndex++ {
		getStart := parseChunkIndex * getChunkSize
		if getStart >= parseItemCount {
			break
		}
		getEnd := getStart + getChunkSize
		if getEnd > parseItemCount {
			getEnd = parseItemCount
		}
		getBounds = append(getBounds, [2]int{getStart, getEnd})
	}
	return getBounds
}

// buildBenchmarkRuntime3CoreRegionNodes renders one worker-prepared core-list region set for the runtime2 benchmark modes.
func buildBenchmarkRuntime3CoreRegionNodes(parseMode string, parseCoreChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult, parseRefreshToken int) []ui.Node {
	getSchedulerShardIDs := buildBenchmarkRuntime3SchedulerShardIDs(parseMode)
	getModeKey := buildBenchmarkRuntime3ModeKey(parseMode)
	getRegionNodes := make([]ui.Node, 0, len(parseCoreChunks))
	for parseChunkIndex, getChunk := range parseCoreChunks {
		getRegionNodes = append(getRegionNodes, ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3CoreProps]{
			RendererID:       benchmarkRuntime3CoreRendererID,
			RegionInstanceID: fmt.Sprintf("examples.render-benchmark.%s.core.%02d", getModeKey, parseChunkIndex),
			Props: renderBenchmarkRuntime3CoreProps{
				GetItems:        getChunk.GetItems,
				GetWorker:       getChunk.GetWorker,
				GetWorkDigest:   getChunk.GetWorkDigest,
				GetRefreshToken: parseRefreshToken,
			},
			SchedulerShardIDs: getSchedulerShardIDs,
		}))
	}
	return getRegionNodes
}

// buildBenchmarkRuntime3ContentRegionNodes renders one worker-prepared content-card region set for the runtime2 benchmark modes.
func buildBenchmarkRuntime3ContentRegionNodes(parseMode string, parseContentChunks []benchmarkshared.BenchmarkWorkerContentChunkResult, parseRefreshToken int) []ui.Node {
	getSchedulerShardIDs := buildBenchmarkRuntime3SchedulerShardIDs(parseMode)
	getModeKey := buildBenchmarkRuntime3ModeKey(parseMode)
	getRegionNodes := make([]ui.Node, 0, len(parseContentChunks))
	for parseChunkIndex, getChunk := range parseContentChunks {
		getRegionNodes = append(getRegionNodes, ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3ContentProps]{
			RendererID:       benchmarkRuntime3ContentRendererID,
			RegionInstanceID: fmt.Sprintf("examples.render-benchmark.%s.content.%02d", getModeKey, parseChunkIndex),
			Props: renderBenchmarkRuntime3ContentProps{
				GetItems:        getChunk.GetItems,
				GetWorker:       getChunk.GetWorker,
				GetWorkDigest:   getChunk.GetWorkDigest,
				GetRefreshToken: parseRefreshToken,
			},
			SchedulerShardIDs: getSchedulerShardIDs,
		}))
	}
	return getRegionNodes
}

// renderBenchmarkDeepTree renders the recursive deep-tree scenario for the current runtime mode.
func renderBenchmarkDeepTree(parseProps renderBenchmarkDeepTreeProps) ui.Node {
	if parseProps.GetDepth <= 0 {
		return html.Div(
			html.Props{
				ID:    "benchmark-deep-leaf",
				Class: "benchmark-deep-leaf rounded-xl border border-cyan-400/20 bg-cyan-400/10 px-3 py-2 text-xs text-cyan-100",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseProps.GetRefreshToken)},
			},
			html.Text("Leaf"),
		)
	}
	return html.Div(
		html.Props{Class: "benchmark-deep-node border-l border-white/10 pl-2"},
		ui.CreateElement(renderBenchmarkDeepTree, renderBenchmarkDeepTreeProps{
			GetDepth:        parseProps.GetDepth - 1,
			GetRefreshToken: parseProps.GetRefreshToken,
		}),
	)
}

// renderBenchmarkContentCard renders one nested content-card subtree for the current runtime mode.
func renderBenchmarkContentCard(parseProps renderBenchmarkContentCardProps) ui.Node {
	getTagNodes := make([]ui.Node, 0, len(parseProps.GetItem.GetTags))
	for _, getTag := range parseProps.GetItem.GetTags {
		getTagNodes = append(getTagNodes, html.Span(
			html.Props{Class: "benchmark-content-tag rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300"},
			html.Text(getTag),
		))
	}
	return html.Article(
		html.Props{
			Class: "benchmark-content-card rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20",
			Data: map[string]string{
				"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
				"card-id":       strconv.Itoa(parseProps.GetItem.GetID),
			},
		},
		html.Div(
			html.Props{Class: "flex items-center justify-between gap-3"},
			html.H2(
				html.Props{Class: "benchmark-content-title text-base font-semibold text-white"},
				html.Text(parseProps.GetItem.GetTitle),
			),
			html.Span(
				html.Props{Class: "benchmark-content-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1 text-[11px] uppercase tracking-[0.16em] text-cyan-100"},
				html.Text(parseProps.GetItem.GetStatus),
			),
		),
		html.P(
			html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"},
			html.Text(parseProps.GetItem.GetSummary),
		),
		html.P(
			html.Props{Class: "benchmark-content-meta mt-3 text-xs uppercase tracking-[0.16em] text-slate-400"},
			html.Text(parseProps.GetItem.GetMeta),
		),
		html.Div(
			html.Props{Class: "mt-4 flex flex-wrap gap-2"},
			getTagNodes...,
		),
	)
}

// renderBenchmarkManyHooks renders one hook-heavy leaf used by the current runtime benchmark mode.
func renderBenchmarkManyHooks(parseProps renderBenchmarkHookCellProps) ui.Node {
	for parseIndex := 0; parseIndex < benchmarkHooksPerComponent; parseIndex++ {
		ui.UseState(parseIndex + parseProps.GetIndex)
		ui.UseEffect(func() func() { return nil })
		ui.UseMemo(func() int { return parseIndex * 2 }, parseIndex)
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

// renderBenchmarkRuntime3CoreRegion renders the public parallel-region shell used for the experimental runtime3 core-list scenario.
func renderBenchmarkRuntime3CoreRegion(parseProps renderBenchmarkRuntime3CoreProps) ui.Node {
	getItems := make([]ui.Node, 0, len(parseProps.GetItems))
	for _, getItem := range parseProps.GetItems {
		getItems = append(getItems, html.Div(
			html.Props{
				Class: "benchmark-core-item rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-slate-100",
				Data: map[string]string{
					"prep-digest": strconv.FormatUint(getItem.GetDigest, 10),
				},
			},
			html.Text(getItem.GetText),
		))
	}
	return html.Div(
		html.Props{
			Class: "benchmark-core-region grid gap-2",
			Data: map[string]string{
				"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
				"prep-worker":   parseProps.GetWorker,
				"prep-digest":   strconv.FormatUint(parseProps.GetWorkDigest, 10),
			},
		},
		getItems...,
	)
}

// renderBenchmarkRuntime3ContentRegion renders the public parallel-region shell used for the experimental runtime3 content-card scenario.
func renderBenchmarkRuntime3ContentRegion(parseProps renderBenchmarkRuntime3ContentProps) ui.Node {
	getItems := make([]ui.Node, 0, len(parseProps.GetItems))
	for _, getItem := range parseProps.GetItems {
		getItems = append(getItems, renderBenchmarkContentCard(renderBenchmarkContentCardProps{
			GetItem: benchmarkshared.BenchmarkContentCardData{
				GetID:      getItem.GetID,
				GetTitle:   getItem.GetTitle,
				GetSummary: getItem.GetSummary,
				GetStatus:  getItem.GetStatus,
				GetMeta:    getItem.GetMeta,
				GetTags:    getItem.GetTags,
			},
			GetRefreshToken: parseProps.GetRefreshToken,
		}))
	}
	return html.Div(
		html.Props{
			Class: "benchmark-content-region grid gap-4 lg:grid-cols-2",
			Data: map[string]string{
				"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
				"prep-worker":   parseProps.GetWorker,
				"prep-digest":   strconv.FormatUint(parseProps.GetWorkDigest, 10),
			},
		},
		getItems...,
	)
}

// renderBenchmarkRuntime3DeepRegion renders the public parallel-region shell used for the experimental runtime3 deep-tree scenario.
func renderBenchmarkRuntime3DeepRegion(parseProps renderBenchmarkRuntime3DeepProps) ui.Node {
	return renderBenchmarkDeepTree(renderBenchmarkDeepTreeProps{
		GetDepth:        parseProps.GetDepth,
		GetRefreshToken: parseProps.GetRefreshToken,
	})
}

// renderBenchmarkRuntime3HookRegion renders the public parallel-region shell used by one experimental runtime3 hook cell.
func renderBenchmarkRuntime3HookRegion(parseProps renderBenchmarkRuntime3HookProps) ui.Node {
	return html.Div(
		html.Props{
			Class: "benchmark-hook-node rounded-xl border border-cyan-400/20 bg-cyan-400/10 px-3 py-2 text-sm text-cyan-50",
			Data: map[string]string{
				"label":         parseProps.GetLabel,
				"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
			},
		},
		html.Text(parseProps.GetLabel),
	)
}

// renderBenchmarkRuntime3HookCell renders one hook-owning wrapper that still emits a public parallel-region shell in runtime3 mode.
func renderBenchmarkRuntime3HookCell(parseProps renderBenchmarkHookCellProps) ui.Node {
	for parseIndex := 0; parseIndex < benchmarkHooksPerComponent; parseIndex++ {
		ui.UseState(parseIndex + parseProps.GetIndex)
		ui.UseEffect(func() func() { return nil })
		ui.UseMemo(func() int { return parseIndex * 3 }, parseIndex)
	}
	getRegionID := fmt.Sprintf(
		"examples.render-benchmark.%s.hook.%02d",
		buildBenchmarkRuntime3ModeKey(parseProps.GetMode),
		parseProps.GetIndex,
	)
	return ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3HookProps]{
		RendererID:       benchmarkRuntime3HookRendererID,
		RegionInstanceID: getRegionID,
		Props: renderBenchmarkRuntime3HookProps{
			GetLabel:        "Hooks " + strconv.Itoa(parseProps.GetIndex+1),
			GetRefreshToken: parseProps.GetRefreshToken,
		},
		SchedulerShardIDs: parseProps.GetSchedulerShardIDs,
	})
}

// buildBenchmarkRuntimeNode renders the active benchmark scenario through the current runtime path.
func buildBenchmarkRuntimeNode(parseView string, parseCoreItems []string, parseContentItems []benchmarkshared.BenchmarkContentCardData, parseTreeDepth int, parseHookCount int, parseRefreshToken int) ui.Node {
	switch parseView {
	case "content":
		getItems := make([]ui.Node, 0, len(parseContentItems))
		for _, getItem := range parseContentItems {
			getItems = append(getItems, ui.CreateElement(renderBenchmarkContentCard, renderBenchmarkContentCardProps{
				GetItem:         getItem,
				GetRefreshToken: parseRefreshToken,
			}))
		}
		return html.Div(
			html.Props{
				ID:    "content-container",
				Class: "grid gap-4 lg:grid-cols-2",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			getItems...,
		)
	case "deep":
		return ui.CreateElement(renderBenchmarkDeepTree, renderBenchmarkDeepTreeProps{
			GetDepth:        parseTreeDepth,
			GetRefreshToken: parseRefreshToken,
		})
	case "hooks":
		getItems := make([]ui.Node, 0, parseHookCount)
		for parseIndex := 0; parseIndex < parseHookCount; parseIndex++ {
			getItems = append(getItems, ui.CreateElement(renderBenchmarkManyHooks, renderBenchmarkHookCellProps{
				GetIndex:        parseIndex,
				GetRefreshToken: parseRefreshToken,
			}))
		}
		return html.Div(
			html.Props{
				ID:    "hooks-container",
				Class: "grid gap-3 sm:grid-cols-2 lg:grid-cols-4",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			getItems...,
		)
	default:
		getItems := make([]ui.Node, 0, len(parseCoreItems))
		for _, getItem := range parseCoreItems {
			getItems = append(getItems, html.Div(
				html.Props{Class: "benchmark-core-item rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-slate-100"},
				html.Text(getItem),
			))
		}
		return html.Div(
			html.Props{
				ID:    "core-list-container",
				Class: "grid gap-2",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			getItems...,
		)
	}
}

// buildBenchmarkRuntime3Node renders the active benchmark scenario through the runtime2 parallel-region shell path.
func buildBenchmarkRuntime3Node(parseMode string, parseView string, parseCoreChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult, parseContentChunks []benchmarkshared.BenchmarkWorkerContentChunkResult, parseTreeDepth int, parseHookCount int, parseRefreshToken int) ui.Node {
	getSchedulerShardIDs := buildBenchmarkRuntime3SchedulerShardIDs(parseMode)
	switch parseView {
	case "content":
		return html.Div(
			html.Props{
				ID:    "content-container",
				Class: "grid gap-4",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			buildBenchmarkRuntime3ContentRegionNodes(parseMode, parseContentChunks, parseRefreshToken)...,
		)
	case "deep":
		return ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3DeepProps]{
			RendererID:       benchmarkRuntime3DeepRendererID,
			RegionInstanceID: benchmarkRuntime3DeepRegionID,
			Props: renderBenchmarkRuntime3DeepProps{
				GetDepth:        parseTreeDepth,
				GetRefreshToken: parseRefreshToken,
			},
			SchedulerShardIDs: getSchedulerShardIDs,
		})
	case "hooks":
		getItems := make([]ui.Node, 0, parseHookCount)
		for parseIndex := 0; parseIndex < parseHookCount; parseIndex++ {
			getItems = append(getItems, ui.CreateElement(renderBenchmarkRuntime3HookCell, renderBenchmarkHookCellProps{
				GetIndex:             parseIndex,
				GetRefreshToken:      parseRefreshToken,
				GetMode:              parseMode,
				GetSchedulerShardIDs: getSchedulerShardIDs,
			}))
		}
		return html.Div(
			html.Props{
				ID:    "hooks-container",
				Class: "grid gap-3 sm:grid-cols-2 lg:grid-cols-4",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			getItems...,
		)
	default:
		return html.Div(
			html.Props{
				ID:    "core-list-container",
				Class: "grid gap-2",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			buildBenchmarkRuntime3CoreRegionNodes(parseMode, parseCoreChunks, parseRefreshToken)...,
		)
	}
}

// renderBenchmarkApp renders the benchmark control surface and active browser scenario for the requested framework mode.
func renderBenchmarkApp(parseProps renderBenchmarkAppProps) ui.Node {
	getCoreItems := ui.UseState([]string{})
	getContentItems := ui.UseState([]benchmarkshared.BenchmarkContentCardData{})
	getCoreChunks := ui.UseState([]benchmarkshared.BenchmarkWorkerCoreChunkResult{})
	getContentChunks := ui.UseState([]benchmarkshared.BenchmarkWorkerContentChunkResult{})
	getView := ui.UseState("core")
	getTreeDepth := ui.UseState(0)
	getHookCount := ui.UseState(0)
	getRefreshToken := ui.UseState(0)
	getLastAction := ui.UseState("idle")
	getPoolRef := ui.UseRef[*interop.WorkerPool](nil)
	getPoolRevision := ui.UseState(0)
	getPrepareRevision := ui.UseState(0)
	getWorkerState := ui.UseState(buildBenchmarkWorkerState{
		GetWorkerCount: buildBenchmarkWorkerCount(parseProps.GetMode),
	})
	clearBenchmarkPreparedChunks := func() {
		getCoreChunks.Set(nil)
		getContentChunks.Set(nil)
	}
	handleBenchmarkPrepareBump := func() {
		getPrepareRevision.Update(func(parseValue int) int {
			return parseValue + 1
		})
	}
	handleBenchmarkWorkerPoolEffect(parseProps.GetMode, getPoolRef, getPoolRevision, getWorkerState)
	handleBenchmarkWorkerPrepareEffect(
		parseProps.GetMode,
		getView.Get(),
		getCoreItems.Get(),
		getContentItems.Get(),
		getPrepareRevision.Get(),
		getPoolRevision.Get(),
		getPoolRef,
		getCoreChunks,
		getContentChunks,
		getWorkerState,
	)

	handleCoreRender := ui.UseEvent(func() {
		getView.Set("core")
		getCoreItems.Set(buildBenchmarkCoreItems())
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-render")
	})
	handleCoreUpdate := ui.UseEvent(func() {
		getView.Set("core")
		getCoreItems.Update(func(parseItems []string) []string {
			return buildBenchmarkUpdatedCoreItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-update")
	})
	handleCoreClear := ui.UseEvent(func() {
		getView.Set("core")
		getCoreItems.Set([]string{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-clear")
	})
	handleContentRender := ui.UseEvent(func() {
		getView.Set("content")
		getContentItems.Set(buildBenchmarkContentItems())
		getCoreItems.Set([]string{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-render")
	})
	handleContentUpdate := ui.UseEvent(func() {
		getView.Set("content")
		getContentItems.Update(func(parseItems []benchmarkshared.BenchmarkContentCardData) []benchmarkshared.BenchmarkContentCardData {
			return buildBenchmarkUpdatedContentItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-update")
	})
	handleContentClear := ui.UseEvent(func() {
		getView.Set("content")
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getCoreItems.Set([]string{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-clear")
	})
	handleDeepRender := ui.UseEvent(func() {
		getView.Set("deep")
		getCoreItems.Set([]string{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(benchmarkDeepTreeDepth)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("deep-render")
	})
	handleHooksRender := ui.UseEvent(func() {
		getView.Set("hooks")
		getCoreItems.Set([]string{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getHookCount.Set(benchmarkHookComponentCount)
		handleBenchmarkPrepareBump()
		getLastAction.Set("hooks-render")
	})
	handleRefresh := ui.UseEvent(func() {
		getRefreshToken.Update(func(parseValue int) int {
			return parseValue + 1
		})
		getLastAction.Set("refresh")
	})

	var getBenchmarkNode ui.Node
	switch parseProps.GetMode {
	case benchmarkModeRuntime3, benchmarkModeRuntime3Workers:
		getBenchmarkNode = buildBenchmarkRuntime3Node(
			parseProps.GetMode,
			getView.Get(),
			getCoreChunks.Get(),
			getContentChunks.Get(),
			getTreeDepth.Get(),
			getHookCount.Get(),
			getRefreshToken.Get(),
		)
	default:
		getBenchmarkNode = buildBenchmarkRuntimeNode(
			getView.Get(),
			getCoreItems.Get(),
			getContentItems.Get(),
			getTreeDepth.Get(),
			getHookCount.Get(),
			getRefreshToken.Get(),
		)
	}

	getMetricNodes := []ui.Node{
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("Framework")), html.P(html.Props{ID: "metric-framework", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(parseProps.GetMode))),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("View")), html.P(html.Props{ID: "metric-view", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(getView.Get()))),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("Core Count")), html.P(html.Props{ID: "metric-core-count", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(len(getCoreItems.Get()))))),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("Content Count")), html.P(html.Props{ID: "metric-content-count", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(len(getContentItems.Get()))))),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("Refresh Count")), html.P(html.Props{ID: "metric-refresh-count", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(getRefreshToken.Get())))),
		html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/35 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text("Last Action")), html.P(html.Props{ID: "metric-last-action", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(getLastAction.Get()))),
	}
	if hasBenchmarkWorkerMode(parseProps.GetMode) {
		getWorkerSnapshot := getWorkerState.Get()
		getWorkerStatusText := "idle"
		if getWorkerSnapshot.IsBooting {
			getWorkerStatusText = "booting"
		} else if getWorkerSnapshot.IsPreparing {
			getWorkerStatusText = "preparing"
		} else if getWorkerSnapshot.IsReady {
			getWorkerStatusText = "ready"
		}
		if strings.TrimSpace(getWorkerSnapshot.GetErrorText) != "" {
			getWorkerStatusText = "error"
		}
		getMetricNodes = append(getMetricNodes,
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Worker Status")), html.P(html.Props{ID: "metric-worker-status", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(getWorkerStatusText))),
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Worker Count")), html.P(html.Props{ID: "metric-worker-count", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(getWorkerSnapshot.GetWorkerCount)))),
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Prepared Items")), html.P(html.Props{ID: "metric-worker-items", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(getWorkerSnapshot.GetPreparedItems)))),
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Last Batch")), html.P(html.Props{ID: "metric-worker-batch-ms", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.FormatInt(getWorkerSnapshot.GetLastBatchMS, 10)+" ms"))),
		)
	}

	return html.Div(
		html.Props{
			ID:    "benchmark-app",
			Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.12),transparent_24%),radial-gradient(circle_at_top_right,rgba(59,130,246,0.10),transparent_20%),linear-gradient(180deg,#020617_0%,#07111f_42%,#0f172a_100%)] px-4 py-8 text-white",
			Data: map[string]string{
				"framework": parseProps.GetMode,
			},
		},
		html.Div(
			html.Props{Class: "mx-auto max-w-7xl"},
			html.Div(
				html.Props{Class: "rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"},
				html.P(
					html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"},
					html.Text("Example 201"),
				),
				html.H1(
					html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white"},
					html.Text("Browser Render Benchmark Subject"),
				),
				html.P(
					html.Props{Class: "mt-4 max-w-4xl text-sm leading-7 text-slate-300"},
					html.Text("This page exposes one benchmark subject with a stable DOM contract. The same browser-side Playwright harness drives React 18, runtime1, runtime2 with one Go WASM worker for chunk preparation, and runtime2 with four Go WASM workers for the same off-main-thread preparation step."),
				),
				html.Div(
					html.Props{ID: "benchmark-controls", Class: "mt-8 grid gap-3 md:grid-cols-3 xl:grid-cols-5"},
					html.Button(html.Props{ID: "btn-core-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleCoreRender}, html.Text("Render Core Items")),
					html.Button(html.Props{ID: "btn-core-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreUpdate}, html.Text("Update Core Items")),
					html.Button(html.Props{ID: "btn-core-clear", Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: handleCoreClear}, html.Text("Clear Core Items")),
					html.Button(html.Props{ID: "btn-content-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleContentRender}, html.Text("Render Content Cards")),
					html.Button(html.Props{ID: "btn-content-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleContentUpdate}, html.Text("Update Content Cards")),
					html.Button(html.Props{ID: "btn-content-clear", Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: handleContentClear}, html.Text("Clear Content Cards")),
					html.Button(html.Props{ID: "btn-deep-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleDeepRender}, html.Text("Render Deep Tree")),
					html.Button(html.Props{ID: "btn-hooks-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleHooksRender}, html.Text("Render Hook Grid")),
					html.Button(html.Props{ID: "btn-refresh", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleRefresh}, html.Text("Refresh Current View")),
				),
				html.Div(
					html.Props{ID: "benchmark-metrics", Class: "mt-8 grid gap-3 md:grid-cols-2 xl:grid-cols-5 2xl:grid-cols-10"},
					getMetricNodes...,
				),
			),
			html.Div(
				html.Props{ID: "benchmark-container", Class: "mt-6 rounded-[28px] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30"},
				getBenchmarkNode,
			),
		),
	)
}

// main mounts the benchmark subject in the requested framework mode.
func main() {
	utils.DisableAllDebug()
	getMode := buildBenchmarkMode()
	if getMode == benchmarkModeRuntime3 || getMode == benchmarkModeRuntime3Workers {
		registerBenchmarkRuntime3Renderers()
	}
	ui.Render(ui.CreateElement(renderBenchmarkApp, renderBenchmarkAppProps{GetMode: getMode}), "#app")
	utils.WaitForever()
}
