package runtime

import (
	"strings"
	"time"
)

const maxProfilingEvents = 256

type componentRenderTrace struct {
	Name                  string
	Path                  string
	RenderCount           int
	RerenderCount         int
	LastTrigger           string
	LastRenderDurationNs  int64
	TotalRenderDurationNs int64
	TriggerCounts         map[string]int
	LastRenderedAt        string
}

type routeStartupBudget struct {
	RouteFamily                     string
	SampleCount                     int
	LastRoutePath                   string
	BootstrapReadDurationTotalNs    int64
	HydrationDurationTotalNs        int64
	StartupCommitDurationTotalNs    int64
	FirstInteractionDurationTotalNs int64
	WASMTransferBytesTotal          int64
	WASMDecodedBytesTotal           int64
	BootstrapDecodedBytesTotal      int64
	CacheWarmupDurationTotalNs      int64
	ServiceWorkerOverheadTotalNs    int64
	InitialRouteDataBytesTotal      int64
}

// StartupCostAttribution captures startup-cost components for startup reporting.
type StartupCostAttribution struct {
	WASMTransferBytes       int64
	WASMDecodedBytes        int64
	BootstrapDecodedBytes   int64
	CacheWarmupDurationNs   int64
	ServiceWorkerOverheadNs int64
	InitialRouteDataBytes   int64
}

// ProfilingEvent captures one structured profiling timeline entry.
type ProfilingEvent struct {
	Domain        string
	Name          string
	Phase         string
	Target        string
	CorrelationID string
	DurationNs    int64
	Timestamp     string
	Fields        map[string]string
}

// ReportProfilingEvent records a profiling timeline entry on the global runtime.
func ReportProfilingEvent(parseDomain string, parseName string, parsePhase string, parseTarget string, parseDurationNs int64, parseFields map[string]string) {
	GetGlobalRuntime().RecordProfilingEvent(ProfilingEvent{
		Domain:     parseDomain,
		Name:       parseName,
		Phase:      parsePhase,
		Target:     parseTarget,
		DurationNs: parseDurationNs,
		Fields:     parseFields,
	})
}

// RecordProfilingEvent records a profiling timeline entry on this runtime.
func (parseRt *Runtime) RecordProfilingEvent(parseEvent ProfilingEvent) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.recordProfilingEventLocked(parseEvent)
}

// recordProfilingEventLocked is a core package helper.
func (parseRt *Runtime) recordProfilingEventLocked(parseEvent ProfilingEvent) {
	if parseRt == nil {
		return
	}
	parseTrimmedDomain := strings.TrimSpace(parseEvent.Domain)
	if parseTrimmedDomain == "" {
		parseTrimmedDomain = "runtime"
	}
	parseTrimmedName := strings.TrimSpace(parseEvent.Name)
	if parseTrimmedName == "" {
		parseTrimmedName = "event"
	}
	parseTrimmedPhase := strings.TrimSpace(parseEvent.Phase)
	if parseTrimmedPhase == "" {
		parseTrimmedPhase = "instant"
	}
	if strings.TrimSpace(parseEvent.Timestamp) == "" {
		parseEvent.Timestamp = time.Now().UTC().Format(timeFormatRFC3339Milli)
	}
	parseEvent.Domain = parseTrimmedDomain
	parseEvent.Name = parseTrimmedName
	parseEvent.Phase = parseTrimmedPhase
	parseEvent.Target = strings.TrimSpace(parseEvent.Target)
	parseEvent.CorrelationID = strings.TrimSpace(parseEvent.CorrelationID)
	parseEvent.Fields = cloneLogFields(parseEvent.Fields)

	parseRt.profiling.events = append(parseRt.profiling.events, parseEvent)
	if len(parseRt.profiling.events) > maxProfilingEvents {
		// Trim in-place: shift the tail down without allocating a new backing array.
		parseTail := parseRt.profiling.events[len(parseRt.profiling.events)-maxProfilingEvents:]
		parseRt.profiling.events = parseRt.profiling.events[:maxProfilingEvents]
		copy(parseRt.profiling.events, parseTail)
	}
}

// recordComponentRenderTrace is a core package helper.
func (parseRt *Runtime) recordComponentRenderTrace(parseFiber *Fiber, parseDurationNs int64) {
	if parseRt == nil || parseFiber == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.recordComponentRenderTraceLocked(parseFiber, parseDurationNs)
}

// recordComponentRenderTraceLocked is a core package helper.
func (parseRt *Runtime) recordComponentRenderTraceLocked(parseFiber *Fiber, parseDurationNs int64) {
	if parseRt == nil || parseFiber == nil {
		return
	}
	parseKind, parseName := describeFiber(parseFiber)
	if parseKind != "component" {
		return
	}
	parsePath := diagnosticPathForFiber(parseFiber)
	parseKey := parsePath
	if parseKey == "" {
		parseKey = parseName
	}
	if parseKey == "" {
		parseKey = "component"
	}
	if parseRt.profiling.componentRenders == nil {
		parseRt.profiling.componentRenders = make(map[string]*componentRenderTrace, 64)
	}
	parseTrace := parseRt.profiling.componentRenders[parseKey]
	if parseTrace == nil {
		parseTrace = &componentRenderTrace{
			Name:          parseName,
			Path:          parsePath,
			TriggerCounts: make(map[string]int, 4),
		}
		parseRt.profiling.componentRenders[parseKey] = parseTrace
	}
	parseTrigger := componentRenderTrigger(parseFiber)
	parseTrace.Name = parseName
	parseTrace.Path = parsePath
	parseTrace.RenderCount++
	if parseFiber.alternate != nil {
		parseTrace.RerenderCount++
	}
	parseTrace.LastTrigger = parseTrigger
	parseTrace.LastRenderDurationNs = parseDurationNs
	parseTrace.TotalRenderDurationNs += parseDurationNs
	parseTrace.LastRenderedAt = time.Now().UTC().Format(timeFormatRFC3339Milli)
	parseTrace.TriggerCounts[parseTrigger]++
	parseRt.profiling.totalRenderDurationNs += parseDurationNs
}

// ClearProfiling resets collected profiling counters and timeline events.
func ClearProfiling() {
	GetGlobalRuntime().ClearProfiling()
}

// ClearProfiling resets collected profiling counters and timeline events.
func (parseRt *Runtime) ClearProfiling() {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.profiling = runtimeProfiling{}
}

// componentRenderTrigger is a core package helper.
func componentRenderTrigger(parseFiber *Fiber) string {
	if parseFiber == nil {
		return "unknown"
	}
	if parseFiber.alternate == nil {
		return "mount"
	}
	if parseTrigger := strings.TrimSpace(parseFiber.updateOrigin); parseTrigger != "" {
		return parseTrigger
	}
	if parseFiber.alternate != nil && !propsEqual(parseFiber.alternate.props, parseFiber.props) {
		return "props"
	}
	return "parent"
}

// BeginStartupProfiling marks the beginning of a startup workflow when one has
// not already been started for the current runtime.
func BeginStartupProfiling(parseMode string) {
	GetGlobalRuntime().BeginStartupProfiling(parseMode)
}

// BeginStartupProfiling marks the beginning of a startup workflow when one has
// not already been started for the current runtime.
func (parseRt *Runtime) BeginStartupProfiling(parseMode string) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	parseRt.beginStartupProfilingLocked(parseMode)
}

// beginStartupProfilingLocked is a core package helper.
func (parseRt *Runtime) beginStartupProfilingLocked(parseMode string) {
	if parseRt == nil {
		return
	}
	if parseRt.currentRoot != nil || !parseRt.profiling.startupStartedAt.IsZero() {
		return
	}
	parseRt.profiling.startupMode = strings.TrimSpace(parseMode)
	if parseRt.profiling.startupMode == "" {
		parseRt.profiling.startupMode = "render"
	}
	parseRt.profiling.startupStartedAt = time.Now().UTC()
	parseRt.profiling.bootstrapReadDurationNs = 0
	parseRt.profiling.startupWASMTransferBytes = 0
	parseRt.profiling.startupWASMDecodedBytes = 0
	parseRt.profiling.startupBootstrapDecodedBytes = 0
	parseRt.profiling.startupCacheWarmupDurationNs = 0
	parseRt.profiling.startupServiceWorkerOverheadNs = 0
	parseRt.profiling.startupInitialRouteDataBytes = 0
	parseRt.profiling.hydrationDurationNs = 0
	parseRt.profiling.startupCommitDurationNs = 0
	parseRt.profiling.firstInteractionDurationNs = 0
	parseRt.profiling.firstInteractionCaptured = false
	parseRt.profiling.firstInteractionEvent = ""
	parseRt.profiling.startupRoutePath = ""
	parseRt.profiling.startupRouteFamily = ""
	parseRt.recordProfilingEventLocked(ProfilingEvent{
		Domain: "runtime",
		Name:   "startup",
		Phase:  "start",
		Target: parseRt.profiling.startupMode,
	})
}

// RecordStartupBootstrapRead stores bootstrap read or decode timing and emits a
// profiling timeline event.
func RecordStartupBootstrapRead(parseDurationNs int64, parseSource string) {
	GetGlobalRuntime().RecordStartupBootstrapRead(parseDurationNs, parseSource)
}

// RecordStartupBootstrapRead stores bootstrap read or decode timing and emits a
// profiling timeline event.
func (parseRt *Runtime) RecordStartupBootstrapRead(parseDurationNs int64, parseSource string) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if parseDurationNs < 0 {
		parseDurationNs = 0
	}
	parseRt.profiling.bootstrapReadDurationNs = parseDurationNs
	parseRt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "startup.bootstrap",
		Phase:      "finish",
		Target:     strings.TrimSpace(parseSource),
		DurationNs: parseDurationNs,
	})
}

// StoreStartupCostAttribution stores startup-cost attribution values.
func StoreStartupCostAttribution(storeAttribution StartupCostAttribution) {
	GetGlobalRuntime().StoreStartupCostAttribution(storeAttribution)
}

// StoreStartupCostAttribution stores startup-cost attribution values.
func (parseRt *Runtime) StoreStartupCostAttribution(storeAttribution StartupCostAttribution) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if parseRt.profiling.startupStartedAt.IsZero() {
		return
	}
	if storeAttribution.WASMTransferBytes > 0 {
		parseRt.profiling.startupWASMTransferBytes = storeAttribution.WASMTransferBytes
	}
	if storeAttribution.WASMDecodedBytes > 0 {
		parseRt.profiling.startupWASMDecodedBytes = storeAttribution.WASMDecodedBytes
	}
	if storeAttribution.BootstrapDecodedBytes > 0 {
		parseRt.profiling.startupBootstrapDecodedBytes = storeAttribution.BootstrapDecodedBytes
	}
	if storeAttribution.CacheWarmupDurationNs > 0 {
		parseRt.profiling.startupCacheWarmupDurationNs = storeAttribution.CacheWarmupDurationNs
	}
	if storeAttribution.ServiceWorkerOverheadNs > 0 {
		parseRt.profiling.startupServiceWorkerOverheadNs = storeAttribution.ServiceWorkerOverheadNs
	}
	if storeAttribution.InitialRouteDataBytes > 0 {
		parseRt.profiling.startupInitialRouteDataBytes = storeAttribution.InitialRouteDataBytes
	}
}

// recordFirstInteraction is a core package helper.
func (parseRt *Runtime) recordFirstInteraction(parseEventKind string) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if parseRt.profiling.startupStartedAt.IsZero() || parseRt.profiling.firstInteractionCaptured {
		return
	}
	parseDurationNs := time.Since(parseRt.profiling.startupStartedAt).Nanoseconds()
	if parseDurationNs < 0 {
		parseDurationNs = 0
	}
	parseRt.profiling.firstInteractionCaptured = true
	parseRt.profiling.firstInteractionDurationNs = parseDurationNs
	parseRt.profiling.firstInteractionEvent = strings.TrimSpace(parseEventKind)
	if parseRt.profiling.startupRouteFamily != "" {
		parseRt.recordRouteStartupBudgetLocked(parseRt.profiling.startupRouteFamily, parseRt.profiling.startupRoutePath)
	}
	parseRt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "startup.first_interaction",
		Phase:      "finish",
		Target:     parseRt.profiling.firstInteractionEvent,
		DurationNs: parseDurationNs,
	})
}

// RecordStartupRouteContext captures which route family the active startup flow
// is targeting so first-interaction budgets can be attributed correctly.
func RecordStartupRouteContext(parseRoutePath string) {
	GetGlobalRuntime().RecordStartupRouteContext(parseRoutePath)
}

// RecordStartupRouteContext captures which route family the active startup flow
// is targeting so first-interaction budgets can be attributed correctly.
func (parseRt *Runtime) RecordStartupRouteContext(parseRoutePath string) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if parseRt.profiling.startupStartedAt.IsZero() {
		return
	}
	parseTrimmed := normalizeRoutePathForBudget(parseRoutePath)
	if parseTrimmed == "" {
		parseTrimmed = "/"
	}
	parseRt.profiling.startupRoutePath = parseTrimmed
	parseRt.profiling.startupRouteFamily = routeFamilyForPath(parseTrimmed)
}

// recordRouteStartupBudgetLocked is a core package helper.
func (parseRt *Runtime) recordRouteStartupBudgetLocked(parseRouteFamily string, parseRoutePath string) {
	if parseRt == nil || strings.TrimSpace(parseRouteFamily) == "" {
		return
	}
	if parseRt.profiling.routeStartupBudgets == nil {
		parseRt.profiling.routeStartupBudgets = make(map[string]*routeStartupBudget, 8)
	}
	parseEntry := parseRt.profiling.routeStartupBudgets[parseRouteFamily]
	if parseEntry == nil {
		parseEntry = &routeStartupBudget{RouteFamily: parseRouteFamily}
		parseRt.profiling.routeStartupBudgets[parseRouteFamily] = parseEntry
	}
	parseEntry.SampleCount++
	parseEntry.LastRoutePath = parseRoutePath
	parseEntry.BootstrapReadDurationTotalNs += parseRt.profiling.bootstrapReadDurationNs
	parseEntry.HydrationDurationTotalNs += parseRt.profiling.hydrationDurationNs
	parseEntry.StartupCommitDurationTotalNs += parseRt.profiling.startupCommitDurationNs
	parseEntry.FirstInteractionDurationTotalNs += parseRt.profiling.firstInteractionDurationNs
	parseEntry.WASMTransferBytesTotal += parseRt.profiling.startupWASMTransferBytes
	parseEntry.WASMDecodedBytesTotal += parseRt.profiling.startupWASMDecodedBytes
	parseEntry.BootstrapDecodedBytesTotal += parseRt.profiling.startupBootstrapDecodedBytes
	parseEntry.CacheWarmupDurationTotalNs += parseRt.profiling.startupCacheWarmupDurationNs
	parseEntry.ServiceWorkerOverheadTotalNs += parseRt.profiling.startupServiceWorkerOverheadNs
	parseEntry.InitialRouteDataBytesTotal += parseRt.profiling.startupInitialRouteDataBytes
}

// normalizeRoutePathForBudget is a core package helper.
func normalizeRoutePathForBudget(parseRaw string) string {
	parsePath := strings.TrimSpace(parseRaw)
	if parsePath == "" {
		return "/"
	}
	if parseIdx := strings.Index(parsePath, "?"); parseIdx >= 0 {
		parsePath = parsePath[:parseIdx]
	}
	if parseIdx2 := strings.Index(parsePath, "#"); parseIdx2 >= 0 {
		parsePath = parsePath[:parseIdx2]
	}
	if parsePath == "" {
		return "/"
	}
	if !strings.HasPrefix(parsePath, "/") {
		parsePath = "/" + parsePath
	}
	return parsePath
}

// routeFamilyForPath is a core package helper.
func routeFamilyForPath(parsePath string) string {
	parseTrimmed := strings.Trim(parsePath, "/")
	if parseTrimmed == "" {
		return "/"
	}
	parseParts := strings.Split(parseTrimmed, "/")
	if len(parseParts) == 1 {
		return "/" + parseParts[0]
	}
	return "/" + parseParts[0] + "/*"
}
