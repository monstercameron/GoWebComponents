//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	benchmarkshared "github.com/monstercameron/GoWebComponents/examples/201-render-benchmark/shared"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/html"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	benchmarkModeRuntime          = "runtime1"
	benchmarkModeRuntime3         = "runtime2"
	benchmarkModeRuntime3Workers  = "runtime2-workers"
	benchmarkModeRuntime3Workers4 = "runtime2-workers4"

	benchmarkCoreListSize              = 40
	benchmarkCoreStressListSize        = 240
	benchmarkCoreStressHalfSize        = benchmarkCoreStressListSize / 2
	benchmarkCoreAppendCount           = 100
	benchmarkContentCardCount          = 12
	benchmarkDeepTreeDepth             = 60
	benchmarkHookComponentCount        = 40
	benchmarkHooksPerComponent         = 20
	benchmarkPrimitiveRowCount         = 200
	benchmarkPrimitiveRemoveCount      = 100
	benchmarkEnterpriseSectionCount    = 6
	benchmarkEnterpriseRecordCount     = 5
	benchmarkEnterpriseMetricCount     = 4
	benchmarkEnterpriseTargetSectionID = "section-3"
	benchmarkRuntime3ShardCount        = 4

	benchmarkRuntime3CoreRendererID    = "examples.render-benchmark.runtime3.core"
	benchmarkRuntime3ContentRendererID = "examples.render-benchmark.runtime3.content"
	benchmarkRuntime3DeepRendererID    = "examples.render-benchmark.runtime3.deep"
	benchmarkRuntime3HookRendererID    = "examples.render-benchmark.runtime3.hook"

	benchmarkRuntime3DeepRegionID = "examples.render-benchmark.runtime3.deep.primary"
)

// buildBenchmarkQueryValues parses the current benchmark query string.
func buildBenchmarkQueryValues() url.Values {
	getLocation, parseErr := interop.GetWindowLocation()
	if parseErr != nil {
		return url.Values{}
	}
	getQueryValues, parseQueryErr := url.ParseQuery(strings.TrimPrefix(getLocation.Search(), "?"))
	if parseQueryErr != nil {
		return url.Values{}
	}
	return getQueryValues
}

type renderBenchmarkAppProps struct {
	GetMode string
}

type renderBenchmarkDeepTreeProps struct {
	GetDepth        int
	GetRefreshToken int
	GetTreeVersion  int
}

type renderBenchmarkContentCardProps struct {
	GetItem         benchmarkshared.BenchmarkContentCardData
	GetRefreshToken int
}

type renderBenchmarkRuntime3CoreProps struct {
	GetItems      []benchmarkshared.BenchmarkPreparedCoreItem
	GetWorker     string
	GetWorkDigest uint64
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
	GetTreeVersion  int
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

type benchmarkEnterpriseMetricData struct {
	GetLabel string
	GetValue string
}

type benchmarkEnterpriseRecordData struct {
	GetID       string
	GetTitle    string
	GetOwner    string
	GetStatus   string
	GetRevision int
	GetMetrics  []benchmarkEnterpriseMetricData
}

type benchmarkEnterpriseSectionData struct {
	GetID       string
	GetTitle    string
	GetSummary  string
	GetStatus   string
	GetRevision int
	GetRecords  []benchmarkEnterpriseRecordData
}

type benchmarkPrimitiveRowData struct {
	GetID    int
	GetLabel string
	GetState string
	IsActive bool
}

// buildBenchmarkMode resolves the requested benchmark framework mode from the current query string.
func buildBenchmarkMode() string {
	getFramework := strings.ToLower(strings.TrimSpace(buildBenchmarkQueryValues().Get("framework")))
	switch getFramework {
	case "", "runtime", benchmarkModeRuntime:
		return benchmarkModeRuntime
	case benchmarkModeRuntime3Workers4, benchmarkModeRuntime3Workers, "runtime2-4workers", "runtime2workers4":
		return benchmarkModeRuntime3Workers4
	case "runtime3", benchmarkModeRuntime3:
		return benchmarkModeRuntime3
	default:
		if strings.HasPrefix(getFramework, benchmarkModeRuntime3Workers) {
			return getFramework
		}
		return benchmarkModeRuntime
	}
}

// buildBenchmarkCoreItems builds the stable list payload used by the render and update scenarios.
func buildBenchmarkCoreItems() []benchmarkshared.BenchmarkCoreRowData {
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, benchmarkCoreListSize)
	for parseIndex := 0; parseIndex < benchmarkCoreListSize; parseIndex++ {
		getItems[parseIndex] = benchmarkshared.BenchmarkCoreRowData{
			GetID:   parseIndex + 1,
			GetText: "Item " + strconv.Itoa(parseIndex),
		}
	}
	return getItems
}

// buildBenchmarkCoreStressItems builds the heavier ascending list payload used by the list-churn scenarios.
func buildBenchmarkCoreStressItems() []benchmarkshared.BenchmarkCoreRowData {
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, 0, benchmarkCoreStressListSize)
	for parseIndex := 0; parseIndex < benchmarkCoreStressListSize; parseIndex++ {
		getItems = append(getItems, benchmarkshared.BenchmarkCoreRowData{
			GetID:   parseIndex + 1,
			GetText: buildBenchmarkCoreStressText(parseIndex + 1),
		})
	}
	return getItems
}

// buildBenchmarkCoreStressText builds one heavier core-row label used by the list-churn scenarios.
func buildBenchmarkCoreStressText(parseRowID int) string {
	return fmt.Sprintf(
		"Row %04d / lane %d / cluster %d / bucket %d / payload %d",
		parseRowID,
		(parseRowID%11)+1,
		(parseRowID%7)+1,
		(parseRowID%5)+1,
		(parseRowID%13)+1,
	)
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
func buildBenchmarkUpdatedCoreItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, len(parseItems))
	for parseIndex, getItem := range parseItems {
		getItems[parseIndex] = benchmarkshared.BenchmarkCoreRowData{
			GetID:   getItem.GetID,
			GetText: getItem.GetText + " (Updated)",
		}
	}
	return getItems
}

// buildBenchmarkCoreAppendedItems appends one fixed block of heavier rows after the current core-list payload.
func buildBenchmarkCoreAppendedItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := append([]benchmarkshared.BenchmarkCoreRowData(nil), parseItems...)
	for parseOffset := 0; parseOffset < benchmarkCoreAppendCount; parseOffset++ {
		getRowID := 1001 + parseOffset
		getItems = append(getItems, benchmarkshared.BenchmarkCoreRowData{
			GetID:   getRowID,
			GetText: buildBenchmarkCoreStressText(getRowID),
		})
	}
	return getItems
}

// buildBenchmarkCorePrependedItems prepends one fixed block of heavier rows ahead of the current core-list payload.
func buildBenchmarkCorePrependedItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, 0, len(parseItems)+benchmarkCoreAppendCount)
	for parseOffset := 0; parseOffset < benchmarkCoreAppendCount; parseOffset++ {
		getRowID := 2001 + parseOffset
		getItems = append(getItems, benchmarkshared.BenchmarkCoreRowData{
			GetID:   getRowID,
			GetText: buildBenchmarkCoreStressText(getRowID),
		})
	}
	return append(getItems, parseItems...)
}

// buildBenchmarkCoreReversedItems reverses the current core-list order while preserving row identity.
func buildBenchmarkCoreReversedItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := append([]benchmarkshared.BenchmarkCoreRowData(nil), parseItems...)
	for getLeftIndex, getRightIndex := 0, len(getItems)-1; getLeftIndex < getRightIndex; getLeftIndex, getRightIndex = getLeftIndex+1, getRightIndex-1 {
		getItems[getLeftIndex], getItems[getRightIndex] = getItems[getRightIndex], getItems[getLeftIndex]
	}
	return getItems
}

// filterBenchmarkCoreStressItems keeps one deterministic subset of the heavier core rows for the filter scenario.
func filterBenchmarkCoreStressItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := make([]benchmarkshared.BenchmarkCoreRowData, 0, len(parseItems)/3)
	for _, getItem := range parseItems {
		if getItem.GetID%3 != 0 {
			continue
		}
		getItems = append(getItems, getItem)
	}
	return getItems
}

// buildBenchmarkCoreSortedItems sorts the current core-list rows into ascending stable row ID order.
func buildBenchmarkCoreSortedItems(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
	getItems := append([]benchmarkshared.BenchmarkCoreRowData(nil), parseItems...)
	sort.Slice(getItems, func(parseLeft int, parseRight int) bool {
		return getItems[parseLeft].GetID < getItems[parseRight].GetID
	})
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

// buildBenchmarkPrimitiveRows builds the flat primitive host-node payload used by the primitive benchmark scenarios.
func buildBenchmarkPrimitiveRows() []benchmarkPrimitiveRowData {
	getRows := make([]benchmarkPrimitiveRowData, benchmarkPrimitiveRowCount)
	for parseIndex := 0; parseIndex < benchmarkPrimitiveRowCount; parseIndex++ {
		getRows[parseIndex] = benchmarkPrimitiveRowData{
			GetID:    parseIndex + 1,
			GetLabel: fmt.Sprintf("Primitive %03d", parseIndex+1),
			GetState: "steady",
			IsActive: false,
		}
	}
	return getRows
}

// buildBenchmarkUpdatedPrimitiveTextRows updates only the visible text payload for the primitive text benchmark.
func buildBenchmarkUpdatedPrimitiveTextRows(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
	getRows := make([]benchmarkPrimitiveRowData, len(parseRows))
	for parseIndex, getRow := range parseRows {
		getRows[parseIndex] = benchmarkPrimitiveRowData{
			GetID:    getRow.GetID,
			GetLabel: getRow.GetLabel + " (Live)",
			GetState: getRow.GetState,
			IsActive: getRow.IsActive,
		}
	}
	return getRows
}

// buildBenchmarkUpdatedPrimitiveAttributeRows updates only the row attributes and classes for the primitive attribute benchmark.
func buildBenchmarkUpdatedPrimitiveAttributeRows(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
	getRows := make([]benchmarkPrimitiveRowData, len(parseRows))
	for parseIndex, getRow := range parseRows {
		getRows[parseIndex] = benchmarkPrimitiveRowData{
			GetID:    getRow.GetID,
			GetLabel: getRow.GetLabel,
			GetState: "active",
			IsActive: true,
		}
	}
	return getRows
}

// buildBenchmarkAppendedPrimitiveRows appends one fixed block of primitive rows for child-list append measurement.
func buildBenchmarkAppendedPrimitiveRows(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
	getRows := append([]benchmarkPrimitiveRowData(nil), parseRows...)
	for parseOffset := 0; parseOffset < benchmarkPrimitiveRemoveCount; parseOffset++ {
		getRowID := len(parseRows) + parseOffset + 1
		getRows = append(getRows, benchmarkPrimitiveRowData{
			GetID:    getRowID,
			GetLabel: fmt.Sprintf("Primitive %03d", getRowID),
			GetState: "steady",
			IsActive: false,
		})
	}
	return getRows
}

// buildBenchmarkTrimmedPrimitiveRows removes one fixed trailing block of primitive rows for child-list removal measurement.
func buildBenchmarkTrimmedPrimitiveRows(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
	if len(parseRows) <= benchmarkPrimitiveRemoveCount {
		return []benchmarkPrimitiveRowData{}
	}
	getKeepCount := len(parseRows) - benchmarkPrimitiveRemoveCount
	getRows := make([]benchmarkPrimitiveRowData, getKeepCount)
	copy(getRows, parseRows[:getKeepCount])
	return getRows
}

// buildBenchmarkEnterpriseMetrics builds one nested metric row set for the enterprise workspace scenarios.
func buildBenchmarkEnterpriseMetrics(parseSectionIndex int, parseRecordIndex int, parseRevision int) []benchmarkEnterpriseMetricData {
	getMetrics := make([]benchmarkEnterpriseMetricData, 0, benchmarkEnterpriseMetricCount)
	getMetrics = append(getMetrics,
		benchmarkEnterpriseMetricData{GetLabel: "owners", GetValue: strconv.Itoa(6 + parseSectionIndex + parseRevision)},
		benchmarkEnterpriseMetricData{GetLabel: "controls", GetValue: strconv.Itoa(14 + parseRecordIndex + parseRevision)},
		benchmarkEnterpriseMetricData{GetLabel: "sla", GetValue: strconv.Itoa(24+(parseSectionIndex*2)+parseRevision) + "h"},
		benchmarkEnterpriseMetricData{GetLabel: "risk", GetValue: "r" + strconv.Itoa(((parseSectionIndex+parseRecordIndex+parseRevision)%5)+1)},
	)
	return getMetrics
}

// buildBenchmarkEnterpriseSections builds one nested enterprise workspace payload for subtree update scenarios.
func buildBenchmarkEnterpriseSections() []benchmarkEnterpriseSectionData {
	getSectionTitles := []string{
		"Portfolio Oversight",
		"Capacity Planning",
		"Risk Controls",
		"Customer Rollout",
		"Compliance Evidence",
		"Executive Review",
	}
	getSections := make([]benchmarkEnterpriseSectionData, 0, benchmarkEnterpriseSectionCount)
	for parseSectionIndex := 0; parseSectionIndex < benchmarkEnterpriseSectionCount; parseSectionIndex++ {
		getSectionID := "section-" + strconv.Itoa(parseSectionIndex+1)
		getRecords := make([]benchmarkEnterpriseRecordData, 0, benchmarkEnterpriseRecordCount)
		for parseRecordIndex := 0; parseRecordIndex < benchmarkEnterpriseRecordCount; parseRecordIndex++ {
			getRecordID := fmt.Sprintf("%s-record-%02d", getSectionID, parseRecordIndex+1)
			getRecords = append(getRecords, benchmarkEnterpriseRecordData{
				GetID:       getRecordID,
				GetTitle:    fmt.Sprintf("Workstream %02d / Batch %02d", parseSectionIndex+1, parseRecordIndex+1),
				GetOwner:    fmt.Sprintf("team-%d", ((parseSectionIndex+parseRecordIndex)%4)+1),
				GetStatus:   "steady",
				GetRevision: 0,
				GetMetrics:  buildBenchmarkEnterpriseMetrics(parseSectionIndex, parseRecordIndex, 0),
			})
		}
		getSections = append(getSections, benchmarkEnterpriseSectionData{
			GetID:       getSectionID,
			GetTitle:    getSectionTitles[parseSectionIndex],
			GetSummary:  fmt.Sprintf("Nested enterprise review pack %d with shared controls, owners, and SLA markers.", parseSectionIndex+1),
			GetStatus:   "steady",
			GetRevision: 0,
			GetRecords:  getRecords,
		})
	}
	return getSections
}

// buildBenchmarkUpdatedEnterpriseSections updates one large nested enterprise subtree in place while preserving shape.
func buildBenchmarkUpdatedEnterpriseSections(parseSections []benchmarkEnterpriseSectionData) []benchmarkEnterpriseSectionData {
	getSections := make([]benchmarkEnterpriseSectionData, len(parseSections))
	copy(getSections, parseSections)
	for parseSectionIndex, getSection := range getSections {
		if getSection.GetID != benchmarkEnterpriseTargetSectionID {
			continue
		}
		getSection.GetRevision++
		getSection.GetStatus = "escalated"
		getSection.GetTitle = getSection.GetTitle + " (Escalated)"
		getSection.GetSummary = getSection.GetSummary + " Escalation review reopened."
		getRecords := make([]benchmarkEnterpriseRecordData, len(getSection.GetRecords))
		for parseRecordIndex, getRecord := range getSection.GetRecords {
			getRecord.GetRevision = getSection.GetRevision
			getRecord.GetStatus = "escalated"
			getRecord.GetTitle = getRecord.GetTitle + " / Escalation " + strconv.Itoa(getSection.GetRevision)
			getRecord.GetOwner = getRecord.GetOwner + "-priority"
			getRecord.GetMetrics = buildBenchmarkEnterpriseMetrics(parseSectionIndex, parseRecordIndex, getSection.GetRevision)
			getRecords[parseRecordIndex] = getRecord
		}
		getSection.GetRecords = getRecords
		getSections[parseSectionIndex] = getSection
	}
	return getSections
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
	getWorkerCount := buildBenchmarkWorkerCount(parseMode)
	if getWorkerCount <= 1 {
		return []string{"runtime2-shard-a"}
	}
	getSchedulerShardIDs := make([]string, 0, getWorkerCount)
	for parseWorkerIndex := 0; parseWorkerIndex < getWorkerCount; parseWorkerIndex++ {
		getSchedulerShardIDs = append(getSchedulerShardIDs, fmt.Sprintf("runtime2-shard-%02d", parseWorkerIndex+1))
	}
	return getSchedulerShardIDs
}

// buildBenchmarkRuntime3ChunkCount resolves the chunk fan-out used by one runtime2 benchmark mode.
func buildBenchmarkRuntime3ChunkCount(parseMode string) int {
	return buildBenchmarkWorkerCount(parseMode)
}

// buildBenchmarkRuntime3ChunkBounds partitions one list length into stable region-sized chunk bounds.
func buildBenchmarkRuntime3ChunkBounds(parseItemCount int, parseChunkCount int) [][2]int {
	if parseItemCount <= 0 {
		return nil
	}
	getChunkCount := parseChunkCount
	if getChunkCount < 1 {
		getChunkCount = 1
	}
	if getChunkCount > parseItemCount {
		getChunkCount = parseItemCount
	}
	getChunkSize := (parseItemCount + getChunkCount - 1) / getChunkCount
	getBounds := make([][2]int, 0, getChunkCount)
	for parseChunkIndex := 0; parseChunkIndex < getChunkCount; parseChunkIndex++ {
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

// buildBenchmarkRuntime3CorePropsFromChunks flattens worker-prepared core chunks into one stable core-region payload.
func buildBenchmarkRuntime3CorePropsFromChunks(parseCoreChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult) renderBenchmarkRuntime3CoreProps {
	getItems := make([]benchmarkshared.BenchmarkPreparedCoreItem, 0)
	getWorkerNames := make([]string, 0, len(parseCoreChunks))
	getSeenWorkerNames := map[string]struct{}{}
	getWorkDigest := uint64(1469598103934665603)
	for parseChunkIndex, getChunk := range parseCoreChunks {
		getItems = append(getItems, getChunk.GetItems...)
		getWorkerName := strings.TrimSpace(getChunk.GetWorker)
		if getWorkerName != "" {
			if _, hasWorkerName := getSeenWorkerNames[getWorkerName]; !hasWorkerName {
				getSeenWorkerNames[getWorkerName] = struct{}{}
				getWorkerNames = append(getWorkerNames, getWorkerName)
			}
		}
		getWorkDigest ^= getChunk.GetWorkDigest + uint64(parseChunkIndex+1)
		getWorkDigest *= 1099511628211
	}
	return renderBenchmarkRuntime3CoreProps{
		GetItems:      getItems,
		GetWorker:     strings.Join(getWorkerNames, ","),
		GetWorkDigest: getWorkDigest,
	}
}

// buildBenchmarkRuntime3CoreRegionNodes renders one worker-prepared core-list region set for the runtime2 benchmark modes.
func buildBenchmarkRuntime3CoreRegionNodes(parseMode string, parseCoreChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult, parseRefreshToken int) []ui.Node {
	getSchedulerShardIDs := buildBenchmarkRuntime3SchedulerShardIDs(parseMode)
	getModeKey := buildBenchmarkRuntime3ModeKey(parseMode)
	_ = parseRefreshToken
	getRegionNodes := make([]ui.Node, 0, len(parseCoreChunks))
	for parseChunkIndex, getChunk := range parseCoreChunks {
		getRegionNodes = append(getRegionNodes, ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3CoreProps]{
			RendererID:       benchmarkRuntime3CoreRendererID,
			RegionInstanceID: fmt.Sprintf("examples.render-benchmark.%s.core.%02d", getModeKey, parseChunkIndex),
			Props: renderBenchmarkRuntime3CoreProps{
				GetItems:      getChunk.GetItems,
				GetWorker:     getChunk.GetWorker,
				GetWorkDigest: getChunk.GetWorkDigest,
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
		getItemNode := Div(
			Class("benchmark-core-item rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-slate-100"),
			Data("row-id", strconv.Itoa(getItem.GetID)),
			Text(getItem.GetText),
		)
		getItems = append(getItems, WithKey(getItemNode, strconv.Itoa(getItem.GetID)))
	}
	return html.Div(
		html.Props{
			Class: "benchmark-core-region grid gap-2",
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
		},
		getItems...,
	)
}

// renderBenchmarkRuntime3DeepRegion renders the public parallel-region shell used for the experimental runtime3 deep-tree scenario.
func renderBenchmarkRuntime3DeepRegion(parseProps renderBenchmarkRuntime3DeepProps) ui.Node {
	return renderBenchmarkDeepTree(renderBenchmarkDeepTreeProps{
		GetDepth:        parseProps.GetDepth,
		GetRefreshToken: parseProps.GetRefreshToken,
		GetTreeVersion:  parseProps.GetTreeVersion,
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

// renderBenchmarkDeepTree renders one nested deep-tree node with explicit revision markers for deep update scenarios.
func renderBenchmarkDeepTree(parseProps renderBenchmarkDeepTreeProps) ui.Node {
	if parseProps.GetDepth <= 0 {
		return html.Div(
			html.Props{
				ID:    "benchmark-deep-leaf",
				Class: "benchmark-deep-leaf rounded-xl border border-cyan-400/20 bg-cyan-400/10 px-3 py-2 text-xs text-cyan-100",
				Data: map[string]string{
					"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
					"tree-version":  strconv.Itoa(parseProps.GetTreeVersion),
				},
			},
			html.Text("Leaf / Revision "+strconv.Itoa(parseProps.GetTreeVersion)),
		)
	}
	return html.Div(
		html.Props{
			Class: "benchmark-deep-node border-l border-white/10 pl-2",
			Data: map[string]string{
				"depth":         strconv.Itoa(parseProps.GetDepth),
				"refresh-token": strconv.Itoa(parseProps.GetRefreshToken),
				"tree-version":  strconv.Itoa(parseProps.GetTreeVersion),
			},
		},
		html.Div(
			html.Props{Class: "benchmark-deep-label mb-2 text-[11px] uppercase tracking-[0.18em] text-slate-400"},
			html.Text("Compliance Layer "+strconv.Itoa(parseProps.GetDepth)+" / Revision "+strconv.Itoa(parseProps.GetTreeVersion)),
		),
		ui.CreateElement(renderBenchmarkDeepTree, renderBenchmarkDeepTreeProps{
			GetDepth:        parseProps.GetDepth - 1,
			GetRefreshToken: parseProps.GetRefreshToken,
			GetTreeVersion:  parseProps.GetTreeVersion,
		}),
	)
}

// renderBenchmarkDeepTreeRoot renders the deep-tree benchmark container with stable root markers.
func renderBenchmarkDeepTreeRoot(parseDepth int, parseRefreshToken int, parseTreeVersion int) ui.Node {
	return html.Div(
		html.Props{
			ID:    "benchmark-deep-root",
			Class: "grid gap-2",
			Data: map[string]string{
				"refresh-token": strconv.Itoa(parseRefreshToken),
				"tree-version":  strconv.Itoa(parseTreeVersion),
			},
		},
		ui.CreateElement(renderBenchmarkDeepTree, renderBenchmarkDeepTreeProps{
			GetDepth:        parseDepth,
			GetRefreshToken: parseRefreshToken,
			GetTreeVersion:  parseTreeVersion,
		}),
	)
}

// renderBenchmarkEnterpriseView renders one nested enterprise workspace tree for large subtree mutation benchmarks.
func renderBenchmarkEnterpriseView(parseSections []benchmarkEnterpriseSectionData, parseRefreshToken int) ui.Node {
	getSectionNodes := make([]ui.Node, 0, len(parseSections))
	for _, getSection := range parseSections {
		getRecordNodes := make([]ui.Node, 0, len(getSection.GetRecords))
		for _, getRecord := range getSection.GetRecords {
			getMetricNodes := make([]ui.Node, 0, len(getRecord.GetMetrics))
			for _, getMetric := range getRecord.GetMetrics {
				getMetricNodes = append(getMetricNodes, html.Span(
					html.Props{
						Class: "benchmark-enterprise-metric rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300",
						Data:  map[string]string{"metric-label": getMetric.GetLabel},
					},
					html.Text(getMetric.GetLabel+": "+getMetric.GetValue),
				))
			}
			getRecordNodes = append(getRecordNodes, html.Article(
				html.Props{
					Key:   getRecord.GetID,
					Class: "benchmark-enterprise-record rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20",
					Data: map[string]string{
						"section-id":          getSection.GetID,
						"record-id":           getRecord.GetID,
						"enterprise-revision": strconv.Itoa(getRecord.GetRevision),
					},
				},
				html.Div(
					html.Props{Class: "flex items-center justify-between gap-3"},
					html.H3(html.Props{Class: "benchmark-enterprise-record-title text-sm font-semibold text-white"}, html.Text(getRecord.GetTitle)),
					html.Span(html.Props{Class: "benchmark-enterprise-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1 text-[11px] uppercase tracking-[0.16em] text-cyan-100"}, html.Text(getRecord.GetStatus)),
				),
				html.P(html.Props{Class: "mt-2 text-xs uppercase tracking-[0.16em] text-slate-400"}, html.Text("Owner "+getRecord.GetOwner)),
				html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-2"}, getMetricNodes...),
			))
		}
		getSectionNodes = append(getSectionNodes, html.Section(
			html.Props{
				Key:   getSection.GetID,
				Class: "benchmark-enterprise-section rounded-[24px] border border-white/10 bg-slate-950/35 p-5",
				Data: map[string]string{
					"section-id":          getSection.GetID,
					"enterprise-revision": strconv.Itoa(getSection.GetRevision),
					"refresh-token":       strconv.Itoa(parseRefreshToken),
				},
			},
			html.Div(
				html.Props{Class: "flex items-center justify-between gap-3"},
				html.Div(
					html.Props{Class: "space-y-2"},
					html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-slate-400"}, html.Text(getSection.GetID)),
					html.H2(html.Props{Class: "text-lg font-semibold text-white"}, html.Text(getSection.GetTitle)),
					html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(getSection.GetSummary)),
				),
				html.Span(html.Props{Class: "benchmark-enterprise-section-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] uppercase tracking-[0.16em] text-cyan-100"}, html.Text(getSection.GetStatus)),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 xl:grid-cols-2"}, getRecordNodes...),
		))
	}
	return html.Div(
		html.Props{
			ID:    "enterprise-container",
			Class: "grid gap-4",
			Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
		},
		getSectionNodes...,
	)
}

// renderBenchmarkPrimitiveView renders the flat primitive host-node grid used to isolate basic DOM operation costs.
func renderBenchmarkPrimitiveView(parseRows []benchmarkPrimitiveRowData, parseRefreshToken int) ui.Node {
	getRowNodes := make([]ui.Node, 0, len(parseRows))
	for _, getRow := range parseRows {
		getRowClass := "benchmark-primitive-row rounded-xl border px-3 py-2 text-sm text-slate-100 transition-colors"
		if getRow.IsActive {
			getRowClass += " benchmark-primitive-row-active border-cyan-400/30 bg-cyan-400/12"
		} else {
			getRowClass += " border-white/10 bg-white/[0.04]"
		}
		getRowNodes = append(getRowNodes, html.Div(
			html.Props{
				Key:   strconv.Itoa(getRow.GetID),
				Class: getRowClass,
				Data: map[string]string{
					"primitive-id":    strconv.Itoa(getRow.GetID),
					"primitive-state": getRow.GetState,
				},
			},
			html.Span(
				html.Props{Class: "benchmark-primitive-label"},
				html.Text(getRow.GetLabel),
			),
		))
	}
	return html.Div(
		html.Props{
			ID:    "primitive-container",
			Class: "grid gap-2 sm:grid-cols-2 xl:grid-cols-4",
			Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
		},
		getRowNodes...,
	)
}

// buildBenchmarkRuntimeNode renders the active benchmark scenario through the current runtime path.
func buildBenchmarkRuntimeNode(parseView string, parseCoreItems []benchmarkshared.BenchmarkCoreRowData, parseContentItems []benchmarkshared.BenchmarkContentCardData, parsePrimitiveRows []benchmarkPrimitiveRowData, parseEnterpriseSections []benchmarkEnterpriseSectionData, parseTreeDepth int, parseTreeVersion int, parseHookCount int, parseRefreshToken int) ui.Node {
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
		return renderBenchmarkDeepTreeRoot(parseTreeDepth, parseRefreshToken, parseTreeVersion)
	case "primitive":
		return renderBenchmarkPrimitiveView(parsePrimitiveRows, parseRefreshToken)
	case "enterprise":
		return renderBenchmarkEnterpriseView(parseEnterpriseSections, parseRefreshToken)
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
				html.Props{
					Key:   strconv.Itoa(getItem.GetID),
					Class: "benchmark-core-item rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-slate-100",
					Data:  map[string]string{"row-id": strconv.Itoa(getItem.GetID)},
				},
				html.Text(getItem.GetText),
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
func buildBenchmarkRuntime3Node(parseMode string, parseView string, parseCoreItems []benchmarkshared.BenchmarkCoreRowData, parseCoreChunks []benchmarkshared.BenchmarkWorkerCoreChunkResult, parseContentChunks []benchmarkshared.BenchmarkWorkerContentChunkResult, parsePrimitiveRows []benchmarkPrimitiveRowData, parseEnterpriseSections []benchmarkEnterpriseSectionData, parseTreeDepth int, parseTreeVersion int, parseHookCount int, parseRefreshToken int) ui.Node {
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
		return html.Div(
			html.Props{
				ID:    "benchmark-deep-root",
				Class: "grid gap-2",
				Data: map[string]string{
					"refresh-token": strconv.Itoa(parseRefreshToken),
					"tree-version":  strconv.Itoa(parseTreeVersion),
				},
			},
			ui.ParallelRegion(ui.ParallelRegionSpec[renderBenchmarkRuntime3DeepProps]{
				RendererID:       benchmarkRuntime3DeepRendererID,
				RegionInstanceID: benchmarkRuntime3DeepRegionID,
				Props: renderBenchmarkRuntime3DeepProps{
					GetDepth:        parseTreeDepth,
					GetRefreshToken: parseRefreshToken,
					GetTreeVersion:  parseTreeVersion,
				},
				SchedulerShardIDs: getSchedulerShardIDs,
			}),
		)
	case "primitive":
		return renderBenchmarkPrimitiveView(parsePrimitiveRows, parseRefreshToken)
	case "enterprise":
		return renderBenchmarkEnterpriseView(parseEnterpriseSections, parseRefreshToken)
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
		getCoreRegionChunks := parseCoreChunks
		if hasBenchmarkWorkerCoreFastPath(parseMode, len(parseCoreItems)) {
			getFastPathChunks, _ := requestBenchmarkWorkerCoreChunksByLocalCache(parseCoreItems, buildBenchmarkWorkerWorkScale(), 0)
			if len(getFastPathChunks) > 0 {
				getCoreRegionChunks = getFastPathChunks
			}
		}
		return html.Div(
			html.Props{
				ID:    "core-list-container",
				Class: "grid gap-2",
				Data:  map[string]string{"refresh-token": strconv.Itoa(parseRefreshToken)},
			},
			// Keep one region per prepared chunk so append-only updates can preserve unchanged regions.
			buildBenchmarkRuntime3CoreRegionNodes(parseMode, getCoreRegionChunks, parseRefreshToken)...,
		)
	}
}

// renderBenchmarkApp renders the benchmark control surface and active browser scenario for the requested framework mode.
func renderBenchmarkApp(parseProps renderBenchmarkAppProps) ui.Node {
	getCoreItems := ui.UseState([]benchmarkshared.BenchmarkCoreRowData{})
	getContentItems := ui.UseState([]benchmarkshared.BenchmarkContentCardData{})
	getPrimitiveRows := ui.UseState([]benchmarkPrimitiveRowData{})
	getEnterpriseSections := ui.UseState([]benchmarkEnterpriseSectionData{})
	getCoreChunks := ui.UseState([]benchmarkshared.BenchmarkWorkerCoreChunkResult{})
	getContentChunks := ui.UseState([]benchmarkshared.BenchmarkWorkerContentChunkResult{})
	getView := ui.UseState("core")
	getTreeDepth := ui.UseState(0)
	getTreeVersion := ui.UseState(0)
	getHookCount := ui.UseState(0)
	getRefreshToken := ui.UseState(0)
	getLastAction := ui.UseState("idle")
	getWorkersRef := ui.UseRef[[]interop.Worker](nil)
	getWorkersRevision := ui.UseState(0)
	getWorkerGenerationRef := ui.UseRef(uint64(0))
	getCoreChunkCacheByDependencyRef := ui.UseRef(map[uint64][]benchmarkshared.BenchmarkWorkerCoreChunkResult{})
	getContentChunkCacheByDependencyRef := ui.UseRef(map[uint64][]benchmarkshared.BenchmarkWorkerContentChunkResult{})
	getLastCorePreparedItemsRef := ui.UseRef([]benchmarkshared.BenchmarkCoreRowData(nil))
	getPrepareRevision := ui.UseState(0)
	getWorkerState := ui.UseState(buildBenchmarkWorkerState{
		GetWorkerCount: buildBenchmarkWorkerCount(parseProps.GetMode),
	})
	clearBenchmarkPreparedChunks := func() {
		getCoreChunks.Set(nil)
		getContentChunks.Set(nil)
	}
	handleBenchmarkApplyCoreFastPath := func(parseCoreItems []benchmarkshared.BenchmarkCoreRowData) bool {
		if !hasBenchmarkWorkerMode(parseProps.GetMode) {
			return false
		}
		if !hasBenchmarkWorkerCoreFastPath(parseProps.GetMode, len(parseCoreItems)) {
			return false
		}
		warnBenchmarkWorkerCoreFastPath()
		parseStartedAt := time.Now()
		getGeneration := getWorkerGenerationRef.Get() + 1
		getWorkerGenerationRef.Set(getGeneration)
		getPreparedChunks, getCacheHitCount := requestBenchmarkWorkerCoreChunksByLocalCache(parseCoreItems, buildBenchmarkWorkerWorkScale(), getGeneration)
		getCoreItemsDependency := buildBenchmarkWorkerCoreItemsDependency(parseCoreItems)
		storeBenchmarkWorkerCoreChunkCache(getCoreChunkCacheByDependencyRef, getCoreItemsDependency, getPreparedChunks)
		getCoreChunks.Set(cloneBenchmarkWorkerCoreChunks(getPreparedChunks))
		getLastCorePreparedItemsRef.Set(buildBenchmarkCoreItemsClone(parseCoreItems))
		getWorkerState.Update(func(parsePrevious buildBenchmarkWorkerState) buildBenchmarkWorkerState {
			parsePrevious.IsPreparing = false
			parsePrevious.IsReady = true
			parsePrevious.GetPreparedBatchCount++
			parsePrevious.GetPreparedChunks = len(getPreparedChunks)
			parsePrevious.GetAdaptiveChunkCount = len(getPreparedChunks)
			parsePrevious.GetCacheHitCount = getCacheHitCount
			parsePrevious.GetPreparedItems = len(parseCoreItems)
			parsePrevious.GetLastBatchMS = time.Since(parseStartedAt).Milliseconds()
			parsePrevious.GetErrorText = ""
			return parsePrevious
		})
		return true
	}
	handleBenchmarkPrepareBump := func() {
		getPrepareRevision.Update(func(parseValue int) int {
			return parseValue + 1
		})
	}
	handleBenchmarkWorkerFleetEffect(parseProps.GetMode, getWorkersRef, getWorkersRevision, getWorkerState)
	handleBenchmarkWorkerPrepareEffect(
		parseProps.GetMode,
		getView.Get(),
		getCoreItems.Get(),
		getContentItems.Get(),
		getPrepareRevision.Get(),
		getWorkersRevision.Get(),
		getWorkersRef,
		getWorkerGenerationRef,
		getCoreChunkCacheByDependencyRef,
		getContentChunkCacheByDependencyRef,
		getLastCorePreparedItemsRef,
		getCoreChunks,
		getContentChunks,
		getWorkerState,
	)

	handleCoreRender := ui.UseEvent(func() {
		getCoreRenderItems := buildBenchmarkCoreItems()
		getView.Set("core")
		getCoreItems.Set(getCoreRenderItems)
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		if !handleBenchmarkApplyCoreFastPath(getCoreRenderItems) {
			handleBenchmarkPrepareBump()
		}
		getLastAction.Set("core-render")
	})
	handleCoreStressRender := ui.UseEvent(func() {
		getView.Set("core")
		getCoreItems.Set(buildBenchmarkCoreStressItems())
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-stress-render")
	})
	handleCoreUpdate := ui.UseEvent(func() {
		getView.Set("core")
		getUpdatedCoreItems := buildBenchmarkUpdatedCoreItems(getCoreItems.Get())
		getCoreItems.Set(getUpdatedCoreItems)
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		if !handleBenchmarkApplyCoreFastPath(getUpdatedCoreItems) {
			handleBenchmarkPrepareBump()
		}
		getLastAction.Set("core-update")
	})
	handleCoreAppend := ui.UseEvent(func() {
		getView.Set("core")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getCoreItems.Update(func(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
			return buildBenchmarkCoreAppendedItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-append")
	})
	handleCorePrepend := ui.UseEvent(func() {
		getView.Set("core")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getCoreItems.Update(func(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
			return buildBenchmarkCorePrependedItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-prepend")
	})
	handleCoreReverse := ui.UseEvent(func() {
		getView.Set("core")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getCoreItems.Update(func(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
			return buildBenchmarkCoreReversedItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-reverse")
	})
	handleCoreFilter := ui.UseEvent(func() {
		getView.Set("core")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getCoreItems.Update(func(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
			return filterBenchmarkCoreStressItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-filter")
	})
	handleCoreSort := ui.UseEvent(func() {
		getView.Set("core")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getCoreItems.Update(func(parseItems []benchmarkshared.BenchmarkCoreRowData) []benchmarkshared.BenchmarkCoreRowData {
			return buildBenchmarkCoreSortedItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-sort")
	})
	handleCoreClear := ui.UseEvent(func() {
		getView.Set("core")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("core-clear")
	})
	handleContentRender := ui.UseEvent(func() {
		getView.Set("content")
		getContentItems.Set(buildBenchmarkContentItems())
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-render")
	})
	handleContentUpdate := ui.UseEvent(func() {
		getView.Set("content")
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		getContentItems.Update(func(parseItems []benchmarkshared.BenchmarkContentCardData) []benchmarkshared.BenchmarkContentCardData {
			return buildBenchmarkUpdatedContentItems(parseItems)
		})
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-update")
	})
	handleContentClear := ui.UseEvent(func() {
		getView.Set("content")
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("content-clear")
	})
	handleDeepRender := ui.UseEvent(func() {
		getView.Set("deep")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(benchmarkDeepTreeDepth)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("deep-render")
	})
	handleDeepUpdate := ui.UseEvent(func() {
		getView.Set("deep")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		if getTreeDepth.Get() == 0 {
			getTreeDepth.Set(benchmarkDeepTreeDepth)
		}
		getTreeVersion.Update(func(parseValue int) int {
			return parseValue + 1
		})
		getRefreshToken.Update(func(parseValue int) int {
			return parseValue + 1
		})
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("deep-update")
	})
	handleEnterpriseRender := ui.UseEvent(func() {
		getView.Set("enterprise")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set(buildBenchmarkEnterpriseSections())
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("enterprise-render")
	})
	handleEnterpriseSubtreeUpdate := ui.UseEvent(func() {
		getView.Set("enterprise")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Update(func(parseSections []benchmarkEnterpriseSectionData) []benchmarkEnterpriseSectionData {
			if len(parseSections) == 0 {
				return buildBenchmarkUpdatedEnterpriseSections(buildBenchmarkEnterpriseSections())
			}
			return buildBenchmarkUpdatedEnterpriseSections(parseSections)
		})
		getRefreshToken.Update(func(parseValue int) int {
			return parseValue + 1
		})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("enterprise-subtree-update")
	})
	handlePrimitiveRender := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set(buildBenchmarkPrimitiveRows())
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-render")
	})
	handlePrimitiveTextUpdate := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Update(func(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
			if len(parseRows) == 0 {
				return buildBenchmarkUpdatedPrimitiveTextRows(buildBenchmarkPrimitiveRows())
			}
			return buildBenchmarkUpdatedPrimitiveTextRows(parseRows)
		})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-text-update")
	})
	handlePrimitiveAttributeUpdate := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Update(func(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
			if len(parseRows) == 0 {
				return buildBenchmarkUpdatedPrimitiveAttributeRows(buildBenchmarkPrimitiveRows())
			}
			return buildBenchmarkUpdatedPrimitiveAttributeRows(parseRows)
		})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-attribute-update")
	})
	handlePrimitiveAppend := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Update(func(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
			if len(parseRows) == 0 {
				return buildBenchmarkAppendedPrimitiveRows(buildBenchmarkPrimitiveRows())
			}
			return buildBenchmarkAppendedPrimitiveRows(parseRows)
		})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-append")
	})
	handlePrimitiveRemove := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Update(func(parseRows []benchmarkPrimitiveRowData) []benchmarkPrimitiveRowData {
			if len(parseRows) == 0 {
				return buildBenchmarkTrimmedPrimitiveRows(buildBenchmarkPrimitiveRows())
			}
			return buildBenchmarkTrimmedPrimitiveRows(parseRows)
		})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-remove")
	})
	handlePrimitiveClear := ui.UseEvent(func() {
		getView.Set("primitive")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
		getHookCount.Set(0)
		handleBenchmarkPrepareBump()
		getLastAction.Set("primitive-clear")
	})
	handleHooksRender := ui.UseEvent(func() {
		getView.Set("hooks")
		getCoreItems.Set([]benchmarkshared.BenchmarkCoreRowData{})
		getContentItems.Set([]benchmarkshared.BenchmarkContentCardData{})
		getPrimitiveRows.Set([]benchmarkPrimitiveRowData{})
		getEnterpriseSections.Set([]benchmarkEnterpriseSectionData{})
		clearBenchmarkPreparedChunks()
		getTreeDepth.Set(0)
		getTreeVersion.Set(0)
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
	switch {
	case parseProps.GetMode == benchmarkModeRuntime3, hasBenchmarkWorkerMode(parseProps.GetMode):
		getBenchmarkNode = buildBenchmarkRuntime3Node(
			parseProps.GetMode,
			getView.Get(),
			getCoreItems.Get(),
			getCoreChunks.Get(),
			getContentChunks.Get(),
			getPrimitiveRows.Get(),
			getEnterpriseSections.Get(),
			getTreeDepth.Get(),
			getTreeVersion.Get(),
			getHookCount.Get(),
			getRefreshToken.Get(),
		)
	default:
		getBenchmarkNode = buildBenchmarkRuntimeNode(
			getView.Get(),
			getCoreItems.Get(),
			getContentItems.Get(),
			getPrimitiveRows.Get(),
			getEnterpriseSections.Get(),
			getTreeDepth.Get(),
			getTreeVersion.Get(),
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
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Batch Count")), html.P(html.Props{ID: "metric-worker-batch-count", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(getWorkerSnapshot.GetPreparedBatchCount)))),
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Prepared Items")), html.P(html.Props{ID: "metric-worker-items", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.Itoa(getWorkerSnapshot.GetPreparedItems)))),
			html.Div(html.Props{Class: "rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"}, html.P(html.Props{Class: "text-[11px] uppercase tracking-[0.18em] text-cyan-100"}, html.Text("Last Batch")), html.P(html.Props{ID: "metric-worker-batch-ms", Class: "mt-2 text-lg font-semibold text-white"}, html.Text(strconv.FormatInt(getWorkerSnapshot.GetLastBatchMS, 10)+" ms"))),
		)
	}

	getControlsNode := ui.UseMemo(func() ui.Node {
		return html.Div(
			html.Props{ID: "benchmark-controls", Class: "mt-8 grid gap-3 md:grid-cols-3 xl:grid-cols-5"},
			html.Button(html.Props{ID: "btn-core-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleCoreRender}, html.Text("Render Core Items")),
			html.Button(html.Props{ID: "btn-core-stress-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleCoreStressRender}, html.Text("Render Core Stress")),
			html.Button(html.Props{ID: "btn-core-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreUpdate}, html.Text("Update Core Items")),
			html.Button(html.Props{ID: "btn-core-append", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreAppend}, html.Text("Append Core Rows")),
			html.Button(html.Props{ID: "btn-core-prepend", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCorePrepend}, html.Text("Prepend Core Rows")),
			html.Button(html.Props{ID: "btn-core-reverse", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreReverse}, html.Text("Reverse Core Rows")),
			html.Button(html.Props{ID: "btn-core-filter", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreFilter}, html.Text("Filter Core Rows")),
			html.Button(html.Props{ID: "btn-core-sort", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleCoreSort}, html.Text("Sort Core Rows")),
			html.Button(html.Props{ID: "btn-core-clear", Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: handleCoreClear}, html.Text("Clear Core Items")),
			html.Button(html.Props{ID: "btn-content-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleContentRender}, html.Text("Render Content Cards")),
			html.Button(html.Props{ID: "btn-content-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleContentUpdate}, html.Text("Update Content Cards")),
			html.Button(html.Props{ID: "btn-content-clear", Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: handleContentClear}, html.Text("Clear Content Cards")),
			html.Button(html.Props{ID: "btn-primitive-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handlePrimitiveRender}, html.Text("Render Primitive Grid")),
			html.Button(html.Props{ID: "btn-primitive-text-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handlePrimitiveTextUpdate}, html.Text("Update Primitive Text")),
			html.Button(html.Props{ID: "btn-primitive-attr-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handlePrimitiveAttributeUpdate}, html.Text("Update Primitive Attrs")),
			html.Button(html.Props{ID: "btn-primitive-append", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handlePrimitiveAppend}, html.Text("Append Primitive Rows")),
			html.Button(html.Props{ID: "btn-primitive-remove", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handlePrimitiveRemove}, html.Text("Remove Primitive Rows")),
			html.Button(html.Props{ID: "btn-primitive-clear", Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: handlePrimitiveClear}, html.Text("Clear Primitive Grid")),
			html.Button(html.Props{ID: "btn-deep-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleDeepRender}, html.Text("Render Deep Tree")),
			html.Button(html.Props{ID: "btn-deep-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleDeepUpdate}, html.Text("Update Deep Tree")),
			html.Button(html.Props{ID: "btn-enterprise-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleEnterpriseRender}, html.Text("Render Enterprise Workspace")),
			html.Button(html.Props{ID: "btn-enterprise-update", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleEnterpriseSubtreeUpdate}, html.Text("Update Enterprise Section")),
			html.Button(html.Props{ID: "btn-hooks-render", Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: handleHooksRender}, html.Text("Render Hook Grid")),
			html.Button(html.Props{ID: "btn-refresh", Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: handleRefresh}, html.Text("Refresh Current View")),
		)
	})

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
					html.Text("This page exposes one benchmark subject with a stable DOM contract. The same browser-side Playwright harness drives React 19.2.4, runtime1, runtime2 with one Go WASM worker for chunk preparation, and runtime2 with a configurable Go WASM worker count for the same off-main-thread preparation step."),
				),
				getControlsNode,
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
	if getMode == benchmarkModeRuntime3 || hasBenchmarkWorkerMode(getMode) {
		registerBenchmarkRuntime3Renderers()
	}
	ui.Render(ui.CreateElement(renderBenchmarkApp, renderBenchmarkAppProps{GetMode: getMode}), "#app")
	utils.WaitForever()
}
