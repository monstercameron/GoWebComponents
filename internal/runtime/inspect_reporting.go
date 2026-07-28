package runtime

import (
	"maps"
	"sort"
	"strings"
	"sync"
	"time"
)

// diagnosticDedupKey identifies one diagnostic for repeat-count coalescing.
// A comparable struct key avoids concatenating the fields into one large
// string per report.
type diagnosticDedupKey struct {
	severity    DiagnosticSeverity
	source      string
	message     string
	path        string
	stack       string
	topFrame    string
	consequence string
	fields      string
}

var (
	diagnosticsMu   sync.Mutex
	diagnosticIndex = map[diagnosticDedupKey]int{}
	diagnostics     []Diagnostic
	logsMu          sync.Mutex
	// logBuffer is a circular buffer once it reaches maxLogEntries: logHead is
	// the index of the oldest entry and inserts overwrite in place, so steady-
	// state logging never reallocates. GetLogs linearizes oldest-first.
	logBuffer []LogEntry
	logHead   int
)

const defaultMaxLogEntries = 200

// maxLogEntries bounds the log ring. A var for the same reason as
// maxProfilingEvents: RuntimeLimits.MaxLogEntries was a public field nothing
// read. Written only under logsMu, which guards the buffer.
var maxLogEntries = defaultMaxLogEntries

const defaultMaxDiagnosticEntries = 500

var maxDiagnosticEntries = defaultMaxDiagnosticEntries

// ReportDiagnostic records or increments a runtime diagnostic entry.
func ReportDiagnostic(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string) {
	ReportDiagnosticWithContext(parseSource, parseSeverity, parseMessage, "", nil)
}

// ReportDiagnosticWithContext records or increments a runtime diagnostic entry
// and optionally attaches fiber-path context for devtools and debugging.
func ReportDiagnosticWithContext(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string, parsePath string, parseComponentStack []string) {
	reportDiagnosticWithContextDetails(parseSource, parseSeverity, parseMessage, parsePath, parseComponentStack, "", "", nil)
}

// reportDiagnosticWithContextDetails is a core package helper.
func reportDiagnosticWithContextDetails(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string, parsePath string, parseComponentStack []string, parseTopFrame string, parseConsequence string, parseExtraFields map[string]string) {
	parseTrimmedSource := strings.TrimSpace(parseSource)
	if parseTrimmedSource == "" {
		parseTrimmedSource = "runtime"
	}
	parseTrimmedMessage := strings.TrimSpace(parseMessage)
	if parseTrimmedMessage == "" {
		return
	}
	parseTrimmedPath := strings.TrimSpace(parsePath)
	parseStackKey := strings.Join(parseComponentStack, " > ")
	parseTrimmedTopFrame := strings.TrimSpace(parseTopFrame)
	parseTrimmedConsequence := strings.TrimSpace(parseConsequence)

	// Strict escalation must run on every report (including dedup repeats),
	// but strict mode is off by default; classification and metadata are
	// otherwise only needed when a new entry is stored, so repeats skip them.
	parseClassification := DiagnosticClassification("")
	parseDetails := diagnosticDetails{}
	hasParseDetails := false
	if strictDiagnosticsEnabled() {
		parseClassification = classifyDiagnostic(parseTrimmedSource, parseSeverity, parseTrimmedMessage)
		parseDetails = diagnosticMetadata(parseTrimmedSource, parseSeverity, parseClassification, parseTrimmedMessage)
		hasParseDetails = true
		if shouldEscalateDiagnosticStrictly(parseTrimmedSource, parseSeverity, parseClassification, parseDetails) {
			escalateStrictDiagnostic(parseTrimmedSource, parseDetails, parseTrimmedMessage, parseTrimmedPath, parseComponentStack)
		}
	}

	// Dedup-first: the key is derived from the same inputs the context-fields
	// map is built from, so repeat diagnostics (e.g. a missing-key warning
	// firing every render) increment a counter without paying the field-map,
	// clone, and log construction below.
	parseKey := diagnosticDedupKey{
		severity:    parseSeverity,
		source:      parseTrimmedSource,
		message:     parseTrimmedMessage,
		path:        parseTrimmedPath,
		stack:       parseStackKey,
		topFrame:    parseTrimmedTopFrame,
		consequence: parseTrimmedConsequence,
		fields:      diagnosticFieldsKey(parseExtraFields),
	}

	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	if parseIndex, parseOk := diagnosticIndex[parseKey]; parseOk {
		diagnostics[parseIndex].Count++
		return
	}
	if !hasParseDetails {
		parseClassification = classifyDiagnostic(parseTrimmedSource, parseSeverity, parseTrimmedMessage)
		parseDetails = diagnosticMetadata(parseTrimmedSource, parseSeverity, parseClassification, parseTrimmedMessage)
	}
	// diagnosticContextFields builds a fresh map, so the entry takes ownership
	// of it directly; the log entry below clones its own copy.
	parseFields := diagnosticContextFields(parseTrimmedPath, parseComponentStack, parseTopFrame, parseConsequence, parseExtraFields)

	diagnosticIndex[parseKey] = len(diagnostics)
	diagnostics = append(diagnostics, Diagnostic{
		Source:         parseTrimmedSource,
		Severity:       parseSeverity,
		Classification: parseClassification,
		Code:           parseDetails.Code,
		Docs:           parseDetails.Docs,
		Remediation:    parseDetails.Remediation,
		Recoverable:    parseDetails.Recoverable,
		TopFrame:       parseTrimmedTopFrame,
		Consequence:    parseTrimmedConsequence,
		Message:        parseTrimmedMessage,
		Count:          1,
		Path:           parseTrimmedPath,
		ComponentStack: append([]string(nil), parseComponentStack...),
		Fields:         parseFields,
	})
	trimDiagnosticsLocked(maxDiagnosticEntries)

	reportLogWithKnownDetails(parseTrimmedSource, logLevelForSeverity(parseSeverity), parseClassification, parseDetails, parseTrimmedMessage, "", parseFields, parseTrimmedTopFrame, parseTrimmedConsequence)
}

// trimDiagnosticsLocked bounds the process-wide diagnostic ring. When the
// ring overflows it keeps only the newest half of the limit: dropping one
// entry at a time would rebuild the slice and the dedup index on EVERY
// report once full, turning a diagnostic storm (e.g. per-node hydration
// mismatches) quadratic — measured at 478ms for 2000 warnings. Halving
// amortizes the rebuild to O(1) per report; the ring holds between limit/2
// and limit entries.
func trimDiagnosticsLocked(parseLimit int) {
	if parseLimit <= 0 || len(diagnostics) <= parseLimit {
		return
	}
	parseKeep := parseLimit / 2
	if parseKeep < 1 {
		parseKeep = 1
	}
	diagnostics = append([]Diagnostic(nil), diagnostics[len(diagnostics)-parseKeep:]...)
	diagnosticIndex = make(map[diagnosticDedupKey]int, len(diagnostics))
	for parseIndex, parseDiagnostic := range diagnostics {
		parseKey := diagnosticDedupKey{
			severity:    parseDiagnostic.Severity,
			source:      parseDiagnostic.Source,
			message:     parseDiagnostic.Message,
			path:        parseDiagnostic.Path,
			stack:       strings.Join(parseDiagnostic.ComponentStack, " > "),
			topFrame:    strings.TrimSpace(parseDiagnostic.TopFrame),
			consequence: strings.TrimSpace(parseDiagnostic.Consequence),
			fields:      diagnosticFieldsKey(parseDiagnostic.Fields),
		}
		diagnosticIndex[parseKey] = parseIndex
	}
}

// GetDiagnostics returns a copy of the current diagnostic list. The entries
// are value copies; their ComponentStack slices and Fields maps are shared
// with the store, which never mutates them after insert — treat them as
// read-only. (Deep-cloning them per read dominated the cost of devtools
// inspection snapshots.)
func GetDiagnostics() []Diagnostic {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	parseClone := make([]Diagnostic, len(diagnostics))
	copy(parseClone, diagnostics)
	return parseClone
}

// ReportLog records one structured framework log entry.
func ReportLog(parseDomain string, parseLevel LogLevel, parseMessage string) {
	ReportLogWithFields(parseDomain, parseLevel, DiagnosticInformational, parseMessage, "", nil)
}

// ReportLogWithFields records one structured framework log entry with optional
// correlation id and fields.
func ReportLogWithFields(parseDomain string, parseLevel LogLevel, parseClassification DiagnosticClassification, parseMessage string, parseCorrelationID string, parseFields map[string]string) {
	reportLogWithFieldsDetails(parseDomain, parseLevel, parseClassification, parseMessage, parseCorrelationID, parseFields, "", "")
}

// reportLogWithFieldsDetails is a core package helper.
func reportLogWithFieldsDetails(parseDomain string, parseLevel LogLevel, parseClassification DiagnosticClassification, parseMessage string, parseCorrelationID string, parseFields map[string]string, parseTopFrame string, parseConsequence string) {
	parseTrimmedDomain := strings.TrimSpace(parseDomain)
	if parseTrimmedDomain == "" {
		parseTrimmedDomain = "runtime"
	}
	parseTrimmedMessage := strings.TrimSpace(parseMessage)
	if parseTrimmedMessage == "" {
		return
	}
	if parseClassification == "" {
		parseClassification = DiagnosticInformational
	}
	if parseLevel == "" {
		parseLevel = LogInfo
	}
	parseDetails := diagnosticMetadata(parseTrimmedDomain, logSeverity(parseLevel), parseClassification, parseTrimmedMessage)
	reportLogWithKnownDetails(parseTrimmedDomain, parseLevel, parseClassification, parseDetails, parseTrimmedMessage, parseCorrelationID, parseFields, parseTopFrame, parseConsequence)
}

// reportLogWithKnownDetails records one structured log entry whose diagnostic
// metadata was already computed by the caller (the diagnostic-report path
// derives it once for both the diagnostic entry and its log entry).
func reportLogWithKnownDetails(parseTrimmedDomain string, parseLevel LogLevel, parseClassification DiagnosticClassification, parseDetails diagnosticDetails, parseTrimmedMessage string, parseCorrelationID string, parseFields map[string]string, parseTopFrame string, parseConsequence string) {
	if parseTrimmedMessage == "" {
		return
	}
	if parseClassification == "" {
		parseClassification = DiagnosticInformational
	}
	if parseLevel == "" {
		parseLevel = LogInfo
	}

	parseEntry := LogEntry{
		Domain:         parseTrimmedDomain,
		Level:          parseLevel,
		Classification: parseClassification,
		Code:           parseDetails.Code,
		Docs:           parseDetails.Docs,
		Remediation:    parseDetails.Remediation,
		Recoverable:    parseDetails.Recoverable,
		TopFrame:       strings.TrimSpace(parseTopFrame),
		Consequence:    strings.TrimSpace(parseConsequence),
		Message:        parseTrimmedMessage,
		Timestamp:      time.Now().UTC().Format(timeFormatRFC3339Milli),
		CorrelationID:  strings.TrimSpace(parseCorrelationID),
		Fields:         cloneLogFields(parseFields),
	}

	logsMu.Lock()
	defer logsMu.Unlock()
	if len(logBuffer) < maxLogEntries {
		logBuffer = append(logBuffer, parseEntry)
		return
	}
	logBuffer[logHead] = parseEntry
	logHead++
	if logHead == len(logBuffer) {
		logHead = 0
	}
}

// GetLogs returns a copy of the current in-memory log buffer, oldest first.
// The entries are value copies; their Fields maps are shared with the store,
// which never mutates them after insert — treat them as read-only.
func GetLogs() []LogEntry {
	logsMu.Lock()
	defer logsMu.Unlock()
	parseClone := make([]LogEntry, 0, len(logBuffer))
	parseClone = append(parseClone, logBuffer[logHead:]...)
	parseClone = append(parseClone, logBuffer[:logHead]...)
	return parseClone
}

// ClearDiagnostics removes all recorded diagnostics.
func ClearDiagnostics() {
	diagnosticsMu.Lock()
	defer diagnosticsMu.Unlock()
	diagnosticIndex = map[diagnosticDedupKey]int{}
	diagnostics = nil
}

// ClearLogs removes all buffered framework logs.
func ClearLogs() {
	logsMu.Lock()
	defer logsMu.Unlock()
	logBuffer = nil
	logHead = 0
}

// Inspect captures a snapshot of the current runtime tree, profiling state, and diagnostics.
func (parseRt *Runtime) Inspect() InspectionSnapshot {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	parseSnapshot := InspectionSnapshot{
		Diagnostics: GetDiagnostics(),
		Logs:        GetLogs(),
	}
	if parseRt == nil {
		return parseSnapshot
	}

	// Event Fields maps are cloned at insert (RecordProfilingEvent) and never
	// mutated afterwards, so the snapshot shares them read-only.
	parseEvents := make([]ProfilingEvent, len(parseRt.profiling.events))
	copy(parseEvents, parseRt.profiling.events)
	// Timestamps are formatted here, for a reader, rather than at record time on
	// the render thread. See materializeProfilingTimestamps.
	parseEvents = materializeProfilingTimestamps(parseEvents)
	var parseRoot *FiberSnapshot
	var parseStats InspectionStats
	if parseRt.currentRoot != nil {
		parseRoot, parseStats = inspectFiberTree(parseRt.currentRoot)
	}
	parseStartedAt := ""
	if !parseRt.profiling.startupStartedAt.IsZero() {
		parseStartedAt = parseRt.profiling.startupStartedAt.UTC().Format(timeFormatRFC3339Milli)
	}
	parseSnapshot.Root = parseRoot
	parseSnapshot.Stats = parseStats
	parseSnapshot.Profiling = ProfilingSnapshot{
		RenderCalls:                      parseRt.profiling.renderCalls,
		ScheduledRootUpdates:             parseRt.profiling.scheduledRootUpdates,
		ScheduledFiberMarks:              parseRt.profiling.scheduledFiberMarks,
		ScheduledGranularMarks:           parseRt.profiling.scheduledGranularMarks,
		WorkLoopPasses:                   parseRt.profiling.workLoopPasses,
		ProcessedUnits:                   parseRt.profiling.processedUnits,
		CommitCount:                      parseRt.profiling.commitCount,
		FineGrainedCommits:               parseRt.profiling.fineGrainedCommits,
		FineGrainedDescendantHostCommits: parseRt.profiling.fineGrainedDescendantHostCommits,
		FineGrainedDescendantTextCommits: parseRt.profiling.fineGrainedDescendantTextCommits,
		EffectExecutions:                 parseRt.profiling.effectExecutions,
		CleanupExecutions:                parseRt.profiling.cleanupExecutions,
		LastRenderDurationNs:             parseRt.profiling.lastRenderDurationNs,
		LastCommitDurationNs:             parseRt.profiling.lastCommitDurationNs,
		LastEffectDurationNs:             parseRt.profiling.lastEffectDurationNs,
		LastCleanupDurationNs:            parseRt.profiling.lastCleanupDurationNs,
		PhaseTotals: ProfilingPhaseTotalsSnapshot{
			RenderDurationNs:  parseRt.profiling.totalRenderDurationNs,
			DiffDurationNs:    parseRt.profiling.totalDiffDurationNs,
			CommitDurationNs:  parseRt.profiling.totalCommitDurationNs,
			EffectDurationNs:  parseRt.profiling.totalEffectDurationNs,
			CleanupDurationNs: parseRt.profiling.totalCleanupDurationNs,
		},
		ComponentRenders: collectComponentRenderTraces(parseRt.profiling.componentRenders, 30),
		RecentEvents:     parseEvents,
		FlamegraphFrames: collectFlamegraphFrames(parseRoot, 256),
		Startup: StartupProfilingSnapshot{
			Mode:                       parseRt.profiling.startupMode,
			StartedAt:                  parseStartedAt,
			BootstrapReadDurationNs:    parseRt.profiling.bootstrapReadDurationNs,
			WASMTransferBytes:          parseRt.profiling.startupWASMTransferBytes,
			WASMDecodedBytes:           parseRt.profiling.startupWASMDecodedBytes,
			BootstrapDecodedBytes:      parseRt.profiling.startupBootstrapDecodedBytes,
			CacheWarmupDurationNs:      parseRt.profiling.startupCacheWarmupDurationNs,
			ServiceWorkerOverheadNs:    parseRt.profiling.startupServiceWorkerOverheadNs,
			InitialRouteDataBytes:      parseRt.profiling.startupInitialRouteDataBytes,
			HydrationDurationNs:        parseRt.profiling.hydrationDurationNs,
			StartupCommitDurationNs:    parseRt.profiling.startupCommitDurationNs,
			FirstInteractionDurationNs: parseRt.profiling.firstInteractionDurationNs,
			FirstInteractionCaptured:   parseRt.profiling.firstInteractionCaptured,
			FirstInteractionEvent:      parseRt.profiling.firstInteractionEvent,
			RouteBudgets:               buildRouteStartupBudgets(parseRt.profiling.routeStartupBudgets, 12),
		},
		HotBranches: collectHotBranches(parseRoot, 5),
	}
	parseSnapshot.Hydration = inspectHydrationDebugSnapshot(parseRt.lastHydrationMetrics, parseSnapshot.Diagnostics)
	return parseSnapshot
}

const timeFormatRFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

// classifyDiagnostic is a core package helper.
func classifyDiagnostic(parseSource string, parseSeverity DiagnosticSeverity, parseMessage string) DiagnosticClassification {
	switch parseSeverity {
	case DiagnosticInfo:
		return DiagnosticInformational
	case DiagnosticError:
		return DiagnosticCorrectness
	case DiagnosticWarning:
		parseTrimmed := strings.ToLower(strings.TrimSpace(parseMessage))
		if strings.Contains(parseTrimmed, "slow ") {
			return DiagnosticPerformance
		}
		if strings.Contains(parseTrimmed, "fell back") ||
			strings.Contains(parseTrimmed, "recovered") ||
			strings.Contains(parseTrimmed, "ignoring ") ||
			strings.Contains(parseTrimmed, "discarded unexpected") ||
			(strings.Contains(parseTrimmed, "redirect loop") && parseSource == "router") {
			return DiagnosticUnsupportedRecover
		}
		return DiagnosticCorrectness
	default:
		return DiagnosticInformational
	}
}

// diagnosticContextFields is a core package helper.
func diagnosticContextFields(parsePath string, parseComponentStack []string, parseTopFrame string, parseConsequence string, parseExtraFields map[string]string) map[string]string {
	parseFields := cloneLogFields(parseExtraFields)
	if len(parseFields) == 0 {
		parseFields = map[string]string{}
	}
	if strings.TrimSpace(parsePath) != "" {
		parseFields["path"] = strings.TrimSpace(parsePath)
	}
	if len(parseComponentStack) > 0 {
		parseFields["component_stack"] = strings.Join(parseComponentStack, " > ")
	}
	if strings.TrimSpace(parseTopFrame) != "" {
		parseFields["top_frame"] = strings.TrimSpace(parseTopFrame)
	}
	if strings.TrimSpace(parseConsequence) != "" {
		parseFields["runtime"] = strings.TrimSpace(parseConsequence)
	}
	if len(parseFields) == 0 {
		return nil
	}
	return parseFields
}

// diagnosticFieldsKey is a core package helper.
func diagnosticFieldsKey(parseFields map[string]string) string {
	if len(parseFields) == 0 {
		return ""
	}
	parseKeys := make([]string, 0, len(parseFields))
	for parseKey := range parseFields {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+"="+parseFields[parseKey2])
	}
	return strings.Join(parseParts, "|")
}

// logLevelForSeverity is a core package helper.
func logLevelForSeverity(parseSeverity DiagnosticSeverity) LogLevel {
	switch parseSeverity {
	case DiagnosticError:
		return LogError
	case DiagnosticWarning:
		return LogWarn
	default:
		return LogInfo
	}
}

// cloneLogFields is a core package helper.
func cloneLogFields(parseFields map[string]string) map[string]string {
	if len(parseFields) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseFields))
	maps.Copy(parseClone, parseFields)
	return parseClone
}
