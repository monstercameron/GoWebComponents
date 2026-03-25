package runtime

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
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
	Name              string
	Path              string
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
	HydrationDurationNs        int64
	StartupCommitDurationNs    int64
	FirstInteractionDurationNs int64
	FirstInteractionCaptured   bool
	FirstInteractionEvent      string
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

var (
	diagnosticsMu   sync.Mutex
	diagnosticIndex = map[string]int{}
	diagnostics     []Diagnostic
	logsMu          sync.Mutex
	logBuffer       []LogEntry
)

const maxLogEntries = 200

// ReportDiagnostic records or increments a runtime diagnostic entry.
func ReportDiagnostic(source string, severity DiagnosticSeverity, message string) {
	ReportDiagnosticWithContext(source, severity, message, "", nil)
}

// ReportDiagnosticWithContext records or increments a runtime diagnostic entry
// and optionally attaches fiber-path context for devtools and debugging.
func ReportDiagnosticWithContext(source string, severity DiagnosticSeverity, message string, path string, componentStack []string) {
	reportDiagnosticWithContextDetails(source, severity, message, path, componentStack, "", "", nil)
}

func reportDiagnosticWithContextDetails(source string, severity DiagnosticSeverity, message string, path string, componentStack []string, topFrame string, consequence string, extraFields map[string]string) {
	trimmedSource := strings.TrimSpace(source)
	if trimmedSource == "" {
		trimmedSource = "runtime"
	}
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return
	}
	trimmedPath := strings.TrimSpace(path)
	stackKey := strings.Join(componentStack, " > ")
	classification := classifyDiagnostic(trimmedSource, severity, trimmedMessage)
	details := diagnosticMetadata(trimmedSource, severity, classification, trimmedMessage)
	fields := diagnosticContextFields(trimmedPath, componentStack, topFrame, consequence, extraFields)
	if shouldEscalateDiagnosticStrictly(trimmedSource, severity, classification, details) {
		escalateStrictDiagnostic(trimmedSource, details, trimmedMessage, trimmedPath, componentStack)
	}

	key := string(severity) + "|" + trimmedSource + "|" + trimmedMessage + "|" + trimmedPath + "|" + stackKey + "|" + diagnosticFieldsKey(fields)

	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	if index, ok := diagnosticIndex[key]; ok {
		diagnostics[index].Count++
		return
	}

	diagnosticIndex[key] = len(diagnostics)
	diagnostics = append(diagnostics, Diagnostic{
		Source:         trimmedSource,
		Severity:       severity,
		Classification: classification,
		Code:           details.Code,
		Docs:           details.Docs,
		Remediation:    details.Remediation,
		Recoverable:    details.Recoverable,
		TopFrame:       strings.TrimSpace(topFrame),
		Consequence:    strings.TrimSpace(consequence),
		Message:        trimmedMessage,
		Count:          1,
		Path:           trimmedPath,
		ComponentStack: append([]string(nil), componentStack...),
		Fields:         cloneLogFields(fields),
	})

	reportDiagnosticLogDetails(trimmedSource, severity, trimmedMessage, trimmedPath, componentStack, topFrame, consequence, fields)
}

// GetDiagnostics returns a copy of the current diagnostic list.
func GetDiagnostics() []Diagnostic {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	clone := make([]Diagnostic, len(diagnostics))
	for index, diagnostic := range diagnostics {
		clone[index] = diagnostic
		clone[index].ComponentStack = append([]string(nil), diagnostic.ComponentStack...)
		clone[index].Fields = cloneLogFields(diagnostic.Fields)
	}
	return clone
}

// ReportLog records one structured framework log entry.
func ReportLog(domain string, level LogLevel, message string) {
	ReportLogWithFields(domain, level, DiagnosticInformational, message, "", nil)
}

// ReportLogWithFields records one structured framework log entry with optional
// correlation id and fields.
func ReportLogWithFields(domain string, level LogLevel, classification DiagnosticClassification, message string, correlationID string, fields map[string]string) {
	reportLogWithFieldsDetails(domain, level, classification, message, correlationID, fields, "", "")
}

func reportLogWithFieldsDetails(domain string, level LogLevel, classification DiagnosticClassification, message string, correlationID string, fields map[string]string, topFrame string, consequence string) {
	trimmedDomain := strings.TrimSpace(domain)
	if trimmedDomain == "" {
		trimmedDomain = "runtime"
	}
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return
	}
	if classification == "" {
		classification = DiagnosticInformational
	}
	if level == "" {
		level = LogInfo
	}
	details := diagnosticMetadata(trimmedDomain, logSeverity(level), classification, trimmedMessage)

	entry := LogEntry{
		Domain:         trimmedDomain,
		Level:          level,
		Classification: classification,
		Code:           details.Code,
		Docs:           details.Docs,
		Remediation:    details.Remediation,
		Recoverable:    details.Recoverable,
		TopFrame:       strings.TrimSpace(topFrame),
		Consequence:    strings.TrimSpace(consequence),
		Message:        trimmedMessage,
		Timestamp:      time.Now().UTC().Format(timeFormatRFC3339Milli),
		CorrelationID:  strings.TrimSpace(correlationID),
		Fields:         cloneLogFields(fields),
	}

	logsMu.Lock()
	defer logsMu.Unlock()
	logBuffer = append(logBuffer, entry)
	if len(logBuffer) > maxLogEntries {
		logBuffer = append([]LogEntry(nil), logBuffer[len(logBuffer)-maxLogEntries:]...)
	}
}

// GetLogs returns a copy of the current in-memory log buffer.
func GetLogs() []LogEntry {
	logsMu.Lock()
	defer logsMu.Unlock()
	clone := make([]LogEntry, len(logBuffer))
	for index, entry := range logBuffer {
		clone[index] = entry
		clone[index].Fields = cloneLogFields(entry.Fields)
	}
	return clone
}

// ClearDiagnostics removes all recorded diagnostics.
func ClearDiagnostics() {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	diagnosticIndex = map[string]int{}
	diagnostics = nil
}

// ClearLogs removes all buffered framework logs.
func ClearLogs() {
	logsMu.Lock()
	defer logsMu.Unlock()
	logBuffer = nil
}

// Inspect captures a snapshot of the current runtime tree, profiling state, and diagnostics.
func (rt *Runtime) Inspect() InspectionSnapshot {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	snapshot := InspectionSnapshot{
		Diagnostics: GetDiagnostics(),
		Logs:        GetLogs(),
	}
	if rt == nil {
		return snapshot
	}

	events := make([]ProfilingEvent, len(rt.profiling.events))
	for index, event := range rt.profiling.events {
		events[index] = event
		events[index].Fields = cloneLogFields(event.Fields)
	}
	var root *FiberSnapshot
	var stats InspectionStats
	if rt.currentRoot != nil {
		root, stats = inspectFiberTree(rt.currentRoot)
	}
	startedAt := ""
	if !rt.profiling.startupStartedAt.IsZero() {
		startedAt = rt.profiling.startupStartedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	snapshot.Root = root
	snapshot.Stats = stats
	snapshot.Profiling = ProfilingSnapshot{
		RenderCalls:                      rt.profiling.renderCalls,
		ScheduledRootUpdates:             rt.profiling.scheduledRootUpdates,
		ScheduledFiberMarks:              rt.profiling.scheduledFiberMarks,
		ScheduledGranularMarks:           rt.profiling.scheduledGranularMarks,
		WorkLoopPasses:                   rt.profiling.workLoopPasses,
		ProcessedUnits:                   rt.profiling.processedUnits,
		CommitCount:                      rt.profiling.commitCount,
		FineGrainedCommits:               rt.profiling.fineGrainedCommits,
		FineGrainedDescendantHostCommits: rt.profiling.fineGrainedDescendantHostCommits,
		FineGrainedDescendantTextCommits: rt.profiling.fineGrainedDescendantTextCommits,
		EffectExecutions:                 rt.profiling.effectExecutions,
		CleanupExecutions:                rt.profiling.cleanupExecutions,
		LastRenderDurationNs:             rt.profiling.lastRenderDurationNs,
		LastCommitDurationNs:             rt.profiling.lastCommitDurationNs,
		LastEffectDurationNs:             rt.profiling.lastEffectDurationNs,
		LastCleanupDurationNs:            rt.profiling.lastCleanupDurationNs,
		PhaseTotals: ProfilingPhaseTotalsSnapshot{
			RenderDurationNs:  rt.profiling.totalRenderDurationNs,
			DiffDurationNs:    rt.profiling.totalDiffDurationNs,
			CommitDurationNs:  rt.profiling.totalCommitDurationNs,
			EffectDurationNs:  rt.profiling.totalEffectDurationNs,
			CleanupDurationNs: rt.profiling.totalCleanupDurationNs,
		},
		ComponentRenders: collectComponentRenderTraces(rt.profiling.componentRenders, 30),
		RecentEvents:     events,
		FlamegraphFrames: collectFlamegraphFrames(root, 256),
		Startup: StartupProfilingSnapshot{
			Mode:                       rt.profiling.startupMode,
			StartedAt:                  startedAt,
			BootstrapReadDurationNs:    rt.profiling.bootstrapReadDurationNs,
			HydrationDurationNs:        rt.profiling.hydrationDurationNs,
			StartupCommitDurationNs:    rt.profiling.startupCommitDurationNs,
			FirstInteractionDurationNs: rt.profiling.firstInteractionDurationNs,
			FirstInteractionCaptured:   rt.profiling.firstInteractionCaptured,
			FirstInteractionEvent:      rt.profiling.firstInteractionEvent,
		},
		HotBranches: collectHotBranches(root, 5),
	}
	snapshot.Hydration = inspectHydrationDebugSnapshot(rt.lastHydrationMetrics, snapshot.Diagnostics)
	return snapshot
}

const timeFormatRFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

func classifyDiagnostic(source string, severity DiagnosticSeverity, message string) DiagnosticClassification {
	trimmed := strings.ToLower(strings.TrimSpace(message))
	switch severity {
	case DiagnosticInfo:
		return DiagnosticInformational
	case DiagnosticError:
		return DiagnosticCorrectness
	case DiagnosticWarning:
		if strings.Contains(trimmed, "slow ") {
			return DiagnosticPerformance
		}
		if strings.Contains(trimmed, "fell back") ||
			strings.Contains(trimmed, "recovered") ||
			strings.Contains(trimmed, "ignoring ") ||
			strings.Contains(trimmed, "discarded unexpected") ||
			(strings.Contains(trimmed, "redirect loop") && source == "router") {
			return DiagnosticUnsupportedRecover
		}
		return DiagnosticCorrectness
	default:
		return DiagnosticInformational
	}
}

func reportDiagnosticLog(source string, severity DiagnosticSeverity, message string, path string, componentStack []string) {
	reportDiagnosticLogDetails(source, severity, message, path, componentStack, "", "", nil)
}

func reportDiagnosticLogDetails(source string, severity DiagnosticSeverity, message string, path string, componentStack []string, topFrame string, consequence string, fields map[string]string) {
	reportLogWithFieldsDetails(
		source,
		logLevelForSeverity(severity),
		classifyDiagnostic(source, severity, message),
		message,
		"",
		fields,
		topFrame,
		consequence,
	)
}

func diagnosticContextFields(path string, componentStack []string, topFrame string, consequence string, extraFields map[string]string) map[string]string {
	fields := cloneLogFields(extraFields)
	if len(fields) == 0 {
		fields = map[string]string{}
	}
	if strings.TrimSpace(path) != "" {
		fields["path"] = strings.TrimSpace(path)
	}
	if len(componentStack) > 0 {
		fields["component_stack"] = strings.Join(componentStack, " > ")
	}
	if strings.TrimSpace(topFrame) != "" {
		fields["top_frame"] = strings.TrimSpace(topFrame)
	}
	if strings.TrimSpace(consequence) != "" {
		fields["runtime"] = strings.TrimSpace(consequence)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func diagnosticFieldsKey(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+fields[key])
	}
	return strings.Join(parts, "|")
}

func logLevelForSeverity(severity DiagnosticSeverity) LogLevel {
	switch severity {
	case DiagnosticError:
		return LogError
	case DiagnosticWarning:
		return LogWarn
	default:
		return LogInfo
	}
}

func cloneLogFields(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	clone := make(map[string]string, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}

func inspectFiberTree(fiber *Fiber) (*FiberSnapshot, InspectionStats) {
	return inspectFiberTreeWithPath(fiber, nil)
}

func inspectFiberTreeWithPath(fiber *Fiber, path []string) (*FiberSnapshot, InspectionStats) {
	if fiber == nil {
		return nil, InspectionStats{}
	}

	kind, name := describeFiber(fiber)
	hooks := inspectHooks(fiber.hooks)
	currentPath := append(append([]string(nil), path...), name)
	node := &FiberSnapshot{
		Name:              name,
		Path:              strings.Join(currentPath, " > "),
		Kind:              kind,
		Dirty:             fiber.dirty,
		NeedsUpdate:       fiber.needsUpdate,
		FineGrained:       fiber.fineGrained,
		ReactiveSource:    firstNonEmpty(strings.Join(fiber.reactiveSourceIDs, ","), fiber.reactiveAtomID),
		UpdateOrigin:      fiber.updateOrigin,
		EffectCount:       len(fiber.effects),
		HookCount:         len(hooks),
		Signature:         buildComponentSignature(fiber, fiber.hooks),
		RenderDurationNs:  fiber.renderDurationNs,
		DiffDurationNs:    fiber.diffDurationNs,
		CommitDurationNs:  fiber.commitDurationNs,
		EffectDurationNs:  fiber.effectDurationNs,
		CleanupDurationNs: fiber.cleanupDurationNs,
		Hooks:             hooks,
	}
	node.SelfDurationNs = node.RenderDurationNs + node.DiffDurationNs + node.CommitDurationNs + node.EffectDurationNs + node.CleanupDurationNs

	stats := InspectionStats{
		TotalFibers: 1,
		HookEntries: len(hooks),
		Effects:     len(fiber.effects),
	}
	if fiber.dirty || fiber.needsUpdate {
		stats.DirtyFibers++
	}

	childSubtreeDurationNs := int64(0)

	switch kind {
	case "component":
		stats.ComponentFibers++
	case "host", "root":
		stats.HostFibers++
	case "text":
		stats.TextFibers++
	}
	if fiber.fineGrained {
		stats.FineGrainedFibers++
	}

	for child := fiber.child; child != nil; child = child.sibling {
		childSnapshot, childStats := inspectFiberTreeWithPath(child, currentPath)
		if childSnapshot != nil {
			node.Children = append(node.Children, *childSnapshot)
			childSubtreeDurationNs += childSnapshot.SubtreeDurationNs
		}
		stats.TotalFibers += childStats.TotalFibers
		stats.DirtyFibers += childStats.DirtyFibers
		stats.ComponentFibers += childStats.ComponentFibers
		stats.HostFibers += childStats.HostFibers
		stats.TextFibers += childStats.TextFibers
		stats.FineGrainedFibers += childStats.FineGrainedFibers
		stats.HookEntries += childStats.HookEntries
		stats.Effects += childStats.Effects
	}
	node.SubtreeDurationNs = node.SelfDurationNs + childSubtreeDurationNs

	return node, stats
}

func collectHotBranches(root *FiberSnapshot, limit int) []HotBranchSnapshot {
	if root == nil || limit <= 0 {
		return nil
	}

	branches := make([]HotBranchSnapshot, 0, limit)
	var walk func(node *FiberSnapshot, path []string)
	walk = func(node *FiberSnapshot, path []string) {
		if node == nil {
			return
		}

		nextPath := append(append([]string(nil), path...), node.Name)
		if node.Kind != "root" && node.SubtreeDurationNs > 0 {
			branches = append(branches, HotBranchSnapshot{
				Name:              node.Name,
				Kind:              node.Kind,
				Path:              strings.Join(nextPath, " > "),
				RenderDurationNs:  node.RenderDurationNs,
				DiffDurationNs:    node.DiffDurationNs,
				CommitDurationNs:  node.CommitDurationNs,
				EffectDurationNs:  node.EffectDurationNs,
				CleanupDurationNs: node.CleanupDurationNs,
				SelfDurationNs:    node.SelfDurationNs,
				SubtreeDurationNs: node.SubtreeDurationNs,
			})
		}

		for index := range node.Children {
			walk(&node.Children[index], nextPath)
		}
	}

	walk(root, nil)
	sort.SliceStable(branches, func(i, j int) bool {
		if branches[i].SubtreeDurationNs == branches[j].SubtreeDurationNs {
			return branches[i].SelfDurationNs > branches[j].SelfDurationNs
		}
		return branches[i].SubtreeDurationNs > branches[j].SubtreeDurationNs
	})
	if len(branches) > limit {
		branches = branches[:limit]
	}
	return branches
}

func collectComponentRenderTraces(entries map[string]*componentRenderTrace, limit int) []ComponentRenderTraceSnapshot {
	if len(entries) == 0 || limit <= 0 {
		return nil
	}

	traces := make([]ComponentRenderTraceSnapshot, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		triggerCounts := make(map[string]int, len(entry.TriggerCounts))
		for trigger, count := range entry.TriggerCounts {
			triggerCounts[trigger] = count
		}
		average := int64(0)
		if entry.RenderCount > 0 {
			average = entry.TotalRenderDurationNs / int64(entry.RenderCount)
		}
		traces = append(traces, ComponentRenderTraceSnapshot{
			Name:                    entry.Name,
			Path:                    entry.Path,
			RenderCount:             entry.RenderCount,
			RerenderCount:           entry.RerenderCount,
			LastTrigger:             entry.LastTrigger,
			LastRenderDurationNs:    entry.LastRenderDurationNs,
			TotalRenderDurationNs:   entry.TotalRenderDurationNs,
			AverageRenderDurationNs: average,
			LastRenderedAt:          entry.LastRenderedAt,
			TriggerCounts:           triggerCounts,
		})
	}
	sort.SliceStable(traces, func(i, j int) bool {
		if traces[i].RenderCount == traces[j].RenderCount {
			return traces[i].TotalRenderDurationNs > traces[j].TotalRenderDurationNs
		}
		return traces[i].RenderCount > traces[j].RenderCount
	})
	if len(traces) > limit {
		traces = traces[:limit]
	}
	return traces
}

func inspectHydrationDebugSnapshot(metrics HydrationMetrics, diagnostics []Diagnostic) HydrationDebugSnapshot {
	snapshot := HydrationDebugSnapshot{
		CorrelationID:        metrics.CorrelationID,
		DurationNs:           metrics.DurationNs,
		ExistingDOMNodeCount: metrics.ExistingDOMNodeCount,
		FallbackCount:        metrics.FallbackCount,
		MismatchCount:        metrics.MismatchCount,
		DiscardedNodeCount:   metrics.DiscardedNodeCount,
		Strict:               metrics.Strict,
		Failed:               metrics.Failed,
		Failure:              metrics.Failure,
	}
	if !metrics.StartedAt.IsZero() {
		snapshot.StartedAt = metrics.StartedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	if !metrics.FinishedAt.IsZero() {
		snapshot.FinishedAt = metrics.FinishedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	for _, diagnostic := range diagnostics {
		lower := strings.ToLower(diagnostic.Message)
		if strings.Contains(lower, "hydration ") {
			snapshot.RecentMessages = append(snapshot.RecentMessages, diagnostic.Message)
		}
	}
	if len(snapshot.RecentMessages) > 5 {
		snapshot.RecentMessages = append([]string(nil), snapshot.RecentMessages[len(snapshot.RecentMessages)-5:]...)
	}
	return snapshot
}

func collectFlamegraphFrames(root *FiberSnapshot, limit int) []FlamegraphFrameSnapshot {
	if root == nil || limit <= 0 {
		return nil
	}

	frames := make([]FlamegraphFrameSnapshot, 0, limit)
	var walk func(node *FiberSnapshot, path []string, depth int, startNs int64) int64
	walk = func(node *FiberSnapshot, path []string, depth int, startNs int64) int64 {
		if node == nil {
			return startNs
		}

		nextPath := append(append([]string(nil), path...), node.Name)
		durationNs := node.SubtreeDurationNs

		if node.Kind != "root" && durationNs > 0 {
			frames = append(frames, FlamegraphFrameSnapshot{
				Name:              node.Name,
				Kind:              node.Kind,
				Path:              strings.Join(nextPath, " > "),
				Depth:             depth,
				StartNs:           startNs,
				DurationNs:        durationNs,
				SelfDurationNs:    node.SelfDurationNs,
				RenderDurationNs:  node.RenderDurationNs,
				DiffDurationNs:    node.DiffDurationNs,
				CommitDurationNs:  node.CommitDurationNs,
				EffectDurationNs:  node.EffectDurationNs,
				CleanupDurationNs: node.CleanupDurationNs,
			})
		}

		childStart := startNs
		nextDepth := depth
		if node.Kind != "root" {
			nextDepth++
		}
		for index := range node.Children {
			childStart = walk(&node.Children[index], nextPath, nextDepth, childStart)
		}

		endNs := startNs + durationNs
		if childStart < endNs {
			childStart = endNs
		}
		return childStart
	}

	walk(root, nil, 0, 0)
	if len(frames) > limit {
		frames = frames[:limit]
	}
	return frames
}

func describeFiber(fiber *Fiber) (string, string) {
	if fiber == nil {
		return "unknown", "unknown"
	}

	switch value := fiber.typeOf.(type) {
	case string:
		switch value {
		case "ROOT":
			return "root", "ROOT"
		case "TEXT_ELEMENT":
			return "text", previewValue(fiber.textContent)
		default:
			return "host", value
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
		if strings.TrimSpace(value.Name) != "" {
			return "component", value.Name
		}
		if strings.TrimSpace(value.QualifiedName) != "" {
			return "component", value.QualifiedName
		}
		return "component", "Component"
	default:
		prettyName, _ := describeCallableIdentity(value)
		return "component", prettyName
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func diagnosticComponentStack(fiber *Fiber) []string {
	if fiber == nil {
		return nil
	}

	stack := make([]string, 0, 8)
	for current := fiber; current != nil; current = current.parent {
		kind, name := describeFiber(current)
		if kind == "root" || kind == "text" || strings.TrimSpace(name) == "" {
			continue
		}
		stack = append(stack, name)
	}
	for left, right := 0, len(stack)-1; left < right; left, right = left+1, right-1 {
		stack[left], stack[right] = stack[right], stack[left]
	}
	return stack
}

func diagnosticPathForFiber(fiber *Fiber) string {
	stack := diagnosticComponentStack(fiber)
	if len(stack) == 0 {
		return ""
	}
	return strings.Join(stack, " > ")
}

// CurrentFiberPath returns the current component path while a hook is rendering.
func CurrentFiberPath() string {
	return diagnosticPathForFiber(GetCurrentFiber())
}

func describeCallable(value interface{}) string {
	prettyName, _ := describeCallableIdentity(value)
	return prettyName
}

func inspectHooks(hooks *Hooks) []HookSnapshot {
	if hooks == nil {
		return nil
	}

	result := make([]HookSnapshot, 0, len(hooks.states)/2+len(hooks.memos)+len(hooks.refs)+len(hooks.ids)+len(hooks.fetches)+len(hooks.atoms)+len(hooks.callbacks)+len(hooks.deps))
	for index := 0; index+1 < len(hooks.states); index += 2 {
		result = append(result, HookSnapshot{Slot: index / 2, Kind: "state", Value: previewValue(hooks.states[index])})
	}
	for index, memo := range hooks.memos {
		result = append(result, HookSnapshot{
			Slot:         index,
			Kind:         "memo",
			Value:        previewValue(memo.value),
			Dependencies: previewDeps(memo.deps),
		})
	}
	for index, ref := range hooks.refs {
		if ref == nil {
			result = append(result, HookSnapshot{Slot: index, Kind: "ref", Value: "<nil>"})
			continue
		}
		result = append(result, HookSnapshot{Slot: index, Kind: "ref", Value: previewValue(ref.Current)})
	}
	for index, id := range hooks.ids {
		result = append(result, HookSnapshot{Slot: index, Kind: "id", Value: id})
	}
	for index, atom := range hooks.atoms {
		result = append(result, HookSnapshot{Slot: index, Kind: "atom", Value: atom})
	}
	for index, callback := range hooks.callbacks {
		result = append(result, HookSnapshot{
			Slot:         index,
			Kind:         "callback",
			Value:        describeCallable(callback.fn),
			Dependencies: previewDeps(callback.deps),
		})
	}
	for index, fetch := range hooks.fetches {
		status := "idle"
		switch {
		case fetch.state.Loading:
			status = "loading"
		case fetch.state.Error != "":
			status = "error"
		case fetch.state.Data != nil:
			status = "ready"
		}
		result = append(result, HookSnapshot{
			Slot:   index,
			Kind:   "fetch",
			Value:  fmt.Sprintf("url=%q loading=%t error=%q", fetch.url, fetch.state.Loading, fetch.state.Error),
			Status: status,
		})
	}
	for index, deps := range hooks.deps {
		cleanupStatus := "none"
		if index < len(hooks.cleanups) && hooks.cleanups[index] != nil {
			cleanupStatus = "registered"
		}
		epoch := 0
		if index < len(hooks.effectEpochs) {
			epoch = hooks.effectEpochs[index]
		}
		result = append(result, HookSnapshot{
			Slot:         index,
			Kind:         "effect",
			Value:        fmt.Sprintf("deps=%d", len(deps)),
			Dependencies: previewDeps(deps),
			Status:       fmt.Sprintf("cleanup=%s epoch=%d", cleanupStatus, epoch),
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Kind == result[j].Kind {
			return result[i].Slot < result[j].Slot
		}
		return result[i].Kind < result[j].Kind
	})
	return result
}

func previewDeps(values []interface{}) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, previewValue(value))
	}
	return strings.Join(parts, ", ")
}

func previewValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}

	switch typed := value.(type) {
	case string:
		trimmed := typed
		if len(trimmed) > 48 {
			trimmed = trimmed[:45] + "..."
		}
		return fmt.Sprintf("%q", trimmed)
	case fmt.Stringer:
		return typed.String()
	case error:
		return typed.Error()
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return "<invalid>"
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%s(len=%d)", rv.Type(), rv.Len())
	case reflect.Map:
		return fmt.Sprintf("%s(len=%d)", rv.Type(), rv.Len())
	case reflect.Struct:
		return rv.Type().String()
	case reflect.Pointer:
		if rv.IsNil() {
			return fmt.Sprintf("%s(nil)", rv.Type())
		}
		return fmt.Sprintf("%s", rv.Type())
	case reflect.Func:
		return describeCallable(value)
	default:
		return fmt.Sprintf("%v", value)
	}
}
