package runtime

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
)

type DiagnosticSeverity string

const (
	DiagnosticInfo    DiagnosticSeverity = "info"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticError   DiagnosticSeverity = "error"
)

type DiagnosticClassification string

const (
	DiagnosticInformational      DiagnosticClassification = "informational"
	DiagnosticCorrectness        DiagnosticClassification = "correctness"
	DiagnosticPerformance        DiagnosticClassification = "performance"
	DiagnosticRecovered          DiagnosticClassification = "recovered"
	DiagnosticUnsupportedRecover DiagnosticClassification = "unsupported_recovered"
)

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// Diagnostic describes one deduplicated runtime diagnostic entry.
type Diagnostic struct {
	Source         string
	Severity       DiagnosticSeverity
	Classification DiagnosticClassification
	Code           string
	Docs           string
	Remediation    string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	Message        string
	Count          int
	Path           string
	ComponentStack []string
	Fields         map[string]string
}

// LogEntry describes one recent structured framework log.
type LogEntry struct {
	Domain         string
	Level          LogLevel
	Classification DiagnosticClassification
	Code           string
	Docs           string
	Remediation    string
	Recoverable    bool
	TopFrame       string
	Consequence    string
	Message        string
	Timestamp      string
	CorrelationID  string
	Fields         map[string]string
}

// HookSnapshot captures one hook entry from an inspected fiber.
type HookSnapshot struct {
	Slot         int
	Kind         string
	Value        string
	Dependencies string
	Status       string
}

// FiberSnapshot captures one inspected fiber subtree.
type FiberSnapshot struct {
	Name string
	Path string
	// AgentRef is the stable key/index-disambiguated ref the agent bridge
	// resolves back to this node (empty on the root, which is not addressable).
	AgentRef          string
	Kind              string
	Dirty             bool
	NeedsUpdate       bool
	FineGrained       bool
	ReactiveSource    string
	UpdateOrigin      string
	EffectCount       int
	HookCount         int
	Signature         *ComponentSignature
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
	Hooks             []HookSnapshot
	Children          []FiberSnapshot
}

// HotBranchSnapshot captures one high-cost subtree from profiling output.
type HotBranchSnapshot struct {
	Name              string
	Kind              string
	Path              string
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
}

// ProfilingPhaseTotalsSnapshot summarizes attributed phase time totals.
type ProfilingPhaseTotalsSnapshot struct {
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
}

// ComponentRenderTraceSnapshot captures per-component render activity.
type ComponentRenderTraceSnapshot struct {
	Name                    string
	Path                    string
	RenderCount             int
	RerenderCount           int
	LastTrigger             string
	LastRenderDurationNs    int64
	TotalRenderDurationNs   int64
	AverageRenderDurationNs int64
	LastRenderedAt          string
	TriggerCounts           map[string]int
}

// FlamegraphFrameSnapshot captures one nested flamegraph frame.
type FlamegraphFrameSnapshot struct {
	Name              string
	Kind              string
	Path              string
	Depth             int
	StartNs           int64
	DurationNs        int64
	SelfDurationNs    int64
	RenderDurationNs  int64
	DiffDurationNs    int64
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
}

// StartupProfilingSnapshot summarizes startup and hydration milestones.
type StartupProfilingSnapshot struct {
	Mode                       string
	StartedAt                  string
	BootstrapReadDurationNs    int64
	WASMTransferBytes          int64
	WASMDecodedBytes           int64
	BootstrapDecodedBytes      int64
	CacheWarmupDurationNs      int64
	ServiceWorkerOverheadNs    int64
	InitialRouteDataBytes      int64
	HydrationDurationNs        int64
	StartupCommitDurationNs    int64
	FirstInteractionDurationNs int64
	FirstInteractionCaptured   bool
	FirstInteractionEvent      string
	RouteBudgets               []RouteStartupBudgetSnapshot
}

// RouteStartupBudgetSnapshot captures startup budget samples grouped by route family.
type RouteStartupBudgetSnapshot struct {
	RouteFamily                       string
	LastRoutePath                     string
	SampleCount                       int
	AverageBootstrapReadDurationNs    int64
	AverageWASMTransferBytes          int64
	AverageWASMDecodedBytes           int64
	AverageBootstrapDecodedBytes      int64
	AverageCacheWarmupDurationNs      int64
	AverageServiceWorkerOverheadNs    int64
	AverageInitialRouteDataBytes      int64
	AverageHydrationDurationNs        int64
	AverageStartupCommitDurationNs    int64
	AverageFirstInteractionDurationNs int64
}

type HydrationDebugSnapshot struct {
	CorrelationID        string
	StartedAt            string
	FinishedAt           string
	DurationNs           int64
	ExistingDOMNodeCount int
	FallbackCount        int
	MismatchCount        int
	DiscardedNodeCount   int
	Strict               bool
	Failed               bool
	Failure              string
	RecentMessages       []string
}

// InspectionStats summarizes the inspected runtime tree.
type InspectionStats struct {
	TotalFibers       int
	DirtyFibers       int
	ComponentFibers   int
	HostFibers        int
	TextFibers        int
	FineGrainedFibers int
	HookEntries       int
	Effects           int
}

// ProfilingSnapshot summarizes runtime profiling counters and hot branches.
type ProfilingSnapshot struct {
	RenderCalls                      int
	ScheduledRootUpdates             int
	ScheduledFiberMarks              int
	ScheduledGranularMarks           int
	WorkLoopPasses                   int
	ProcessedUnits                   int
	CommitCount                      int
	FineGrainedCommits               int
	FineGrainedDescendantHostCommits int
	FineGrainedDescendantTextCommits int
	EffectExecutions                 int
	CleanupExecutions                int
	LastRenderDurationNs             int64
	LastCommitDurationNs             int64
	LastEffectDurationNs             int64
	LastCleanupDurationNs            int64
	PhaseTotals                      ProfilingPhaseTotalsSnapshot
	ComponentRenders                 []ComponentRenderTraceSnapshot
	RecentEvents                     []ProfilingEvent
	FlamegraphFrames                 []FlamegraphFrameSnapshot
	Startup                          StartupProfilingSnapshot
	HotBranches                      []HotBranchSnapshot
}

// InspectionSnapshot is the top-level runtime inspection payload.
type InspectionSnapshot struct {
	Root        *FiberSnapshot
	Stats       InspectionStats
	Profiling   ProfilingSnapshot
	Hydration   HydrationDebugSnapshot
	Diagnostics []Diagnostic
	Logs        []LogEntry
}

// inspectFiberTree is a core package helper.
func inspectFiberTree(parseFiber *Fiber) (*FiberSnapshot, InspectionStats) {
	return inspectFiberTreeWithPath(parseFiber, nil)
}

// inspectFiberTreeWithPath is a core package helper.
func inspectFiberTreeWithPath(parseFiber *Fiber, parsePath []string) (*FiberSnapshot, InspectionStats) {
	if parseFiber == nil {
		return nil, InspectionStats{}
	}

	parseKind, parseName := describeFiber(parseFiber)
	parseHooks := inspectHooks(parseFiber.hooks)
	parseCurrentPath := append(append([]string(nil), parsePath...), parseName)
	parseNode := &FiberSnapshot{
		Name:              parseName,
		Path:              strings.Join(parseCurrentPath, " > "),
		Kind:              parseKind,
		Dirty:             parseFiber.dirty,
		NeedsUpdate:       parseFiber.needsUpdate,
		FineGrained:       parseFiber.fineGrained,
		ReactiveSource:    firstNonEmpty(strings.Join(parseFiber.reactiveSourceIDs, ","), parseFiber.reactiveAtomID),
		UpdateOrigin:      parseFiber.updateOrigin,
		EffectCount:       len(parseFiber.effects),
		HookCount:         len(parseHooks),
		Signature:         buildComponentSignature(parseFiber, parseFiber.hooks),
		RenderDurationNs:  parseFiber.renderDurationNs,
		DiffDurationNs:    parseFiber.diffDurationNs,
		CommitDurationNs:  parseFiber.commitDurationNs,
		EffectDurationNs:  parseFiber.effectDurationNs,
		CleanupDurationNs: parseFiber.cleanupDurationNs,
		Hooks:             parseHooks,
	}
	parseNode.SelfDurationNs = parseNode.RenderDurationNs + parseNode.DiffDurationNs + parseNode.CommitDurationNs + parseNode.EffectDurationNs + parseNode.CleanupDurationNs

	parseStats := InspectionStats{
		TotalFibers: 1,
		HookEntries: len(parseHooks),
		Effects:     len(parseFiber.effects),
	}
	if parseFiber.dirty || parseFiber.needsUpdate {
		parseStats.DirtyFibers++
	}

	parseChildSubtreeDurationNs := int64(0)

	switch parseKind {
	case "component":
		parseStats.ComponentFibers++
	case "host", "root":
		parseStats.HostFibers++
	case "text":
		parseStats.TextFibers++
	}
	if parseFiber.fineGrained {
		parseStats.FineGrainedFibers++
	}

	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseChildSnapshot, parseChildStats := inspectFiberTreeWithPath(parseChild, parseCurrentPath)
		if parseChildSnapshot != nil {
			parseNode.Children = append(parseNode.Children, *parseChildSnapshot)
			parseChildSubtreeDurationNs += parseChildSnapshot.SubtreeDurationNs
		}
		parseStats.TotalFibers += parseChildStats.TotalFibers
		parseStats.DirtyFibers += parseChildStats.DirtyFibers
		parseStats.ComponentFibers += parseChildStats.ComponentFibers
		parseStats.HostFibers += parseChildStats.HostFibers
		parseStats.TextFibers += parseChildStats.TextFibers
		parseStats.FineGrainedFibers += parseChildStats.FineGrainedFibers
		parseStats.HookEntries += parseChildStats.HookEntries
		parseStats.Effects += parseChildStats.Effects
	}
	parseNode.SubtreeDurationNs = parseNode.SelfDurationNs + parseChildSubtreeDurationNs

	return parseNode, parseStats
}

// collectHotBranches is a core package helper.
func collectHotBranches(parseRoot *FiberSnapshot, parseLimit int) []HotBranchSnapshot {
	if parseRoot == nil || parseLimit <= 0 {
		return nil
	}

	parseBranches := make([]HotBranchSnapshot, 0, parseLimit)
	var parseWalk func(parseNode *FiberSnapshot, parsePath []string)
	parseWalk = func(parseNode2 *FiberSnapshot, parsePath2 []string) {
		if parseNode2 == nil {
			return
		}

		parseNextPath := append(append([]string(nil), parsePath2...), parseNode2.Name)
		if parseNode2.Kind != "root" && parseNode2.SubtreeDurationNs > 0 {
			parseBranches = append(parseBranches, HotBranchSnapshot{
				Name:              parseNode2.Name,
				Kind:              parseNode2.Kind,
				Path:              strings.Join(parseNextPath, " > "),
				RenderDurationNs:  parseNode2.RenderDurationNs,
				DiffDurationNs:    parseNode2.DiffDurationNs,
				CommitDurationNs:  parseNode2.CommitDurationNs,
				EffectDurationNs:  parseNode2.EffectDurationNs,
				CleanupDurationNs: parseNode2.CleanupDurationNs,
				SelfDurationNs:    parseNode2.SelfDurationNs,
				SubtreeDurationNs: parseNode2.SubtreeDurationNs,
			})
		}

		for parseIndex := range parseNode2.Children {
			parseWalk(&parseNode2.Children[parseIndex], parseNextPath)
		}
	}

	parseWalk(parseRoot, nil)
	sort.SliceStable(parseBranches, func(parseI, parseJ int) bool {
		if parseBranches[parseI].SubtreeDurationNs == parseBranches[parseJ].SubtreeDurationNs {
			return parseBranches[parseI].SelfDurationNs > parseBranches[parseJ].SelfDurationNs
		}
		return parseBranches[parseI].SubtreeDurationNs > parseBranches[parseJ].SubtreeDurationNs
	})
	if len(parseBranches) > parseLimit {
		parseBranches = parseBranches[:parseLimit]
	}
	return parseBranches
}

// collectComponentRenderTraces is a core package helper.
func collectComponentRenderTraces(parseEntries map[string]*componentRenderTrace, parseLimit int) []ComponentRenderTraceSnapshot {
	if len(parseEntries) == 0 || parseLimit <= 0 {
		return nil
	}

	parseTraces := make([]ComponentRenderTraceSnapshot, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if parseEntry == nil {
			continue
		}
		parseTriggerCounts := make(map[string]int, len(parseEntry.TriggerCounts))
		maps.Copy(parseTriggerCounts, parseEntry.TriggerCounts)
		parseAverage := int64(0)
		if parseEntry.RenderCount > 0 {
			parseAverage = parseEntry.TotalRenderDurationNs / int64(parseEntry.RenderCount)
		}
		parseTraces = append(parseTraces, ComponentRenderTraceSnapshot{
			Name:                    parseEntry.Name,
			Path:                    parseEntry.Path,
			RenderCount:             parseEntry.RenderCount,
			RerenderCount:           parseEntry.RerenderCount,
			LastTrigger:             parseEntry.LastTrigger,
			LastRenderDurationNs:    parseEntry.LastRenderDurationNs,
			TotalRenderDurationNs:   parseEntry.TotalRenderDurationNs,
			AverageRenderDurationNs: parseAverage,
			LastRenderedAt:          parseEntry.LastRenderedAt,
			TriggerCounts:           parseTriggerCounts,
		})
	}
	sort.SliceStable(parseTraces, func(parseI, parseJ int) bool {
		if parseTraces[parseI].RenderCount == parseTraces[parseJ].RenderCount {
			return parseTraces[parseI].TotalRenderDurationNs > parseTraces[parseJ].TotalRenderDurationNs
		}
		return parseTraces[parseI].RenderCount > parseTraces[parseJ].RenderCount
	})
	if len(parseTraces) > parseLimit {
		parseTraces = parseTraces[:parseLimit]
	}
	return parseTraces
}

// buildRouteStartupBudgets builds startup budget snapshots grouped by route family.
func buildRouteStartupBudgets(buildEntries map[string]*routeStartupBudget, buildLimit int) []RouteStartupBudgetSnapshot {
	if len(buildEntries) == 0 || buildLimit <= 0 {
		return nil
	}
	buildSnapshots := make([]RouteStartupBudgetSnapshot, 0, len(buildEntries))
	for _, buildEntry := range buildEntries {
		if buildEntry == nil || strings.TrimSpace(buildEntry.RouteFamily) == "" || buildEntry.SampleCount <= 0 {
			continue
		}
		buildSampleCount := int64(buildEntry.SampleCount)
		buildSnapshots = append(buildSnapshots, RouteStartupBudgetSnapshot{
			RouteFamily:                       buildEntry.RouteFamily,
			LastRoutePath:                     buildEntry.LastRoutePath,
			SampleCount:                       buildEntry.SampleCount,
			AverageBootstrapReadDurationNs:    buildEntry.BootstrapReadDurationTotalNs / buildSampleCount,
			AverageWASMTransferBytes:          buildEntry.WASMTransferBytesTotal / buildSampleCount,
			AverageWASMDecodedBytes:           buildEntry.WASMDecodedBytesTotal / buildSampleCount,
			AverageBootstrapDecodedBytes:      buildEntry.BootstrapDecodedBytesTotal / buildSampleCount,
			AverageCacheWarmupDurationNs:      buildEntry.CacheWarmupDurationTotalNs / buildSampleCount,
			AverageServiceWorkerOverheadNs:    buildEntry.ServiceWorkerOverheadTotalNs / buildSampleCount,
			AverageInitialRouteDataBytes:      buildEntry.InitialRouteDataBytesTotal / buildSampleCount,
			AverageHydrationDurationNs:        buildEntry.HydrationDurationTotalNs / buildSampleCount,
			AverageStartupCommitDurationNs:    buildEntry.StartupCommitDurationTotalNs / buildSampleCount,
			AverageFirstInteractionDurationNs: buildEntry.FirstInteractionDurationTotalNs / buildSampleCount,
		})
	}
	sort.SliceStable(buildSnapshots, func(buildI, buildJ int) bool {
		if buildSnapshots[buildI].AverageFirstInteractionDurationNs == buildSnapshots[buildJ].AverageFirstInteractionDurationNs {
			return buildSnapshots[buildI].RouteFamily < buildSnapshots[buildJ].RouteFamily
		}
		return buildSnapshots[buildI].AverageFirstInteractionDurationNs > buildSnapshots[buildJ].AverageFirstInteractionDurationNs
	})
	if len(buildSnapshots) > buildLimit {
		buildSnapshots = buildSnapshots[:buildLimit]
	}
	return buildSnapshots
}

// inspectHydrationDebugSnapshot is a core package helper.
func inspectHydrationDebugSnapshot(parseMetrics HydrationMetrics, parseDiagnostics []Diagnostic) HydrationDebugSnapshot {
	parseSnapshot := HydrationDebugSnapshot{
		CorrelationID:        parseMetrics.CorrelationID,
		DurationNs:           parseMetrics.DurationNs,
		ExistingDOMNodeCount: parseMetrics.ExistingDOMNodeCount,
		FallbackCount:        parseMetrics.FallbackCount,
		MismatchCount:        parseMetrics.MismatchCount,
		DiscardedNodeCount:   parseMetrics.DiscardedNodeCount,
		Strict:               parseMetrics.Strict,
		Failed:               parseMetrics.Failed,
		Failure:              parseMetrics.Failure,
	}
	if !parseMetrics.StartedAt.IsZero() {
		parseSnapshot.StartedAt = parseMetrics.StartedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	if !parseMetrics.FinishedAt.IsZero() {
		parseSnapshot.FinishedAt = parseMetrics.FinishedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	for _, parseDiagnostic := range parseDiagnostics {
		parseLower := strings.ToLower(parseDiagnostic.Message)
		if strings.Contains(parseLower, "hydration ") {
			parseSnapshot.RecentMessages = append(parseSnapshot.RecentMessages, parseDiagnostic.Message)
		}
	}
	if len(parseSnapshot.RecentMessages) > 5 {
		parseSnapshot.RecentMessages = append([]string(nil), parseSnapshot.RecentMessages[len(parseSnapshot.RecentMessages)-5:]...)
	}
	return parseSnapshot
}

// collectFlamegraphFrames is a core package helper.
func collectFlamegraphFrames(parseRoot *FiberSnapshot, parseLimit int) []FlamegraphFrameSnapshot {
	if parseRoot == nil || parseLimit <= 0 {
		return nil
	}

	parseFrames := make([]FlamegraphFrameSnapshot, 0, parseLimit)
	var parseWalk func(parseNode *FiberSnapshot, parsePath []string, parseDepth int, parseStartNs int64) int64
	parseWalk = func(parseNode2 *FiberSnapshot, parsePath2 []string, parseDepth2 int, parseStartNs2 int64) int64 {
		if parseNode2 == nil {
			return parseStartNs2
		}

		parseNextPath := append(append([]string(nil), parsePath2...), parseNode2.Name)
		parseDurationNs := parseNode2.SubtreeDurationNs

		if parseNode2.Kind != "root" && parseDurationNs > 0 {
			parseFrames = append(parseFrames, FlamegraphFrameSnapshot{
				Name:              parseNode2.Name,
				Kind:              parseNode2.Kind,
				Path:              strings.Join(parseNextPath, " > "),
				Depth:             parseDepth2,
				StartNs:           parseStartNs2,
				DurationNs:        parseDurationNs,
				SelfDurationNs:    parseNode2.SelfDurationNs,
				RenderDurationNs:  parseNode2.RenderDurationNs,
				DiffDurationNs:    parseNode2.DiffDurationNs,
				CommitDurationNs:  parseNode2.CommitDurationNs,
				EffectDurationNs:  parseNode2.EffectDurationNs,
				CleanupDurationNs: parseNode2.CleanupDurationNs,
			})
		}

		parseChildStart := parseStartNs2
		parseNextDepth := parseDepth2
		if parseNode2.Kind != "root" {
			parseNextDepth++
		}
		for parseIndex := range parseNode2.Children {
			parseChildStart = parseWalk(&parseNode2.Children[parseIndex], parseNextPath, parseNextDepth, parseChildStart)
		}

		parseEndNs := parseStartNs2 + parseDurationNs
		if parseChildStart < parseEndNs {
			parseChildStart = parseEndNs
		}
		return parseChildStart
	}

	parseWalk(parseRoot, nil, 0, 0)
	if len(parseFrames) > parseLimit {
		parseFrames = parseFrames[:parseLimit]
	}
	return parseFrames
}

// describeFiber is a core package helper.
func describeFiber(parseFiber *Fiber) (string, string) {
	if parseFiber == nil {
		return "unknown", "unknown"
	}

	switch parseValue := parseFiber.typeOf.(type) {
	case string:
		switch parseValue {
		case "ROOT":
			return "root", "ROOT"
		case "TEXT_ELEMENT":
			return "text", previewValue(parseFiber.textContent)
		default:
			return "host", parseValue
		}
	case *ErrorBoundaryType:
		return "boundary", "ErrorBoundary"
	case *ContextProviderType:
		return "provider", "ContextProvider"
	case *ReactiveTextElementType:
		return "text", "ReactiveText"
	case *ReactiveRegionElementType:
		return "region", "ReactiveRegion"
	case *ComponentType:
		if strings.TrimSpace(parseValue.Name) != "" {
			return "component", parseValue.Name
		}
		if strings.TrimSpace(parseValue.QualifiedName) != "" {
			return "component", parseValue.QualifiedName
		}
		return "component", "Component"
	default:
		parsePrettyName, _ := describeCallableIdentity(parseValue)
		return "component", parsePrettyName
	}
}

// firstNonEmpty is a core package helper.
func firstNonEmpty(parseValues ...string) string {
	for _, parseValue := range parseValues {
		if strings.TrimSpace(parseValue) != "" {
			return parseValue
		}
	}
	return ""
}

// diagnosticComponentStack is a core package helper.
func diagnosticComponentStack(parseFiber *Fiber) []string {
	if parseFiber == nil {
		return nil
	}

	parseStack := make([]string, 0, 8)
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.parent {
		parseKind, parseName := describeFiber(parseCurrent)
		if parseKind == "root" || parseKind == "text" || strings.TrimSpace(parseName) == "" {
			continue
		}
		parseStack = append(parseStack, parseName)
	}
	for parseLeft, parseRight := 0, len(parseStack)-1; parseLeft < parseRight; parseLeft, parseRight = parseLeft+1, parseRight-1 {
		parseStack[parseLeft], parseStack[parseRight] = parseStack[parseRight], parseStack[parseLeft]
	}
	return parseStack
}

// diagnosticPathForFiber is a core package helper.
func diagnosticPathForFiber(parseFiber *Fiber) string {
	parseStack := diagnosticComponentStack(parseFiber)
	if len(parseStack) == 0 {
		return ""
	}
	return strings.Join(parseStack, " > ")
}

// CurrentFiberPath returns the current component path while a hook is rendering.
func CurrentFiberPath() string {
	return diagnosticPathForFiber(GetCurrentFiber())
}

// describeCallable is a core package helper.
func describeCallable(parseValue any) string {
	parsePrettyName, _ := describeCallableIdentity(parseValue)
	return parsePrettyName
}

// inspectHooks is a core package helper.
func inspectHooks(parseHooks *Hooks) []HookSnapshot {
	if parseHooks == nil {
		return nil
	}

	parseResult := make([]HookSnapshot, 0, len(parseHooks.states)/2+len(parseHooks.memos)+len(parseHooks.refs)+len(parseHooks.ids)+len(parseHooks.fetches)+len(parseHooks.atoms)+len(parseHooks.callbacks)+len(parseHooks.deps))
	for parseIndex := 0; parseIndex+1 < len(parseHooks.states); parseIndex += 2 {
		parseResult = append(parseResult, HookSnapshot{Slot: parseIndex / 2, Kind: "state", Value: previewValue(parseHooks.states[parseIndex])})
	}
	for parseIndex2, parseMemo := range parseHooks.memos {
		parseResult = append(parseResult, HookSnapshot{
			Slot:         parseIndex2,
			Kind:         "memo",
			Value:        previewValue(parseMemo.value),
			Dependencies: previewDeps(parseMemo.deps),
		})
	}
	for parseIndex3, parseRef := range parseHooks.refs {
		if parseRef == nil {
			parseResult = append(parseResult, HookSnapshot{Slot: parseIndex3, Kind: "ref", Value: "<nil>"})
			continue
		}
		parseResult = append(parseResult, HookSnapshot{Slot: parseIndex3, Kind: "ref", Value: previewValue(parseRef.Current)})
	}
	for parseIndex4, parseId := range parseHooks.ids {
		parseResult = append(parseResult, HookSnapshot{Slot: parseIndex4, Kind: "id", Value: parseId})
	}
	for parseIndex5, parseAtom := range parseHooks.atoms {
		parseResult = append(parseResult, HookSnapshot{Slot: parseIndex5, Kind: "atom", Value: parseAtom})
	}
	for parseIndex6, parseCallback := range parseHooks.callbacks {
		parseResult = append(parseResult, HookSnapshot{
			Slot:         parseIndex6,
			Kind:         "callback",
			Value:        describeCallable(parseCallback.fn),
			Dependencies: previewDeps(parseCallback.deps),
		})
	}
	for parseIndex7, parseFetch := range parseHooks.fetches {
		parseStatus := "idle"
		switch {
		case parseFetch.state.Loading:
			parseStatus = "loading"
		case parseFetch.state.Error != "":
			parseStatus = "error"
		case parseFetch.state.Data != nil:
			parseStatus = "ready"
		}
		parseResult = append(parseResult, HookSnapshot{
			Slot:   parseIndex7,
			Kind:   "fetch",
			Value:  fmt.Sprintf("url=%q loading=%t error=%q", parseFetch.url, parseFetch.state.Loading, parseFetch.state.Error),
			Status: parseStatus,
		})
	}
	for parseIndex8, parseDeps := range parseHooks.deps {
		parseCleanupStatus := "none"
		if parseIndex8 < len(parseHooks.cleanups) && parseHooks.cleanups[parseIndex8] != nil {
			parseCleanupStatus = "registered"
		}
		parseEpoch := 0
		if parseIndex8 < len(parseHooks.effectEpochs) {
			parseEpoch = parseHooks.effectEpochs[parseIndex8]
		}
		parseResult = append(parseResult, HookSnapshot{
			Slot:         parseIndex8,
			Kind:         "effect",
			Value:        fmt.Sprintf("deps=%d", len(parseDeps)),
			Dependencies: previewDeps(parseDeps),
			Status:       fmt.Sprintf("cleanup=%s epoch=%d", parseCleanupStatus, parseEpoch),
		})
	}

	sort.SliceStable(parseResult, func(parseI, parseJ int) bool {
		if parseResult[parseI].Kind == parseResult[parseJ].Kind {
			return parseResult[parseI].Slot < parseResult[parseJ].Slot
		}
		return parseResult[parseI].Kind < parseResult[parseJ].Kind
	})
	return parseResult
}

// previewDeps is a core package helper.
func previewDeps(parseValues []any) string {
	if len(parseValues) == 0 {
		return ""
	}
	parseParts := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseParts = append(parseParts, previewValue(parseValue))
	}
	return strings.Join(parseParts, ", ")
}

// previewValue is a core package helper.
func previewValue(parseValue any) string {
	if parseValue == nil {
		return "<nil>"
	}

	switch parseTyped := parseValue.(type) {
	case string:
		parseTrimmed := parseTyped
		if len(parseTrimmed) > 48 {
			parseTrimmed = parseTrimmed[:45] + "..."
		}
		return fmt.Sprintf("%q", parseTrimmed)
	case fmt.Stringer:
		return parseTyped.String()
	case error:
		return parseTyped.Error()
	}

	parseRv := reflect.ValueOf(parseValue)
	if !parseRv.IsValid() {
		return "<invalid>"
	}

	switch parseRv.Kind() {
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%s(len=%d)", parseRv.Type(), parseRv.Len())
	case reflect.Map:
		return fmt.Sprintf("%s(len=%d)", parseRv.Type(), parseRv.Len())
	case reflect.Struct:
		return parseRv.Type().String()
	case reflect.Pointer:
		if parseRv.IsNil() {
			return fmt.Sprintf("%s(nil)", parseRv.Type())
		}
		return parseRv.Type().String()
	case reflect.Func:
		return describeCallable(parseValue)
	default:
		return fmt.Sprintf("%v", parseValue)
	}
}
