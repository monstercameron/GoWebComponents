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
func ReportProfilingEvent(domain string, name string, phase string, target string, durationNs int64, fields map[string]string) {
	GetGlobalRuntime().RecordProfilingEvent(ProfilingEvent{
		Domain:     domain,
		Name:       name,
		Phase:      phase,
		Target:     target,
		DurationNs: durationNs,
		Fields:     fields,
	})
}

// RecordProfilingEvent records a profiling timeline entry on this runtime.
func (rt *Runtime) RecordProfilingEvent(event ProfilingEvent) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	rt.recordProfilingEventLocked(event)
}

func (rt *Runtime) recordProfilingEventLocked(event ProfilingEvent) {
	if rt == nil {
		return
	}
	trimmedDomain := strings.TrimSpace(event.Domain)
	if trimmedDomain == "" {
		trimmedDomain = "runtime"
	}
	trimmedName := strings.TrimSpace(event.Name)
	if trimmedName == "" {
		trimmedName = "event"
	}
	trimmedPhase := strings.TrimSpace(event.Phase)
	if trimmedPhase == "" {
		trimmedPhase = "instant"
	}
	if strings.TrimSpace(event.Timestamp) == "" {
		event.Timestamp = time.Now().UTC().Format(timeFormatRFC3339Milli)
	}
	event.Domain = trimmedDomain
	event.Name = trimmedName
	event.Phase = trimmedPhase
	event.Target = strings.TrimSpace(event.Target)
	event.CorrelationID = strings.TrimSpace(event.CorrelationID)
	event.Fields = cloneLogFields(event.Fields)

	rt.profiling.events = append(rt.profiling.events, event)
	if len(rt.profiling.events) > maxProfilingEvents {
		rt.profiling.events = append([]ProfilingEvent(nil), rt.profiling.events[len(rt.profiling.events)-maxProfilingEvents:]...)
	}
}

func (rt *Runtime) recordComponentRenderTrace(fiber *Fiber, durationNs int64) {
	if rt == nil || fiber == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	rt.recordComponentRenderTraceLocked(fiber, durationNs)
}

func (rt *Runtime) recordComponentRenderTraceLocked(fiber *Fiber, durationNs int64) {
	if rt == nil || fiber == nil {
		return
	}
	kind, name := describeFiber(fiber)
	if kind != "component" {
		return
	}
	path := diagnosticPathForFiber(fiber)
	key := path
	if key == "" {
		key = name
	}
	if key == "" {
		key = "component"
	}
	if rt.profiling.componentRenders == nil {
		rt.profiling.componentRenders = make(map[string]*componentRenderTrace, 64)
	}
	trace := rt.profiling.componentRenders[key]
	if trace == nil {
		trace = &componentRenderTrace{
			Name:          name,
			Path:          path,
			TriggerCounts: make(map[string]int, 4),
		}
		rt.profiling.componentRenders[key] = trace
	}
	trigger := componentRenderTrigger(fiber)
	trace.Name = name
	trace.Path = path
	trace.RenderCount++
	if fiber.alternate != nil {
		trace.RerenderCount++
	}
	trace.LastTrigger = trigger
	trace.LastRenderDurationNs = durationNs
	trace.TotalRenderDurationNs += durationNs
	trace.LastRenderedAt = time.Now().UTC().Format(timeFormatRFC3339Milli)
	trace.TriggerCounts[trigger]++
	rt.profiling.totalRenderDurationNs += durationNs
}

// ClearProfiling resets collected profiling counters and timeline events.
func ClearProfiling() {
	GetGlobalRuntime().ClearProfiling()
}

// ClearProfiling resets collected profiling counters and timeline events.
func (rt *Runtime) ClearProfiling() {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	rt.profiling = runtimeProfiling{}
}

func componentRenderTrigger(fiber *Fiber) string {
	if fiber == nil {
		return "unknown"
	}
	if fiber.alternate == nil {
		return "mount"
	}
	if trigger := strings.TrimSpace(fiber.updateOrigin); trigger != "" {
		return trigger
	}
	if fiber.alternate != nil && !propsEqual(fiber.alternate.props, fiber.props) {
		return "props"
	}
	return "parent"
}

// BeginStartupProfiling marks the beginning of a startup workflow when one has
// not already been started for the current runtime.
func BeginStartupProfiling(mode string) {
	GetGlobalRuntime().BeginStartupProfiling(mode)
}

// BeginStartupProfiling marks the beginning of a startup workflow when one has
// not already been started for the current runtime.
func (rt *Runtime) BeginStartupProfiling(mode string) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	rt.beginStartupProfilingLocked(mode)
}

func (rt *Runtime) beginStartupProfilingLocked(mode string) {
	if rt == nil {
		return
	}
	if rt.currentRoot != nil || !rt.profiling.startupStartedAt.IsZero() {
		return
	}
	rt.profiling.startupMode = strings.TrimSpace(mode)
	if rt.profiling.startupMode == "" {
		rt.profiling.startupMode = "render"
	}
	rt.profiling.startupStartedAt = time.Now().UTC()
	rt.profiling.bootstrapReadDurationNs = 0
	rt.profiling.startupWASMTransferBytes = 0
	rt.profiling.startupWASMDecodedBytes = 0
	rt.profiling.startupBootstrapDecodedBytes = 0
	rt.profiling.startupCacheWarmupDurationNs = 0
	rt.profiling.startupServiceWorkerOverheadNs = 0
	rt.profiling.startupInitialRouteDataBytes = 0
	rt.profiling.hydrationDurationNs = 0
	rt.profiling.startupCommitDurationNs = 0
	rt.profiling.firstInteractionDurationNs = 0
	rt.profiling.firstInteractionCaptured = false
	rt.profiling.firstInteractionEvent = ""
	rt.profiling.startupRoutePath = ""
	rt.profiling.startupRouteFamily = ""
	rt.recordProfilingEventLocked(ProfilingEvent{
		Domain: "runtime",
		Name:   "startup",
		Phase:  "start",
		Target: rt.profiling.startupMode,
	})
}

// RecordStartupBootstrapRead stores bootstrap read or decode timing and emits a
// profiling timeline event.
func RecordStartupBootstrapRead(durationNs int64, source string) {
	GetGlobalRuntime().RecordStartupBootstrapRead(durationNs, source)
}

// RecordStartupBootstrapRead stores bootstrap read or decode timing and emits a
// profiling timeline event.
func (rt *Runtime) RecordStartupBootstrapRead(durationNs int64, source string) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if durationNs < 0 {
		durationNs = 0
	}
	rt.profiling.bootstrapReadDurationNs = durationNs
	rt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "startup.bootstrap",
		Phase:      "finish",
		Target:     strings.TrimSpace(source),
		DurationNs: durationNs,
	})
}

// StoreStartupCostAttribution stores startup-cost attribution values.
func StoreStartupCostAttribution(storeAttribution StartupCostAttribution) {
	GetGlobalRuntime().StoreStartupCostAttribution(storeAttribution)
}

// StoreStartupCostAttribution stores startup-cost attribution values.
func (rt *Runtime) StoreStartupCostAttribution(storeAttribution StartupCostAttribution) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if rt.profiling.startupStartedAt.IsZero() {
		return
	}
	if storeAttribution.WASMTransferBytes > 0 {
		rt.profiling.startupWASMTransferBytes = storeAttribution.WASMTransferBytes
	}
	if storeAttribution.WASMDecodedBytes > 0 {
		rt.profiling.startupWASMDecodedBytes = storeAttribution.WASMDecodedBytes
	}
	if storeAttribution.BootstrapDecodedBytes > 0 {
		rt.profiling.startupBootstrapDecodedBytes = storeAttribution.BootstrapDecodedBytes
	}
	if storeAttribution.CacheWarmupDurationNs > 0 {
		rt.profiling.startupCacheWarmupDurationNs = storeAttribution.CacheWarmupDurationNs
	}
	if storeAttribution.ServiceWorkerOverheadNs > 0 {
		rt.profiling.startupServiceWorkerOverheadNs = storeAttribution.ServiceWorkerOverheadNs
	}
	if storeAttribution.InitialRouteDataBytes > 0 {
		rt.profiling.startupInitialRouteDataBytes = storeAttribution.InitialRouteDataBytes
	}
}

func (rt *Runtime) recordFirstInteraction(eventKind string) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if rt.profiling.startupStartedAt.IsZero() || rt.profiling.firstInteractionCaptured {
		return
	}
	durationNs := time.Since(rt.profiling.startupStartedAt).Nanoseconds()
	if durationNs < 0 {
		durationNs = 0
	}
	rt.profiling.firstInteractionCaptured = true
	rt.profiling.firstInteractionDurationNs = durationNs
	rt.profiling.firstInteractionEvent = strings.TrimSpace(eventKind)
	if rt.profiling.startupRouteFamily != "" {
		rt.recordRouteStartupBudgetLocked(rt.profiling.startupRouteFamily, rt.profiling.startupRoutePath)
	}
	rt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "startup.first_interaction",
		Phase:      "finish",
		Target:     rt.profiling.firstInteractionEvent,
		DurationNs: durationNs,
	})
}

// RecordStartupRouteContext captures which route family the active startup flow
// is targeting so first-interaction budgets can be attributed correctly.
func RecordStartupRouteContext(routePath string) {
	GetGlobalRuntime().RecordStartupRouteContext(routePath)
}

// RecordStartupRouteContext captures which route family the active startup flow
// is targeting so first-interaction budgets can be attributed correctly.
func (rt *Runtime) RecordStartupRouteContext(routePath string) {
	if rt == nil {
		return
	}
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if rt.profiling.startupStartedAt.IsZero() {
		return
	}
	trimmed := normalizeRoutePathForBudget(routePath)
	if trimmed == "" {
		trimmed = "/"
	}
	rt.profiling.startupRoutePath = trimmed
	rt.profiling.startupRouteFamily = routeFamilyForPath(trimmed)
}

func (rt *Runtime) recordRouteStartupBudgetLocked(routeFamily string, routePath string) {
	if rt == nil || strings.TrimSpace(routeFamily) == "" {
		return
	}
	if rt.profiling.routeStartupBudgets == nil {
		rt.profiling.routeStartupBudgets = make(map[string]*routeStartupBudget, 8)
	}
	entry := rt.profiling.routeStartupBudgets[routeFamily]
	if entry == nil {
		entry = &routeStartupBudget{RouteFamily: routeFamily}
		rt.profiling.routeStartupBudgets[routeFamily] = entry
	}
	entry.SampleCount++
	entry.LastRoutePath = routePath
	entry.BootstrapReadDurationTotalNs += rt.profiling.bootstrapReadDurationNs
	entry.HydrationDurationTotalNs += rt.profiling.hydrationDurationNs
	entry.StartupCommitDurationTotalNs += rt.profiling.startupCommitDurationNs
	entry.FirstInteractionDurationTotalNs += rt.profiling.firstInteractionDurationNs
	entry.WASMTransferBytesTotal += rt.profiling.startupWASMTransferBytes
	entry.WASMDecodedBytesTotal += rt.profiling.startupWASMDecodedBytes
	entry.BootstrapDecodedBytesTotal += rt.profiling.startupBootstrapDecodedBytes
	entry.CacheWarmupDurationTotalNs += rt.profiling.startupCacheWarmupDurationNs
	entry.ServiceWorkerOverheadTotalNs += rt.profiling.startupServiceWorkerOverheadNs
	entry.InitialRouteDataBytesTotal += rt.profiling.startupInitialRouteDataBytes
}

func normalizeRoutePathForBudget(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "/"
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func routeFamilyForPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "/"
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		return "/" + parts[0]
	}
	return "/" + parts[0] + "/*"
}
