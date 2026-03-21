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
	Kind  string
	Value string
}

// FiberSnapshot captures one inspected fiber subtree.
type FiberSnapshot struct {
	Name              string
	Kind              string
	Dirty             bool
	NeedsUpdate       bool
	FineGrained       bool
	ReactiveSource    string
	UpdateOrigin      string
	EffectCount       int
	HookCount         int
	Signature         *ComponentSignature
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
	CommitDurationNs  int64
	EffectDurationNs  int64
	CleanupDurationNs int64
	SelfDurationNs    int64
	SubtreeDurationNs int64
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
	HotBranches                      []HotBranchSnapshot
}

// InspectionSnapshot is the top-level runtime inspection payload.
type InspectionSnapshot struct {
	Root        *FiberSnapshot
	Stats       InspectionStats
	Profiling   ProfilingSnapshot
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
	reportDiagnosticWithContextDetails(source, severity, message, path, componentStack, "", "")
}

func reportDiagnosticWithContextDetails(source string, severity DiagnosticSeverity, message string, path string, componentStack []string, topFrame string, consequence string) {
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

	key := string(severity) + "|" + trimmedSource + "|" + trimmedMessage + "|" + trimmedPath + "|" + stackKey

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
	})

	reportDiagnosticLogDetails(trimmedSource, severity, trimmedMessage, trimmedPath, componentStack, topFrame, consequence)
}

// GetDiagnostics returns a copy of the current diagnostic list.
func GetDiagnostics() []Diagnostic {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	clone := make([]Diagnostic, len(diagnostics))
	for index, diagnostic := range diagnostics {
		clone[index] = diagnostic
		clone[index].ComponentStack = append([]string(nil), diagnostic.ComponentStack...)
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
	if rt == nil || rt.currentRoot == nil {
		return snapshot
	}

	root, stats := inspectFiberTree(rt.currentRoot)
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
		HotBranches:                      collectHotBranches(root, 5),
	}
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
	reportDiagnosticLogDetails(source, severity, message, path, componentStack, "", "")
}

func reportDiagnosticLogDetails(source string, severity DiagnosticSeverity, message string, path string, componentStack []string, topFrame string, consequence string) {
	fields := map[string]string{}
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
	if fiber == nil {
		return nil, InspectionStats{}
	}

	kind, name := describeFiber(fiber)
	hooks := inspectHooks(fiber.hooks)
	node := &FiberSnapshot{
		Name:              name,
		Kind:              kind,
		Dirty:             fiber.dirty,
		NeedsUpdate:       fiber.needsUpdate,
		FineGrained:       fiber.fineGrained,
		ReactiveSource:    firstNonEmpty(strings.Join(fiber.reactiveSourceIDs, ","), fiber.reactiveAtomID),
		UpdateOrigin:      fiber.updateOrigin,
		EffectCount:       len(fiber.effects),
		HookCount:         len(hooks),
		Signature:         buildComponentSignature(fiber, fiber.hooks),
		CommitDurationNs:  fiber.commitDurationNs,
		EffectDurationNs:  fiber.effectDurationNs,
		CleanupDurationNs: fiber.cleanupDurationNs,
		Hooks:             hooks,
	}
	node.SelfDurationNs = node.CommitDurationNs + node.EffectDurationNs + node.CleanupDurationNs

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
		childSnapshot, childStats := inspectFiberTree(child)
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
		result = append(result, HookSnapshot{Kind: "state", Value: previewValue(hooks.states[index])})
	}
	for _, memo := range hooks.memos {
		result = append(result, HookSnapshot{Kind: "memo", Value: previewValue(memo.value)})
	}
	for _, ref := range hooks.refs {
		if ref == nil {
			result = append(result, HookSnapshot{Kind: "ref", Value: "<nil>"})
			continue
		}
		result = append(result, HookSnapshot{Kind: "ref", Value: previewValue(ref.Current)})
	}
	for _, id := range hooks.ids {
		result = append(result, HookSnapshot{Kind: "id", Value: id})
	}
	for _, atom := range hooks.atoms {
		result = append(result, HookSnapshot{Kind: "atom", Value: atom})
	}
	for _, callback := range hooks.callbacks {
		result = append(result, HookSnapshot{Kind: "callback", Value: describeCallable(callback.fn)})
	}
	for _, fetch := range hooks.fetches {
		result = append(result, HookSnapshot{Kind: "fetch", Value: fmt.Sprintf("url=%q loading=%t error=%q", fetch.url, fetch.state.Loading, fetch.state.Error)})
	}
	for _, deps := range hooks.deps {
		result = append(result, HookSnapshot{Kind: "effect", Value: fmt.Sprintf("deps=%d", len(deps))})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Kind < result[j].Kind
	})
	return result
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
